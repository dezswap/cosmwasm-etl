package dex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type quarantineTargetApp struct {
	parse func(tx parser.RawTx, height uint64) ([]ParsedTx, error)
}

func (a *quarantineTargetApp) ParseTxs(tx parser.RawTx, height uint64) ([]ParsedTx, error) {
	return a.parse(tx, height)
}

func (*quarantineTargetApp) IsValidationExceptionCandidate(string) bool {
	return false
}

func (*quarantineTargetApp) UpdateParsers(map[string]bool, uint64) error {
	return nil
}

// insert implements parser
func Test_insert(t *testing.T) {
	const (
		chainId = "chainId"
		height  = uint64(100)
	)
	poolInfo := PoolInfo{ContractAddr: "ContractAddr", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}}

	tcs := []struct {
		txs       []ParsedTx
		poolInfos []PoolInfo
		errMsg    string
	}{
		{
			[]ParsedTx{createTx, swapTx, provideTx, withdrawTx, transferTx},
			[]PoolInfo{poolInfo},
			"",
		},
		{
			[]ParsedTx{swapTx, provideTx, withdrawTx, transferTx},
			[]PoolInfo{poolInfo},
			"return error",
		},
	}

	for _, tc := range tcs {
		repoMock := RepoMock{}
		pairDtos := []Pair{}
		for _, tx := range tc.txs {
			if tx.Type == CreatePair {
				pairDto := Pair{
					ContractAddr: tx.ContractAddr,
					Assets:       []string{tx.Assets[0].Addr, tx.Assets[1].Addr},
					LpAddr:       tx.LpAddr,
				}
				pairDtos = append(pairDtos, pairDto)
			}
		}
		var err error
		if tc.errMsg != "" {
			err = errors.New(tc.errMsg)
		}
		quarantines := []ParseQuarantine{}
		repoMock.On("Insert", height-1, height, tc.txs, tc.poolInfos, pairDtos, quarantines).Return(err)

		app := dexApp{Repo: &repoMock}
		err = app.insert(height-1, height, tc.txs, tc.poolInfos, quarantines)
		if tc.errMsg != "" {
			assert.Error(t, err, tc.errMsg)
			repoMock.AssertExpectations(t)
		} else {
			assert.NoError(t, err)
		}
	}
}
func Test_srcHeightCheck(t *testing.T) {
	app := &dexApp{
		sameHeightTolerance: 3,
		lastSrcHeight:       100,
		sameHeightCount:     3,
	}

	testCases := []struct {
		srcHeight   uint64
		expectError bool
	}{
		{srcHeight: 100, expectError: true},
		{srcHeight: 101, expectError: false},
		{srcHeight: 101, expectError: false},
		{srcHeight: 101, expectError: false},
		{srcHeight: 101, expectError: false},
		{srcHeight: 101, expectError: true},
	}

	for i, tc := range testCases {
		err := app.checkRemoteHeight(tc.srcHeight)
		if (err != nil) != tc.expectError {
			t.Errorf("Test case %d failed: srcHeight=%d, expectError=%v, got error=%v",
				i, tc.srcHeight, tc.expectError, err)
		} else {
			app.lastSrcHeight = tc.srcHeight
		}
	}
}

func Test_Run_QuarantinesAmbiguousTransactionAndAdvancesHeight(t *testing.T) {
	ambiguousTx := parser.RawTx{Hash: "ambiguous"}
	normalTx := parser.RawTx{Hash: "normal"}
	expectedTx := ParsedTx{
		Hash:         normalTx.Hash,
		Type:         Transfer,
		Sender:       "sender",
		ContractAddr: "pair",
		Assets:       [2]Asset{{Addr: "asset0", Amount: "1"}, {Addr: "asset1", Amount: "0"}},
	}

	target := &quarantineTargetApp{parse: func(tx parser.RawTx, _ uint64) ([]ParsedTx, error) {
		if tx.Hash == ambiguousTx.Hash {
			return nil, &eventlog.AmbiguousEventError{
				Contract: "token",
				Action:   "transfer",
				Key:      "amount",
				Values:   []string{"1", "2"},
			}
		}
		return []ParsedTx{expectedTx}, nil
	}}
	repo := &RepoMock{}
	srcStore := &RawStoreMock{}
	app := &dexApp{
		TargetApp:            target,
		Repo:                 repo,
		SourceDataStore:      srcStore,
		logger:               logging.Discard,
		poolSnapshotInterval: 100,
		sameHeightTolerance:  3,
		quarantineRetryMode:  configs.QuarantineRetryDisabled,
	}

	repo.On("GetTokenExceptions").Return(map[string]bool{}, nil)
	repo.On("GetSyncedHeight").Return(uint64(0), nil)
	srcStore.On("GetSourceSyncedHeight").Return(uint64(1), nil)
	srcStore.On("GetSourceTxs", uint64(1)).Return(parser.RawTxs{ambiguousTx, normalTx}, nil)
	expectedQuarantines := mock.MatchedBy(func(qs []ParseQuarantine) bool {
		if len(qs) != 1 {
			return false
		}
		q := qs[0]
		return q.Height == 1 &&
			q.Hash == ambiguousTx.Hash &&
			q.Stage == "unknown" &&
			q.Contract == "token" &&
			q.Action == "transfer"
	})
	repo.On("Insert", uint64(0), uint64(1), []ParsedTx{expectedTx}, []PoolInfo{}, []Pair{}, expectedQuarantines).Return(nil)

	require.NoError(t, app.Run())
	repo.AssertExpectations(t)
	srcStore.AssertExpectations(t)
}

func Test_Run_DoesNotQuarantineCreatePairTransaction(t *testing.T) {
	tx := parser.RawTx{
		Hash: "create-pair",
		LogResults: eventlog.LogResults{{
			Type: eventlog.WasmType,
			Attributes: eventlog.Attributes{
				{Key: "action", Value: string(CreatePair)},
			},
		}},
	}
	target := &quarantineTargetApp{parse: func(parser.RawTx, uint64) ([]ParsedTx, error) {
		return nil, &eventlog.AmbiguousEventError{Key: "pair", Values: []string{"a", "b"}}
	}}
	repo := &RepoMock{}
	srcStore := &RawStoreMock{}
	app := &dexApp{
		TargetApp:            target,
		Repo:                 repo,
		SourceDataStore:      srcStore,
		logger:               logging.Discard,
		poolSnapshotInterval: 100,
		sameHeightTolerance:  3,
		quarantineRetryMode:  configs.QuarantineRetryDisabled,
	}

	repo.On("GetTokenExceptions").Return(map[string]bool{}, nil)
	repo.On("GetSyncedHeight").Return(uint64(0), nil)
	srcStore.On("GetSourceSyncedHeight").Return(uint64(1), nil)
	srcStore.On("GetSourceTxs", uint64(1)).Return(parser.RawTxs{tx}, nil)

	err := app.Run()
	require.Error(t, err)
	repo.AssertNotCalled(t, "Insert", mock.Anything)
}

func Test_Run_UpsertsPartialQuarantineAndInsertsParsedTxs(t *testing.T) {
	rawTx := parser.RawTx{Hash: "partial"}
	parsedTx := ParsedTx{
		Hash:         rawTx.Hash,
		Type:         Transfer,
		Sender:       "sender",
		ContractAddr: "pair",
		Assets:       [2]Asset{{Addr: "asset0", Amount: "1"}, {Addr: "asset1", Amount: "0"}},
	}
	quarantine := ParseQuarantine{
		Height:   1,
		Hash:     rawTx.Hash,
		Stage:    PartialQuarantineStagePrefix + "wasm_transfer",
		Contract: "token",
		Action:   "transfer",
		Error:    "ambiguous amount",
		RawTx:    rawTx,
	}
	target := &quarantineTargetApp{parse: func(parser.RawTx, uint64) ([]ParsedTx, error) {
		return nil, &PartialParseQuarantineError{
			ParsedTxs:  []ParsedTx{parsedTx},
			Quarantine: quarantine,
			Err:        errors.New(quarantine.Error),
		}
	}}
	repo := &RepoMock{}
	srcStore := &RawStoreMock{}
	app := &dexApp{
		TargetApp:            target,
		Repo:                 repo,
		SourceDataStore:      srcStore,
		logger:               logging.Discard,
		poolSnapshotInterval: 100,
		sameHeightTolerance:  3,
		quarantineRetryMode:  configs.QuarantineRetryDisabled,
	}

	repo.On("GetTokenExceptions").Return(map[string]bool{}, nil)
	repo.On("GetSyncedHeight").Return(uint64(0), nil)
	srcStore.On("GetSourceSyncedHeight").Return(uint64(1), nil)
	srcStore.On("GetSourceTxs", uint64(1)).Return(parser.RawTxs{rawTx}, nil)
	repo.On("Insert", uint64(0), uint64(1), []ParsedTx{parsedTx}, []PoolInfo{}, []Pair{}, []ParseQuarantine{quarantine}).Return(nil)

	require.NoError(t, app.Run())
	repo.AssertExpectations(t)
	srcStore.AssertExpectations(t)
}

func Test_Run_LogsSummaryAtInfoAndHeightProcessedAtDebug(t *testing.T) {
	logBuf := &bytes.Buffer{}
	logger := logrus.New()
	logger.SetOutput(logBuf)
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	target := &quarantineTargetApp{parse: func(tx parser.RawTx, _ uint64) ([]ParsedTx, error) {
		return []ParsedTx{{
			Hash:         tx.Hash,
			Type:         Transfer,
			Sender:       "sender",
			ContractAddr: "pair",
			Assets:       [2]Asset{{Addr: "asset0", Amount: "1"}, {Addr: "asset1", Amount: "0"}},
		}}, nil
	}}
	repo := &RepoMock{}
	srcStore := &RawStoreMock{}
	app := &dexApp{
		TargetApp:            target,
		Repo:                 repo,
		SourceDataStore:      srcStore,
		logger:               logger.WithField("chainId", "chain-1"),
		chainId:              "chain-1",
		poolSnapshotInterval: 100,
		sameHeightTolerance:  3,
		quarantineRetryMode:  configs.QuarantineRetryDisabled,
	}

	repo.On("GetTokenExceptions").Return(map[string]bool{}, nil)
	repo.On("GetSyncedHeight").Return(uint64(0), nil)
	srcStore.On("GetSourceSyncedHeight").Return(uint64(2), nil)
	srcStore.On("GetSourceTxs", uint64(1)).Return(parser.RawTxs{{Hash: "tx1"}}, nil)
	srcStore.On("GetSourceTxs", uint64(2)).Return(parser.RawTxs{{Hash: "tx2"}}, nil)
	repo.On("Insert", uint64(0), uint64(1), mock.Anything, []PoolInfo{}, []Pair{}, []ParseQuarantine{}).Return(nil)
	repo.On("Insert", uint64(1), uint64(2), mock.Anything, []PoolInfo{}, []Pair{}, []ParseQuarantine{}).Return(nil)

	require.NoError(t, app.Run())

	var summary map[string]interface{}
	for _, line := range bytes.Split(bytes.TrimSpace(logBuf.Bytes()), []byte("\n")) {
		var entry map[string]interface{}
		require.NoError(t, json.Unmarshal(line, &entry))
		require.NotEqual(t, "parser.height_processed", entry["event"])
		if entry["event"] == "parser.run_summary" {
			summary = entry
		}
	}

	require.NotNil(t, summary)
	assert.Equal(t, float64(1), summary["from_height"])
	assert.Equal(t, float64(2), summary["to_height"])
	assert.Equal(t, float64(2), summary["processed_height_count"])
	assert.Equal(t, float64(2), summary["parsed_tx_count"])
	assert.Equal(t, float64(0), summary["quarantine_count"])
}

func Test_retryPendingQuarantines_ResolvesSuccessfulReplay(t *testing.T) {
	rawTx := parser.RawTx{Hash: "replay"}
	parsedTx := ParsedTx{
		Hash:         rawTx.Hash,
		Type:         Transfer,
		Sender:       "sender",
		ContractAddr: "pair",
		Assets:       [2]Asset{{Addr: "asset0", Amount: "1"}, {Addr: "asset1", Amount: "0"}},
	}
	target := &quarantineTargetApp{parse: func(tx parser.RawTx, height uint64) ([]ParsedTx, error) {
		require.Equal(t, rawTx, tx)
		require.Equal(t, uint64(10), height)
		return []ParsedTx{parsedTx}, nil
	}}
	repo := &RepoMock{}
	app := &dexApp{
		TargetApp: target,
		Repo:      repo,
		logger:    logging.Discard,
	}
	quarantine := ParseQuarantine{
		ID:     7,
		Height: 10,
		Hash:   rawTx.Hash,
		RawTx:  rawTx,
	}
	repo.On("PendingParseQuarantines").Return([]ParseQuarantine{quarantine}, nil)
	repo.On("ResolveParseQuarantine", uint64(7), uint64(10), []ParsedTx{parsedTx}).Return(nil)

	require.NoError(t, app.retryPendingQuarantines(map[string]bool{}))
	repo.AssertExpectations(t)
}

func Test_retryPendingQuarantines_LeavesAmbiguousReplayPending(t *testing.T) {
	rawTx := parser.RawTx{Hash: "still-ambiguous"}
	target := &quarantineTargetApp{parse: func(parser.RawTx, uint64) ([]ParsedTx, error) {
		return nil, &eventlog.AmbiguousEventError{
			Key:    "amount",
			Values: []string{"1", "2"},
		}
	}}
	repo := &RepoMock{}
	app := &dexApp{
		TargetApp: target,
		Repo:      repo,
		logger:    logging.Discard,
	}
	repo.On("PendingParseQuarantines").Return([]ParseQuarantine{{
		ID:     8,
		Height: 11,
		Hash:   rawTx.Hash,
		RawTx:  rawTx,
	}}, nil)

	require.NoError(t, app.retryPendingQuarantines(map[string]bool{}))
	repo.AssertNotCalled(t, "ResolveParseQuarantine", mock.Anything)
}

func Test_retryPendingQuarantines_SkipsPartialQuarantine(t *testing.T) {
	rawTx := parser.RawTx{Hash: "partial"}
	target := &quarantineTargetApp{parse: func(parser.RawTx, uint64) ([]ParsedTx, error) {
		t.Fatal("partial quarantines must not be retried")
		return nil, nil
	}}
	repo := &RepoMock{}
	app := &dexApp{
		TargetApp: target,
		Repo:      repo,
		logger:    logging.Discard,
	}
	repo.On("PendingParseQuarantines").Return([]ParseQuarantine{{
		ID:     9,
		Height: 12,
		Hash:   rawTx.Hash,
		Stage:  PartialQuarantineStagePrefix + "wasm_transfer",
		RawTx:  rawTx,
	}}, nil)

	require.NoError(t, app.retryPendingQuarantines(map[string]bool{}))
	repo.AssertNumberOfCalls(t, "ResolveParseQuarantine", 0)
}

func Test_validate(t *testing.T) {
	tcs := []struct {
		actualPools []PoolInfo
		exceptions  []string
		errMsg      string
		explain     string
	}{
		{
			[]PoolInfo{
				{ContractAddr: "ContractAddr1", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}},
				{ContractAddr: "ContractAddr2", TotalShare: "2000", Assets: []Asset{{"Asset0", "200"}, {"Asset1", "200"}}},
			},
			[]string{},
			"",
			"must match the expected pools",
		},
		{
			[]PoolInfo{
				{ContractAddr: "ContractAddr1", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}},
			},
			[]string{"ContractAddr2"},
			"",
			"ContractAddr2 is in exception list, must be skipped although actual pool does not match the expected pool",
		},
		{
			[]PoolInfo{
				{ContractAddr: "ContractAddr1", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}},
				{ContractAddr: "ContractAddr2", TotalShare: "2000", Assets: []Asset{{"Asset0", "200"}, {"Asset1", "100"}}},
			},
			[]string{"ContractAddr2"},
			"",
			"ContractAddr2 is in exception list, must be skipped although asset1 amount does not match the expected pool",
		},
		{
			[]PoolInfo{
				{ContractAddr: "ContractAddr1", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}},
				{ContractAddr: "ContractAddr2", TotalShare: "2000", Assets: []Asset{{"Asset0", "200"}, {"Asset1", "200"}}},
				{ContractAddr: "ContractAddr3", TotalShare: "3000", Assets: []Asset{{"Asset0", "300"}, {"Asset1", "300"}}},
				{ContractAddr: "ContractAddr4", TotalShare: "4000", Assets: []Asset{{"Asset0", "400"}, {"Asset1", "400"}}},
			},
			[]string{"ContractAddr3", "ContractAddr4"},
			"",
			"must be skipped because ContractAddr3 and ContractAddr4 are in exception list",
		},
	}
	expectedPools := []PoolInfo{
		{ContractAddr: "ContractAddr1", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}},
		{ContractAddr: "ContractAddr2", TotalShare: "2000", Assets: []Asset{{"Asset0", "200"}, {"Asset1", "200"}}},
	}
	from, to := uint64(0), uint64(0)

	for idx, tc := range tcs {
		repoMock := RepoMock{}
		repoMock.On("ParsedPoolsInfo", from, to).Return(tc.actualPools, nil)
		repoMock.On("ValidationExceptionList").Return(tc.exceptions, nil)
		dexApp := dexApp{Repo: &repoMock}
		err := dexApp.validate(uint64(from), uint64(to), expectedPools)
		errMsg := fmt.Sprintf("tc(%d): %s", idx, tc.explain)
		if tc.errMsg != "" {
			assert.Error(t, err, errMsg)
		} else {
			assert.NoError(t, err, errMsg)
		}
	}
}

func Test_validate_CollectsAllPoolValidationMismatches(t *testing.T) {
	expectedPools := []PoolInfo{
		{ContractAddr: "pair1", TotalShare: "1000", Assets: []Asset{{"asset0", "500"}, {"asset1", "500"}}},
		{ContractAddr: "pair2", TotalShare: "2000", Assets: []Asset{{"asset2", "1000"}, {"asset3", "1000"}}},
	}
	actualPools := []PoolInfo{
		{ContractAddr: "pair1", TotalShare: "900", Assets: []Asset{{"asset0", "400"}, {"asset1", "450"}}},
		{ContractAddr: "unexpected", TotalShare: "3000", Assets: []Asset{{"asset4", "1500"}, {"asset5", "1500"}}},
	}
	repoMock := RepoMock{}
	repoMock.On("ParsedPoolsInfo", uint64(0), uint64(100)).Return(actualPools, nil)
	repoMock.On("ValidationExceptionList").Return([]string{}, nil)
	app := dexApp{Repo: &repoMock}

	err := app.validate(0, 100, expectedPools)
	require.Error(t, err)

	var validationErr *poolValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.ElementsMatch(t, []poolValidationMismatch{
		{Type: validationMismatchAssetAmount, Contract: "pair1", Asset: "asset0", Actual: "400", Expected: "500"},
		{Type: validationMismatchAssetAmount, Contract: "pair1", Asset: "asset1", Actual: "450", Expected: "500"},
		{Type: validationMismatchTotalShare, Contract: "pair1", Actual: "900", Expected: "1000"},
		{Type: validationMismatchUnexpectedPool, Contract: "unexpected", Actual: "3000"},
		{Type: validationMismatchExpectedPoolMissing, Contract: "pair2", Expected: "2000"},
	}, validationErr.Mismatches)
}

func Test_validateAtHeight_LogsEveryPoolValidationMismatch(t *testing.T) {
	expectedPools := []PoolInfo{
		{ContractAddr: "pair1", TotalShare: "1000", Assets: []Asset{{"asset0", "500"}, {"asset1", "500"}}},
		{ContractAddr: "pair2", TotalShare: "2000", Assets: []Asset{{"asset2", "1000"}, {"asset3", "1000"}}},
	}
	actualPools := []PoolInfo{
		{ContractAddr: "pair1", TotalShare: "900", Assets: []Asset{{"asset0", "400"}, {"asset1", "500"}}},
		{ContractAddr: "unexpected", TotalShare: "3000", Assets: []Asset{{"asset4", "1500"}, {"asset5", "1500"}}},
	}
	logBuf := &bytes.Buffer{}
	logger := logrus.New()
	logger.SetOutput(logBuf)
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	srcStore := &RawStoreMock{}
	repo := &RepoMock{}
	app := &dexApp{
		Repo:            repo,
		SourceDataStore: srcStore,
		logger:          logger.WithField("chainId", "chain-1"),
		chainId:         "chain-1",
	}

	srcStore.On("GetPoolInfos", uint64(100)).Return(expectedPools, nil)
	repo.On("ParsedPoolsInfo", uint64(0), uint64(100)).Return(actualPools, nil)
	repo.On("ValidationExceptionList").Return([]string{}, nil)

	assert.False(t, app.validateAtHeight(100))

	var logs []map[string]interface{}
	for _, line := range bytes.Split(bytes.TrimSpace(logBuf.Bytes()), []byte("\n")) {
		var entry map[string]interface{}
		require.NoError(t, json.Unmarshal(line, &entry))
		if entry["event"] == "parser.pool_validation_failed" {
			logs = append(logs, entry)
		}
	}

	require.Len(t, logs, 4)
	assertLogFields := func(contract, mismatchType string) bool {
		for _, entry := range logs {
			if entry["contract"] == contract && entry["mismatch_type"] == mismatchType {
				assert.Equal(t, "chain-1", entry["chain_id"])
				assert.Equal(t, float64(100), entry["height"])
				assert.Equal(t, "pool_validation", entry["operation"])
				return true
			}
		}
		return false
	}
	assert.True(t, assertLogFields("pair1", validationMismatchAssetAmount))
	assert.True(t, assertLogFields("pair1", validationMismatchTotalShare))
	assert.True(t, assertLogFields("unexpected", validationMismatchUnexpectedPool))
	assert.True(t, assertLogFields("pair2", validationMismatchExpectedPoolMissing))
}

var (
	createTx   = ParsedTx{"", time.Time{}, CreatePair, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "1000"}, {"Asset1", "1000"}}, "Lp", "1000", "", 0, nil}
	swapTx     = ParsedTx{"", time.Time{}, Swap, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "1000"}, {"Asset1", "-1000"}}, "", "", "1", 0, nil}
	provideTx  = ParsedTx{"", time.Time{}, Provide, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "1000"}, {"Asset1", "1000"}}, "Lp", "1000", "", 0, nil}
	withdrawTx = ParsedTx{"", time.Time{}, Withdraw, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "-1000"}, {"Asset1", "-1000"}}, "Lp", "1000", "", 0, nil}
	transferTx = ParsedTx{"", time.Time{}, Transfer, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", ""}, {"Asset1", "1000"}}, "", "", "", 0, nil}
)

// Test_matchesPairTransferEntry verifies that transfers are matched to pair entries
// correctly: user->pair requires exact amount; pair->user skips the amount check
// for CW20 tokens (tax may apply) but requires exact amount for native and IBC tokens.
func Test_matchesPairTransferEntry(t *testing.T) {
	mixin := DexMixin{}
	pairAddr := "PAIR_ADDR"

	tcs := []struct {
		entry    transferPopEntry
		transfer ParsedTx
		expected bool
		explain  string
	}{
		// user -> pair
		{
			entry:    transferPopEntry{pairAddr, "TokenA", "1000"},
			transfer: ParsedTx{ContractAddr: pairAddr, Assets: [2]Asset{{"TokenA", "1000"}, {"TokenB", ""}}},
			expected: true,
			explain:  "user->pair transfer with exact amount must match",
		},
		{
			entry:    transferPopEntry{pairAddr, "TokenA", "1000"},
			transfer: ParsedTx{ContractAddr: pairAddr, Assets: [2]Asset{{"TokenA", "999"}, {"TokenB", ""}}},
			expected: false,
			explain:  "user->pair transfer with mismatched amount must not match (unrelated transfer)",
		},
		// pair -> user
		{
			entry:    transferPopEntry{pairAddr, "terra1sepfj7s0aeg5967uxnfk4thzlerrsktkpelm5s", "-500"},
			transfer: ParsedTx{Sender: pairAddr, Assets: [2]Asset{{"TokenA", ""}, {"terra1sepfj7s0aeg5967uxnfk4thzlerrsktkpelm5s", "490"}}},
			expected: true,
			explain:  "pair->user transfer of CW20 token with different amount must match (no amount check)",
		},
		{
			entry:    transferPopEntry{pairAddr, "uusd", "-500"},
			transfer: ParsedTx{Sender: pairAddr, Assets: [2]Asset{{"uusd", "-500"}, {}}},
			expected: true,
			explain:  "pair->user transfer of native token with exact amount must match",
		},
		{
			entry:    transferPopEntry{pairAddr, "uusd", "-500"},
			transfer: ParsedTx{Sender: pairAddr, Assets: [2]Asset{{"uusd", "-490"}, {}}},
			expected: false,
			explain:  "pair->user transfer of native token with different amount must not match",
		},
		{
			entry:    transferPopEntry{pairAddr, "ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4", "-500"},
			transfer: ParsedTx{Sender: pairAddr, Assets: [2]Asset{{"ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4", "-500"}, {}}},
			expected: true,
			explain:  "pair->user transfer of IBC token with exact amount must match",
		},
		{
			entry:    transferPopEntry{pairAddr, "ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4", "-500"},
			transfer: ParsedTx{Sender: pairAddr, Assets: [2]Asset{{"ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4", "-490"}, {}}},
			expected: false,
			explain:  "pair->user transfer of IBC token with different amount must not match",
		},
		{
			entry:    transferPopEntry{pairAddr, "TokenB", "500"},
			transfer: ParsedTx{Sender: pairAddr, Assets: [2]Asset{{"TokenA", ""}, {"TokenB", "500"}}},
			expected: false,
			explain:  "pair->user transfer with positive sign must not match",
		},
		{
			entry:    transferPopEntry{pairAddr, "TokenA", "1000"},
			transfer: ParsedTx{ContractAddr: "OTHER_PAIR", Assets: [2]Asset{{"TokenA", "1000"}, {"TokenB", ""}}},
			expected: false,
			explain:  "different contract must not match",
		},
	}

	for _, tc := range tcs {
		result := mixin.matchesPairTransferEntry(tc.entry, &tc.transfer)
		assert.Equal(t, tc.expected, result, tc.explain)
	}
}

// Test_removeDuplicatedTxs verifies that transfer txs already captured by pair
// action events are removed, while unrelated transfers in the same tx are kept.
func Test_removeDuplicatedTxs(t *testing.T) {
	mixin := DexMixin{}
	pairAddr := "PAIR_ADDR"
	pairTx := &ParsedTx{ContractAddr: pairAddr, Assets: [2]Asset{{"TokenA", "1000"}, {"TokenB", "-500"}}}

	tcs := []struct {
		pairTxs     []*ParsedTx
		transferTxs []*ParsedTx
		expected    int
		explain     string
	}{
		{
			pairTxs:     []*ParsedTx{pairTx},
			transferTxs: []*ParsedTx{{ContractAddr: pairAddr, Assets: [2]Asset{{"TokenA", "1000"}, {"TokenB", ""}}}},
			expected:    0,
			explain:     "user->pair transfer matching pair action must be removed",
		},
		{
			pairTxs:     []*ParsedTx{{ContractAddr: pairAddr, Assets: [2]Asset{{"TokenA", "1000"}, {"terra1sepfj7s0aeg5967uxnfk4thzlerrsktkpelm5s", "-500"}}}},
			transferTxs: []*ParsedTx{{Sender: pairAddr, Assets: [2]Asset{{"TokenA", ""}, {"terra1sepfj7s0aeg5967uxnfk4thzlerrsktkpelm5s", "-490"}}}},
			expected:    0,
			explain:     "pair->user transfer of CW20 token with amount mismatch must be removed",
		},
		{
			pairTxs:     []*ParsedTx{{ContractAddr: pairAddr, Assets: [2]Asset{{"TokenA", "1000"}, {"uusd", "-500"}}}},
			transferTxs: []*ParsedTx{{Sender: pairAddr, Assets: [2]Asset{{"TokenA", ""}, {"uusd", "-500"}}}},
			expected:    0,
			explain:     "pair->user transfer of native token with exact amount must be removed",
		},
		{
			pairTxs:     []*ParsedTx{{ContractAddr: pairAddr, Assets: [2]Asset{{"TokenA", "1000"}, {"uusd", "-500"}}}},
			transferTxs: []*ParsedTx{{Sender: pairAddr, Assets: [2]Asset{{"TokenA", ""}, {"uusd", "-490"}}}},
			expected:    1,
			explain:     "pair->user transfer of native token with amount mismatch must be kept (native transfers are exact)",
		},
		{
			pairTxs: []*ParsedTx{pairTx},
			transferTxs: []*ParsedTx{
				{ContractAddr: pairAddr, Assets: [2]Asset{{"TokenA", "1000"}, {"TokenB", ""}}},
				{ContractAddr: pairAddr, Assets: [2]Asset{{"TokenA", "500"}, {"TokenB", ""}}},
			},
			expected: 1,
			explain:  "unrelated transfer to pair with different amount in same tx must be kept",
		},
		{
			pairTxs:     []*ParsedTx{pairTx},
			transferTxs: []*ParsedTx{{ContractAddr: "OTHER_PAIR", Assets: [2]Asset{{"TokenA", "1000"}, {"TokenB", ""}}}},
			expected:    1,
			explain:     "transfer to unrelated pair must be kept",
		},
		{
			pairTxs: []*ParsedTx{{ContractAddr: pairAddr, Assets: [2]Asset{{"TokenA", "0"}, {"TokenB", "0"}}}},
			transferTxs: []*ParsedTx{
				{Sender: pairAddr, Assets: [2]Asset{{"TokenA", "1000"}, {"TokenB", ""}}},
				{Sender: pairAddr, Assets: [2]Asset{{"TokenA", ""}, {"TokenB", "1000"}}},
			},
			expected: 2,
			explain:  "withdraw where pair records 0 amount: transfer must not be consumed (actual movement recorded separately)",
		},
	}

	for _, tc := range tcs {
		result := mixin.RemoveDuplicatedTxs(tc.pairTxs, tc.transferTxs)
		assert.Len(t, result, tc.expected, tc.explain)
	}
}

func Test_collectLpBurnTxs(t *testing.T) {
	lpPairAddrs := map[string]string{
		"LpToken": "PairContract",
	}

	tcs := []struct {
		burnTxs  []*ParsedTx
		expected []ParsedTx
		errMsg   string
	}{
		{
			burnTxs:  []*ParsedTx{{LpAddr: "LpToken", LpAmount: "-1000"}},
			expected: []ParsedTx{{LpAddr: "LpToken", ContractAddr: "PairContract", LpAmount: "-1000"}},
			errMsg:   "known LP addr must be collected with pair contract addr assigned",
		},
		{
			burnTxs:  []*ParsedTx{{LpAddr: "UnknownLpToken", LpAmount: "-1000"}},
			expected: []ParsedTx{},
			errMsg:   "unknown LP addr must be filtered out",
		},
		{
			burnTxs:  []*ParsedTx{{LpAddr: "LpToken", Sender: "PairContract", LpAmount: "-1000"}},
			expected: []ParsedTx{},
			errMsg:   "burn from pair contract itself (withdraw lp burn) must be filtered out",
		},
	}

	for _, tc := range tcs {
		assert := assert.New(t)
		result := CollectLpBurnTxs(tc.burnTxs, lpPairAddrs)
		assert.Equal(tc.expected, result, tc.errMsg)
	}
}

// testRepoMock extends RepoMock to control the persisted validation height
// and track every SetValidationHeight call.
type testRepoMock struct {
	RepoMock
	mu                sync.Mutex
	validationHeight  uint64
	setValidationArgs []uint64
	setValidationErrs []error
	clearCount        int
}

func (m *testRepoMock) GetValidationHeight() (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.validationHeight, nil
}

func (m *testRepoMock) SetValidationHeight(h uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setValidationArgs = append(m.setValidationArgs, h)
	if len(m.setValidationErrs) > 0 {
		err := m.setValidationErrs[0]
		m.setValidationErrs = m.setValidationErrs[1:]
		if err != nil {
			return err
		}
	}
	m.validationHeight = h
	return nil
}

func (m *testRepoMock) ClearValidationHeight() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clearCount++
	m.validationHeight = 0
	return nil
}

func (m *testRepoMock) validationState() (uint64, []uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.validationHeight, append([]uint64(nil), m.setValidationArgs...)
}

func Test_processPendingValidations_AdvancesPersistedCursor(t *testing.T) {
	pool := PoolInfo{
		ContractAddr: "pool1",
		TotalShare:   "1000",
		Assets:       []Asset{{"token0", "500"}, {"token1", "500"}},
	}
	const firstHeight = uint64(100)
	const secondHeight = uint64(200)

	srcStore := &RawStoreMock{}
	repo := &testRepoMock{validationHeight: firstHeight}
	app := &dexApp{
		Repo:               repo,
		SourceDataStore:    srcStore,
		logger:             logging.Discard,
		validationInterval: 100,
	}

	srcStore.On("GetPoolInfos", firstHeight).Return([]PoolInfo{pool}, nil)
	srcStore.On("GetPoolInfos", secondHeight).Return([]PoolInfo{pool}, nil)
	repo.On("ParsedPoolsInfo", uint64(0), firstHeight).Return([]PoolInfo{pool}, nil)
	repo.On("ParsedPoolsInfo", uint64(0), secondHeight).Return([]PoolInfo{pool}, nil)
	repo.On("ValidationExceptionList").Return([]string{}, nil)

	app.processPendingValidations(250)

	assert.Equal(t, []uint64{secondHeight, 300}, repo.setValidationArgs)
	assert.Equal(t, uint64(300), repo.validationHeight)
}

func Test_processPendingValidations_LeavesCursorOnValidationFailure(t *testing.T) {
	pool := PoolInfo{
		ContractAddr: "pool1",
		TotalShare:   "1000",
		Assets:       []Asset{{"token0", "500"}, {"token1", "500"}},
	}
	const height = uint64(100)

	srcStore := &RawStoreMock{}
	repo := &testRepoMock{validationHeight: height}
	app := &dexApp{
		Repo:               repo,
		SourceDataStore:    srcStore,
		logger:             logging.Discard,
		validationInterval: 100,
	}

	srcStore.On("GetPoolInfos", height).Return([]PoolInfo{pool}, nil)
	repo.On("ParsedPoolsInfo", uint64(0), height).Return([]PoolInfo{}, nil)
	repo.On("ValidationExceptionList").Return([]string{}, nil)

	app.processPendingValidations(height)

	assert.Empty(t, repo.setValidationArgs)
	assert.Equal(t, height, repo.validationHeight)
}

func Test_triggerValidation_PersistsCursorBeforeValidation(t *testing.T) {
	pool := PoolInfo{
		ContractAddr: "pool1",
		TotalShare:   "1000",
		Assets:       []Asset{{"token0", "500"}, {"token1", "500"}},
	}
	const height = uint64(100)

	srcStore := &RawStoreMock{}
	repo := &testRepoMock{}
	app := &dexApp{
		Repo:               repo,
		SourceDataStore:    srcStore,
		logger:             logging.Discard,
		validationInterval: 100,
	}

	srcStore.On("GetPoolInfos", height).Return([]PoolInfo{pool}, nil)
	repo.On("ParsedPoolsInfo", uint64(0), height).Return([]PoolInfo{pool}, nil)
	repo.On("ValidationExceptionList").Return([]string{}, nil)

	app.triggerValidation(height)

	require.Eventually(t, func() bool {
		validationHeight, setValidationArgs := repo.validationState()
		return validationHeight == 200 && assert.ObjectsAreEqual([]uint64{height, 200}, setValidationArgs)
	}, time.Second, 10*time.Millisecond)
}

func Test_processPendingValidations_RetriesCursorAfterAdvanceFailure(t *testing.T) {
	pool := PoolInfo{
		ContractAddr: "pool1",
		TotalShare:   "1000",
		Assets:       []Asset{{"token0", "500"}, {"token1", "500"}},
	}
	const height = uint64(100)

	srcStore := &RawStoreMock{}
	repo := &testRepoMock{
		validationHeight:  height,
		setValidationErrs: []error{errors.New("db unavailable")},
	}
	app := &dexApp{
		Repo:               repo,
		SourceDataStore:    srcStore,
		logger:             logging.Discard,
		validationInterval: 100,
	}

	srcStore.On("GetPoolInfos", height).Return([]PoolInfo{pool}, nil)
	repo.On("ParsedPoolsInfo", uint64(0), height).Return([]PoolInfo{pool}, nil)
	repo.On("ValidationExceptionList").Return([]string{}, nil)

	app.processPendingValidations(height)

	assert.Equal(t, []uint64{200}, repo.setValidationArgs)
	assert.Equal(t, height, repo.validationHeight)
}

func Test_Run_FollowsPersistedValidationCursor(t *testing.T) {
	pool := PoolInfo{
		ContractAddr: "pool1",
		TotalShare:   "1000",
		Assets:       []Asset{{"token0", "500"}, {"token1", "500"}},
	}
	const firstHeight = uint64(100)
	const secondHeight = uint64(200)

	srcStore := &RawStoreMock{}
	repo := &testRepoMock{validationHeight: firstHeight}
	app := &dexApp{
		Repo:                repo,
		SourceDataStore:     srcStore,
		logger:              logging.Discard,
		validationInterval:  100,
		sameHeightTolerance: 5,
	}

	repo.On("GetTokenExceptions").Return(map[string]bool{}, nil)
	repo.On("GetSyncedHeight").Return(secondHeight, nil)
	srcStore.On("GetSourceSyncedHeight").Return(secondHeight, nil)
	srcStore.On("GetPoolInfos", firstHeight).Return([]PoolInfo{pool}, nil)
	srcStore.On("GetPoolInfos", secondHeight).Return([]PoolInfo{pool}, nil)
	repo.On("ParsedPoolsInfo", uint64(0), firstHeight).Return([]PoolInfo{pool}, nil)
	repo.On("ParsedPoolsInfo", uint64(0), secondHeight).Return([]PoolInfo{pool}, nil)
	repo.On("ValidationExceptionList").Return([]string{}, nil)

	require.NoError(t, app.Run())

	require.Eventually(t, func() bool {
		validationHeight, setValidationArgs := repo.validationState()
		return validationHeight == 300 && assert.ObjectsAreEqual([]uint64{secondHeight, 300}, setValidationArgs)
	}, time.Second, 10*time.Millisecond)
}
