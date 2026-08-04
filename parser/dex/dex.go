package dex

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/sirupsen/logrus"
)

// ErrNoNewHeight is returned when the remote node height has not advanced for
// sameHeightTolerance consecutive checks. Callers should treat this as a
// transient "wait for next block" condition, not a hard error.
var ErrNoNewHeight = errors.New("no new height")

const (
	validationMismatchUnexpectedPool      = "unexpected_pool"
	validationMismatchExpectedPoolMissing = "expected_pool_not_found"
	validationMismatchAssetAmount         = "asset_amount_mismatch"
	validationMismatchTotalShare          = "total_share_mismatch"
)

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

	quarantineRetryMode   configs.QuarantineRetryMode
	startupRetryAttempted bool
}

type DexMixin struct{}

var _ parser.ParserApp[ParsedTx] = &dexApp{}
var _ DexParserApp = &dexApp{}

func NewDexApp(app TargetApp, srcStore SourceDataStore, repo Repo, logger logging.Logger, c configs.ParserDexConfig) parser.ParserApp[ParsedTx] {
	retryMode := c.QuarantineRetryMode
	if retryMode == "" {
		retryMode = configs.QuarantineRetryDisabled
	}
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
		quarantineRetryMode:  retryMode,
	}
}

func (app *dexApp) Run() error {
	runStartedAt := time.Now()
	tokenExceptions, err := app.GetTokenExceptions()
	if err != nil {
		return fmt.Errorf("app.Run: %w", err)
	}
	if app.shouldRetryQuarantine() {
		if err := app.retryPendingQuarantines(tokenExceptions); err != nil {
			return fmt.Errorf("app.Run retry quarantine: %w", err)
		}
		if app.quarantineRetryMode == configs.QuarantineRetryStartup {
			app.startupRetryAttempted = true
		}
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

	app.logger.WithFields(logrus.Fields{
		"event":         "parser.sync_status",
		"operation":     "parser.run",
		"chain_id":      app.chainId,
		"local_height":  localSynced,
		"source_height": srcHeight,
		"lag":           srcHeight - localSynced,
	}).Info("parser sync status")

	processedHeightCount := 0
	parsedTxCount := 0
	quarantineCount := 0
	poolSnapshotCount := 0

	for cur := localSynced + 1; cur <= srcHeight; cur++ {
		txs, err := app.GetSourceTxs(cur)
		if err != nil {
			if strings.Contains(err.Error(), fmt.Sprintf("greater than the current height %d", srcHeight-1)) {
				app.logger.WithFields(logrus.Fields{
					"event":         "parser.source_indexing",
					"operation":     "get_source_txs",
					"chain_id":      app.chainId,
					"height":        cur,
					"source_height": srcHeight,
				}).Info("remote node is indexing tx_results")
				return nil
			}
			return fmt.Errorf("app.Run: %w", err)
		}

		if err := app.UpdateParsers(tokenExceptions, cur); err != nil {
			return fmt.Errorf("app.Run: %w", err)
		}

		parsedTxs := []ParsedTx{}
		parseQuarantines := []ParseQuarantine{}
		for _, tx := range txs {
			txs, err := app.ParseTxs(tx, cur)
			if err != nil {
				var partial *PartialParseQuarantineError
				if errors.As(err, &partial) {
					parseQuarantines = append(parseQuarantines, partial.Quarantine)
					app.logger.WithFields(logrus.Fields{
						"event":             "parse_quarantine.partial_created",
						"operation":         "parse_txs",
						"chain_id":          app.chainId,
						"height":            cur,
						"tx_hash":           tx.Hash,
						"stage":             partial.Quarantine.Stage,
						"contract":          partial.Quarantine.Contract,
						"action":            partial.Quarantine.Action,
						"quarantine_status": QuarantineStatusPending,
						"err":               logging.NewErrorField(err),
					}).Warn("partial parse quarantine created")
					parsedTxs = append(parsedTxs, partial.ParsedTxs...)
					continue
				}
				var ambiguity *eventlog.AmbiguousEventError
				if errors.As(err, &ambiguity) && !RawTxContainsCreatePair(tx) {
					// Raw transactions remain available, so parser progress does not prevent deterministic replay.
					parseQuarantines = append(parseQuarantines, ParseQuarantine{
						Height:   cur,
						Hash:     tx.Hash,
						Stage:    parseStage(err),
						Contract: ambiguity.Contract,
						Action:   ambiguity.Action,
						Error:    err.Error(),
						RawTx:    tx,
					})
					app.logger.WithFields(logrus.Fields{
						"event":             "parse_quarantine.created",
						"operation":         "parse_txs",
						"chain_id":          app.chainId,
						"height":            cur,
						"tx_hash":           tx.Hash,
						"stage":             parseStage(err),
						"contract":          ambiguity.Contract,
						"action":            ambiguity.Action,
						"quarantine_status": QuarantineStatusPending,
						"err":               logging.NewErrorField(err),
					}).Warn("parse quarantine created")
					continue
				}
				return fmt.Errorf("app.Run: %w", err)
			}
			parsedTxs = append(parsedTxs, txs...)
		}

		poolInfos := []PoolInfo{}
		poolSnapshotSaved := false
		if (cur % uint64(app.poolSnapshotInterval)) == 0 {
			poolInfos, err = app.GetPoolInfos(cur)
			if err != nil {
				return fmt.Errorf("app.Run: %w", err)
			}
			poolSnapshotSaved = true
		}

		if err := app.insert(cur-1, cur, parsedTxs, poolInfos, parseQuarantines); err != nil {
			return fmt.Errorf("app.Run: %w", err)
		}
		processedHeightCount++
		parsedTxCount += len(parsedTxs)
		quarantineCount += len(parseQuarantines)
		if poolSnapshotSaved {
			poolSnapshotCount++
		}
		app.logger.WithFields(logrus.Fields{
			"event":               "parser.height_processed",
			"operation":           "parser.run",
			"chain_id":            app.chainId,
			"height":              cur,
			"parsed_tx_count":     len(parsedTxs),
			"quarantine_count":    len(parseQuarantines),
			"pool_snapshot_saved": poolSnapshotSaved,
		}).Debug("parser height processed")

		if app.isValidationHeight(cur) {
			app.triggerValidation(cur)
		}
	}
	app.lastSrcHeight = srcHeight
	if processedHeightCount > 0 {
		app.logger.WithFields(logrus.Fields{
			"event":                  "parser.run_summary",
			"operation":              "parser.run",
			"chain_id":               app.chainId,
			"from_height":            localSynced + 1,
			"to_height":              srcHeight,
			"synced_height":          srcHeight,
			"source_height":          srcHeight,
			"processed_height_count": processedHeightCount,
			"parsed_tx_count":        parsedTxCount,
			"quarantine_count":       quarantineCount,
			"pool_snapshot_count":    poolSnapshotCount,
			"duration_ms":            time.Since(runStartedAt).Milliseconds(),
		}).Info("parser run summary")
	}

	return nil
}

// shouldRetryQuarantine applies the configured retry mode to each parser run.
func (app *dexApp) shouldRetryQuarantine() bool {
	switch app.quarantineRetryMode {
	case configs.QuarantineRetryEveryRun:
		return true
	case configs.QuarantineRetryStartup:
		return !app.startupRetryAttempted
	default:
		return false
	}
}

// retryPendingQuarantines replays unresolved raw transactions and resolves only successful parses.
func (app *dexApp) retryPendingQuarantines(tokenExceptions map[string]bool) error {
	quarantines, err := app.PendingParseQuarantines()
	if err != nil {
		return err
	}

	for _, quarantine := range quarantines {
		if IsPartialQuarantineStage(quarantine.Stage) {
			continue
		}
		if err := app.UpdateParsers(tokenExceptions, quarantine.Height); err != nil {
			return fmt.Errorf("update parsers for quarantine id=%d: %w", quarantine.ID, err)
		}
		txs, err := app.ParseTxs(quarantine.RawTx, quarantine.Height)
		if err != nil {
			var ambiguity *eventlog.AmbiguousEventError
			if errors.As(err, &ambiguity) {
				continue
			}
			return fmt.Errorf("reparse quarantine id=%d tx_hash=%s: %w", quarantine.ID, quarantine.Hash, err)
		}
		if err := app.ResolveParseQuarantine(quarantine.ID, quarantine.Height, txs); err != nil {
			return err
		}
		app.logger.WithFields(logrus.Fields{
			"event":             "parse_quarantine.resolved",
			"operation":         "retry_parse_quarantine",
			"chain_id":          app.chainId,
			"height":            quarantine.Height,
			"tx_hash":           quarantine.Hash,
			"stage":             quarantine.Stage,
			"contract":          quarantine.Contract,
			"action":            quarantine.Action,
			"quarantine_status": QuarantineStatusResolved,
			"parsed_tx_count":   len(txs),
		}).Info("parse quarantine resolved")
	}
	return nil
}

// parseStage extracts the explicit ParseTxs wrapper stage for quarantine diagnostics.
func parseStage(err error) string {
	for _, candidate := range []struct {
		marker string
		stage  string
	}{
		{marker: "ParseTxs create_pair", stage: "create_pair"},
		{marker: "ParseTxs pair_action", stage: "pair_action"},
		{marker: "ParseTxs initial_provide", stage: "initial_provide"},
		{marker: "ParseTxs wasm_transfer", stage: "wasm_transfer"},
		{marker: "ParseTxs tax_payment", stage: "tax_payment"},
		{marker: "ParseTxs sort_transfer_attrs", stage: "sort_transfer_attrs"},
		{marker: "ParseTxs transfer", stage: "transfer"},
		{marker: "ParseTxs burn", stage: "burn"},
	} {
		if strings.Contains(err.Error(), candidate.marker) {
			return candidate.stage
		}
	}
	return "unknown"
}

// insert implements parser
func (app *dexApp) insert(srcHeight uint64, targetHeight uint64, txs []ParsedTx, pools []PoolInfo, quarantines []ParseQuarantine) error {
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

	err := app.Insert(srcHeight, targetHeight, txs, pools, pairDtos, quarantines)
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
		var validationErr *poolValidationError
		if errors.As(err, &validationErr) {
			for _, mismatch := range validationErr.Mismatches {
				app.logPoolValidationMismatch(height, mismatch)
			}
		} else {
			app.logger.WithFields(logrus.Fields{
				"event":     "parser.pool_validation_failed",
				"operation": "pool_validation",
				"chain_id":  app.chainId,
				"height":    height,
				"err":       logging.NewErrorField(err),
			}).Error("pool validation failed")
		}
		return false
	}
	return true
}

// logPoolValidationMismatch emits the agent-facing root log for one pool validation mismatch.
func (app *dexApp) logPoolValidationMismatch(height uint64, mismatch poolValidationMismatch) {
	app.logger.WithFields(logrus.Fields{
		"event":         "parser.pool_validation_failed",
		"operation":     "pool_validation",
		"chain_id":      app.chainId,
		"height":        height,
		"contract":      mismatch.Contract,
		"mismatch_type": mismatch.Type,
		"asset":         mismatch.Asset,
		"actual":        mismatch.Actual,
		"expected":      mismatch.Expected,
		"lookup_tables": []string{"parsed_tx", "pool_info", "parse_quarantine"},
	}).Error("pool validation failed")
}

// validate with `expected` from the node, compare database updates, as `actual`
func (app *dexApp) validate(from, to uint64, expected []PoolInfo) error {
	if len(expected) == 0 {
		app.logger.WithFields(logrus.Fields{
			"event":     "parser.pool_validation_skipped",
			"operation": "pool_validation",
			"chain_id":  app.chainId,
			"height":    to,
			"reason":    "empty_expected_pool_info",
		}).Info("no pool info found")
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

	validationErr := &poolValidationError{}
	for _, pool := range actual {
		if _, ok := exceptionMap[pool.ContractAddr]; ok {
			continue
		}
		exp, ok := expectedPool[pool.ContractAddr]
		if !ok {
			validationErr.Add(poolValidationMismatch{
				Type:     validationMismatchUnexpectedPool,
				Contract: pool.ContractAddr,
				Actual:   pool.TotalShare,
			})
			continue
		}
		if err := app.collectPairValidationMismatches(pool, exp, validationErr); err != nil {
			return err
		}

		delete(expectedPool, pool.ContractAddr)
	}
	for _, pool := range expectedPool {
		validationErr.Add(poolValidationMismatch{
			Type:     validationMismatchExpectedPoolMissing,
			Contract: pool.ContractAddr,
			Expected: pool.TotalShare,
		})
	}
	if validationErr.HasMismatches() {
		return validationErr
	}
	return nil
}

// poolValidationMismatch describes one concrete difference between source pool state and parsed DB state.
type poolValidationMismatch struct {
	Type     string
	Contract string
	Asset    string
	Actual   string
	Expected string
}

// poolValidationError aggregates every mismatch found during one validation run.
type poolValidationError struct {
	Mismatches []poolValidationMismatch
}

// Error summarizes the aggregated validation failure without hiding per-pool mismatch details.
func (e *poolValidationError) Error() string {
	if len(e.Mismatches) == 0 {
		return "pool validation failed"
	}
	return fmt.Sprintf("pool validation failed with %d mismatch(es)", len(e.Mismatches))
}

// Add records one mismatch while allowing validation to continue collecting the rest.
func (e *poolValidationError) Add(mismatch poolValidationMismatch) {
	e.Mismatches = append(e.Mismatches, mismatch)
}

// HasMismatches reports whether validation found any differences worth failing on.
func (e *poolValidationError) HasMismatches() bool {
	return len(e.Mismatches) > 0
}

// collectPairValidationMismatches compares one pair and appends every asset/share mismatch.
func (app *dexApp) collectPairValidationMismatches(actual PoolInfo, expected PoolInfo, validationErr *poolValidationError) error {
	var mismatches []poolValidationMismatch

	for idx, expAsset := range expected.Assets {
		actualAmount := ""
		if idx < len(actual.Assets) {
			actualAmount = actual.Assets[idx].Amount
		}
		if expAsset.Amount != actualAmount {
			mismatches = append(mismatches, poolValidationMismatch{
				Type:     validationMismatchAssetAmount,
				Contract: actual.ContractAddr,
				Asset:    expAsset.Addr,
				Actual:   actualAmount,
				Expected: expAsset.Amount,
			})
		}
	}

	if expected.TotalShare != actual.TotalShare {
		mismatches = append(mismatches, poolValidationMismatch{
			Type:     validationMismatchTotalShare,
			Contract: actual.ContractAddr,
			Actual:   actual.TotalShare,
			Expected: expected.TotalShare,
		})
	}

	if len(mismatches) == 0 {
		return nil
	}

	isValidationException := false
	for _, a := range actual.Assets {
		if app.isValidationExceptionCandidate(a.Addr) {
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

	for _, mismatch := range mismatches {
		validationErr.Add(mismatch)
	}
	return nil
}

// isValidationExceptionCandidate safely delegates exception detection to the target app.
func (app *dexApp) isValidationExceptionCandidate(contractAddress string) bool {
	if app.TargetApp == nil {
		return false
	}
	return app.IsValidationExceptionCandidate(contractAddress)
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
