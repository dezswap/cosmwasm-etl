package dex

import (
	"errors"
	"fmt"
	"strings"

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

	// to avoid skipping validation error
	if (localSynced % uint64(app.validationInterval)) == 0 {
		poolInfos, err := app.GetPoolInfos((localSynced))
		if err != nil {
			return fmt.Errorf("app.Run: %w", err)
		}
		if err := app.validate(0, (localSynced), poolInfos); err != nil {
			return fmt.Errorf("app.Run: %w", err)
		}
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

		if (cur % uint64(app.validationInterval)) == 0 {
			if len(poolInfos) == 0 {
				poolInfos, err = app.GetPoolInfos(cur)
				if err != nil {
					return fmt.Errorf("app.Run: %w", err)
				}
			}
			if err := app.validate(0, cur, poolInfos); err != nil {
				return fmt.Errorf("app.Run: %w", err)
			}
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

func (mixin *DexMixin) RemoveDuplicatedTxs(pairTxs []*ParsedTx, transferTxs []*ParsedTx) []ParsedTx {
	txs := []ParsedTx{}
	for idx, tx := range transferTxs {
		duplicated := false
		for i := 0; i < len(pairTxs) && !duplicated; i++ {
			duplicated = mixin.isDuplicatedTx(pairTxs[i], tx)
		}
		if !duplicated {
			txs = append(txs, *transferTxs[idx])
		}
	}
	return txs
}

// TODO: more specific logic
func (mixin *DexMixin) isDuplicatedTx(ptx *ParsedTx, transfer *ParsedTx) bool {
	isSameAssetAmount := func(a1, a2 Asset) bool {
		return a1.Addr == a2.Addr && a1.Amount == a2.Amount
	}
	return (ptx.ContractAddr == transfer.ContractAddr || ptx.ContractAddr == transfer.Sender) && (isSameAssetAmount(ptx.Assets[0], transfer.Assets[0]) || isSameAssetAmount(ptx.Assets[1], transfer.Assets[1]))
}
