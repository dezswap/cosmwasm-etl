package dex

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

// ErrNoNewHeight is returned when the remote node height has not advanced for
// sameHeightTolerance consecutive checks. Callers should treat this as a
// transient "wait for next block" condition, not a hard error.
var ErrNoNewHeight = errors.New("no new height")

type PairParsers struct {
	CreatePairParser parser.Parser[ParsedTx]
	PairActionParser parser.Parser[ParsedTx]
	InitialProvide   parser.Parser[ParsedTx]
	WasmTransfer     parser.Parser[ParsedTx]
	TaxPaymentParser parser.Parser[ParsedTx]
	Transfer         parser.Parser[ParsedTx]
	BurnParser       parser.Parser[ParsedTx]
}

// common mixin logic for all dex apps
type dexApp struct {
	TargetApp
	SourceDataStore
	Repo
	chainId string
	logger  logging.Logger

	poolSnapshotInterval uint
	validationInterval   uint

	sameHeightTolerance uint
	lastSrcHeight       uint64
	sameHeightCount     uint

	validationWorkerOnce         sync.Once
	validationMu                 sync.Mutex
	validationSignal             chan struct{}
	latestValidationSyncedHeight uint64
}

type DexMixin struct{}

var _ parser.ParserApp[ParsedTx] = &dexApp{}
var _ DexParserApp = &dexApp{}

func NewDexApp(app TargetApp, srcStore SourceDataStore, repo Repo, logger logging.Logger, c configs.ParserDexConfig) parser.ParserApp[ParsedTx] {
	return &dexApp{
		TargetApp:            app,
		SourceDataStore:      srcStore,
		Repo:                 repo,
		logger:               logger,
		chainId:              c.ChainId,
		sameHeightTolerance:  c.SameHeightTolerance,
		poolSnapshotInterval: c.PoolSnapshotInterval,
		validationInterval:   c.ValidationInterval,
		validationSignal:     make(chan struct{}, 1),
	}
}

func (app *dexApp) Run() error {
	tokenExceptions, err := app.GetTokenExceptions()
	if err != nil {
		return fmt.Errorf("app.Run: %w", err)
	}

	localSynced, err := app.GetSyncedHeight()
	if err != nil {
		return fmt.Errorf("app.Run: %w", err)
	}

	srcHeight, err := app.GetSourceSyncedHeight()
	if err != nil {
		return fmt.Errorf("app.Run: %w", err)
	}

	if srcHeight < localSynced {
		return errors.New("remote height is less than local synced height")
	}

	if err := app.checkRemoteHeight(srcHeight); err != nil {
		return fmt.Errorf("app.Run: %w", err)
	}

	app.signalValidation(localSynced)

	// to avoid skipping validation error
	if app.isValidationHeight(localSynced) {
		app.triggerValidation(localSynced)
	}

	app.logger.Infof("current synced height: %d, remote node height: %d", localSynced, srcHeight)
	for cur := localSynced + 1; cur <= srcHeight; cur++ {
		txs, err := app.GetSourceTxs(cur)
		if err != nil {
			if strings.Contains(err.Error(), fmt.Sprintf("greater than the current height %d", srcHeight-1)) {
				app.logger.Infof("remote node is indexing tx_results, skip")
				return nil
			}
			return fmt.Errorf("app.Run: %w", err)
		}

		if err := app.UpdateParsers(tokenExceptions, cur); err != nil {
			return fmt.Errorf("app.Run: %w", err)
		}

		parsedTxs := []ParsedTx{}
		for _, tx := range txs {
			txs, err := app.ParseTxs(tx, cur)
			if err != nil {
				return fmt.Errorf("app.Run: %w", err)
			}
			parsedTxs = append(parsedTxs, txs...)
		}

		poolInfos := []PoolInfo{}
		if (cur % uint64(app.poolSnapshotInterval)) == 0 {
			poolInfos, err = app.GetPoolInfos(cur)
			if err != nil {
				return fmt.Errorf("app.Run: %w", err)
			}
		}

		if err := app.insert(cur-1, cur, parsedTxs, poolInfos); err != nil {
			return fmt.Errorf("app.Run: %w", err)
		}

		if app.isValidationHeight(cur) {
			app.triggerValidation(cur)
		}
	}
	app.lastSrcHeight = srcHeight

	return nil
}

// insert implements parser
func (app *dexApp) insert(srcHeight uint64, targetHeight uint64, txs []ParsedTx, pools []PoolInfo) error {
	pairDtos := []Pair{}
	for _, tx := range txs {
		if tx.Type == CreatePair {
			pairDto := Pair{
				ContractAddr: tx.ContractAddr,
				Assets:       []string{tx.Assets[0].Addr, tx.Assets[1].Addr},
				LpAddr:       tx.LpAddr,
			}
			pairDtos = append(pairDtos, pairDto)
		}
	}

	err := app.Insert(srcHeight, targetHeight, txs, pools, pairDtos)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	return nil
}

// checkRemoteHeight implements Dex
func (app *dexApp) checkRemoteHeight(srcHeight uint64) error {
	if srcHeight == app.lastSrcHeight {
		app.sameHeightCount++
		if app.sameHeightCount > app.sameHeightTolerance {
			return fmt.Errorf("remote node height(%d) remains the same for %d consecutive times: %w", srcHeight, app.sameHeightCount, ErrNoNewHeight)
		}
	} else {
		app.sameHeightCount = 0
	}
	return nil
}

// isValidationHeight reports whether the given height is a configured
// validation interval boundary.
func (app *dexApp) isValidationHeight(height uint64) bool {
	if app.validationInterval == 0 || height == 0 {
		return false
	}
	return height%uint64(app.validationInterval) == 0
}

// triggerValidation persists a validation cursor when none exists, then wakes
// the async validator for the synced height that just reached an interval.
func (app *dexApp) triggerValidation(height uint64) {
	validationHeight, err := app.GetValidationHeight()
	if err != nil {
		app.logger.Errorf("validator: failed to load validation height: %s", err)
		return
	}
	if validationHeight == 0 {
		if err := app.SetValidationHeight(height); err != nil {
			app.logger.Errorf("validator: failed to persist validation height %d: %s", height, err)
			return
		}
	}
	app.signalValidation(height)
}

// signalValidation starts the single validation worker and records the latest
// synced height it should validate up to. Multiple calls coalesce into one wakeup.
func (app *dexApp) signalValidation(syncedHeight uint64) {
	if app.validationInterval == 0 {
		return
	}

	app.validationWorkerOnce.Do(func() {
		if app.validationSignal == nil {
			app.validationSignal = make(chan struct{}, 1)
		}
		go app.runValidationWorker()
	})

	app.validationMu.Lock()
	if syncedHeight > app.latestValidationSyncedHeight {
		app.latestValidationSyncedHeight = syncedHeight
	}
	app.validationMu.Unlock()

	select {
	case app.validationSignal <- struct{}{}:
	default:
	}
}

// runValidationWorker consumes validation wakeups and drains all persisted
// validation work up to the latest synced height observed while it is running.
func (app *dexApp) runValidationWorker() {
	for range app.validationSignal {
		for {
			app.validationMu.Lock()
			syncedHeight := app.latestValidationSyncedHeight
			app.latestValidationSyncedHeight = 0
			app.validationMu.Unlock()

			if syncedHeight == 0 {
				break
			}

			app.processPendingValidations(syncedHeight)

			app.validationMu.Lock()
			hasPendingSignal := app.latestValidationSyncedHeight > 0
			app.validationMu.Unlock()
			if !hasPendingSignal {
				break
			}
		}
	}
}

// processPendingValidations validates the persisted validation cursor up to
// syncedHeight. Successful validation advances the cursor by validationInterval
// before the next validation, so restarts continue from the next unresolved
// validation height.
func (app *dexApp) processPendingValidations(syncedHeight uint64) {
	if app.validationInterval == 0 {
		return
	}

	for {
		height, err := app.GetValidationHeight()
		if err != nil {
			app.logger.Errorf("validator: failed to load validation height: %s", err)
			return
		}
		if height == 0 || height > syncedHeight {
			return
		}
		if !app.validateAtHeight(height) {
			return
		}
		next := height + uint64(app.validationInterval)
		if next <= height {
			if err := app.ClearValidationHeight(); err != nil {
				app.logger.Errorf("validator: failed to clear validation height after overflow: %s", err)
			}
			return
		}
		if err := app.SetValidationHeight(next); err != nil {
			app.logger.Errorf("validator: failed to advance validation height from %d to %d: %s", height, next, err)
			return
		}
	}
}

// validateAtHeight compares source pool info with parsed DB state for one
// validation height and returns false when validation should be retried later.
func (app *dexApp) validateAtHeight(height uint64) bool {
	poolInfos, err := app.GetPoolInfos(height)
	if err != nil {
		app.logger.Errorf("validator: GetPoolInfos at height %d: %s", height, err)
		return false
	}
	if err := app.validate(0, height, poolInfos); err != nil {
		app.logger.Errorf("validator: pool mismatch at height %d: %s", height, err)
		return false
	}
	return true
}

// validate with `expected` from the node, compare database updates, as `actual`
func (app *dexApp) validate(from, to uint64, expected []PoolInfo) error {
	if len(expected) == 0 {
		app.logger.Infof("No pool info found at height %d", to)
		return nil
	}

	// TODO: snapshot for expected pools
	// e.g.) expected pools can be difference between pool of height 1000 and 900
	actual, err := app.ParsedPoolsInfo(from, to)
	if err != nil {
		return fmt.Errorf("dexApp.validate: %w", err)
	}

	expectedPool := make(map[string]PoolInfo)
	for _, pool := range expected {
		expectedPool[pool.ContractAddr] = pool
	}

	exceptions, err := app.ValidationExceptionList()
	if err != nil {
		return fmt.Errorf("dexApp.validate: %w", err)
	}
	exceptionMap := make(map[string]bool)
	for _, addr := range exceptions {
		delete(expectedPool, addr)
		exceptionMap[addr] = true
	}

	for _, pool := range actual {
		if _, ok := exceptionMap[pool.ContractAddr]; ok {
			continue
		}
		exp, ok := expectedPool[pool.ContractAddr]
		if !ok {
			return fmt.Errorf("unexpected pool(%s) found", pool.ContractAddr)
		}
		if err := app.comparePair(pool, exp); err != nil {
			return err
		}

		delete(expectedPool, pool.ContractAddr)
	}
	if len(expectedPool) > 0 {
		addrs := []string{}
		for _, pool := range expectedPool {
			addrs = append(addrs, pool.ContractAddr)
		}
		return fmt.Errorf("expected pools(%s) not found", addrs)
	}
	return nil
}

func (app *dexApp) comparePair(actual PoolInfo, expected PoolInfo) error {
	var diffs []string

	for idx, expAsset := range expected.Assets {
		if expAsset.Amount != actual.Assets[idx].Amount {
			diffs = append(diffs, fmt.Sprintf(
				"pool(%s) asset(%s) amount mismatch: actual(%s), expected(%s)", actual.ContractAddr, expAsset.Addr, actual.Assets[idx].Amount, expAsset.Amount,
			))
		}
	}

	if expected.TotalShare != actual.TotalShare {
		diffs = append(diffs, fmt.Sprintf(
			"pool(%s) total share mismatch: actual(%s), expected(%s)", actual.ContractAddr, actual.TotalShare, expected.TotalShare,
		))
	}

	if len(diffs) == 0 {
		return nil
	}

	isValidationException := false
	for _, a := range actual.Assets {
		if app.IsValidationExceptionCandidate(a.Addr) {
			isValidationException = true
		}
	}

	if isValidationException {
		err := app.InsertPairValidationException(app.chainId, actual.ContractAddr)
		if err != nil {
			return err
		}

		return nil
	}

	return errors.New(strings.Join(diffs, "; "))
}

// CollectLpBurnTxs collects LP burn events and associates them with their pair contract.
// LP tokens can be burned directly (outside of withdraw_liquidity), so we need to track
// these burns separately and subtract the burned amount to keep pool calculations accurate.
func CollectLpBurnTxs(burnTxs []*ParsedTx, lpPairAddrs map[string]string) []ParsedTx {
	lpBurnTxs := []ParsedTx{}
	for _, t := range burnTxs {
		pairAddr, ok := lpPairAddrs[t.LpAddr]
		if !ok || t.Sender == pairAddr { // filter out withdraw lp burn
			continue
		}

		t.ContractAddr = pairAddr
		lpBurnTxs = append(lpBurnTxs, *t)
	}
	return lpBurnTxs
}

func (mixin *DexMixin) HasProvide(pairTxs []*ParsedTx) bool {
	for _, tx := range pairTxs {
		if tx.Type == Provide {
			return true
		}
	}
	return false
}

type transferPopEntry struct {
	pairAddr  string
	assetAddr string
	amount    string
}

// RemoveDuplicatedTxs removes transfer events already captured by pair action events.
// By building popList from pairTxs (one per asset) and matching each transfer 1:1,
// it prevents a single pair action from consuming more transfers than it produced.
func (mixin *DexMixin) RemoveDuplicatedTxs(pairTxs []*ParsedTx, transferTxs []*ParsedTx) []ParsedTx {
	popList := []transferPopEntry{}
	for _, ptx := range pairTxs {
		for _, asset := range ptx.Assets {
			if asset.Amount != "" && asset.Amount != "0" {
				popList = append(popList, transferPopEntry{ptx.ContractAddr, asset.Addr, asset.Amount})
			}
		}
	}

	consumed := make([]bool, len(transferTxs))
	for i, tx := range transferTxs {
		for j, entry := range popList {
			if mixin.matchesPairTransferEntry(entry, tx) {
				consumed[i] = true
				popList = append(popList[:j], popList[j+1:]...)
				break
			}
		}
	}

	txs := []ParsedTx{}
	for i, tx := range transferTxs {
		if !consumed[i] {
			txs = append(txs, *tx)
		}
	}
	return txs
}

// matchesPairTransferEntry reports whether tx corresponds to a transfer entry.
func (mixin *DexMixin) matchesPairTransferEntry(entry transferPopEntry, transferTx *ParsedTx) bool {
	isPairSendTx := entry.pairAddr == transferTx.Sender
	isPairReceiveTx := entry.pairAddr == transferTx.ContractAddr
	isEntryOutflow := strings.HasPrefix(entry.amount, "-")

	if (!isEntryOutflow && !isPairReceiveTx) || (isEntryOutflow && !isPairSendTx) {
		return false
	}

	for _, asset := range transferTx.Assets {
		if asset.Addr != entry.assetAddr || asset.Amount == "" {
			continue
		}

		// For pair->user transfers (transferTx.Sender == entry.pairAddr), CW20 token contracts
		// may charge fees/taxes causing the received amount to differ from the pair's recorded amount.
		// e.g. columbus-5 14829F480097AF38ECED3079ACD06BAA0AC0583E6BFCC85375A018617B13BBCB
		// Native tokens (e.g., axpla, uusd, uluna) always transfer exact amounts via the bank module,
		// so require an exact match even for pair->user transfers.
		// For user->pair transfers, the asset amount must match exactly (asset.Amount == entry.amount),
		// to avoid consuming unrelated same-asset transfers within the same tx.
		if (isPairSendTx && isCw20TokenAddress(entry.assetAddr)) || asset.Amount == entry.amount {
			return true
		}
	}

	return false
}
