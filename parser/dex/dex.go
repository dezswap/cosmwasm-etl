package dex

import (
	"fmt"
	"strings"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
)

type PairParsers struct {
	CreatePairParser parser.Parser[ParsedTx]
	PairActionParser parser.Parser[ParsedTx]
	InitialProvide   parser.Parser[ParsedTx]
	WasmTransfer     parser.Parser[ParsedTx]
	Transfer         parser.Parser[ParsedTx]
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
	localSynced, err := app.GetSyncedHeight()
	if err != nil {
		return errors.Wrap(err, "app.Run")
	}

	srcHeight, err := app.GetSourceSyncedHeight()
	if err != nil {
		return errors.Wrap(err, "app.Run")
	}

	if srcHeight < localSynced {
		return errors.New("remote height is less than local synced height")
	}

	if err := app.checkRemoteHeight(srcHeight); err != nil {
		return errors.Wrap(err, "app.Run")
	}

	// to avoid skipping validation error
	if (localSynced % uint64(app.validationInterval)) == 0 {
		poolInfos, err := app.GetPoolInfos((localSynced))
		if err != nil {
			return errors.Wrap(err, "app.Run")
		}
		if err := app.validate(0, (localSynced), poolInfos); err != nil {
			return errors.Wrap(err, "app.Run")
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
			return errors.Wrap(err, "app.Run")
		}

		parsedTxs := []ParsedTx{}
		for _, tx := range txs {
			txs, err := app.ParseTxs(tx, cur)
			if err != nil {
				return errors.Wrap(err, "app.Run")
			}
			parsedTxs = append(parsedTxs, txs...)
		}

		poolInfos := []PoolInfo{}
		if (cur % uint64(app.poolSnapshotInterval)) == 0 {
			poolInfos, err = app.GetPoolInfos(cur)
			if err != nil {
				return errors.Wrap(err, "app.Run")
			}
		}

		if err := app.insert(cur, parsedTxs, poolInfos); err != nil {
			return errors.Wrap(err, "app.Run")
		}

		if (cur % uint64(app.validationInterval)) == 0 {
			if len(poolInfos) == 0 {
				poolInfos, err = app.GetPoolInfos(cur)
				if err != nil {
					return errors.Wrap(err, "app.Run")
				}
			}
			if err := app.validate(0, cur, poolInfos); err != nil {
				return errors.Wrap(err, "app.Run")
			}
		}
	}
	app.lastSrcHeight = srcHeight

	return nil
}

// insert implements parser
func (app *dexApp) insert(height uint64, txs []ParsedTx, pools []PoolInfo) error {
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

	err := app.Repo.Insert(height, txs, pools, pairDtos)
	if err != nil {
		return errors.Wrap(err, "insert")
	}

	return nil
}

// checkRemoteHeight implements Dex
func (app *dexApp) checkRemoteHeight(srcHeight uint64) error {
	if srcHeight == app.lastSrcHeight {
		app.sameHeightCount++
		if app.sameHeightCount > app.sameHeightTolerance {
			errMsg := fmt.Sprintf("remote node height(%d) remains the same for %d consecutive times", srcHeight, app.sameHeightCount)
			return errors.New(errMsg)
		}
	} else {
		app.sameHeightCount = 0
	}
	return nil
}

// insert implements parser
func (app *dexApp) validate(from, to uint64, expected []PoolInfo) error {
	if len(expected) == 0 {
		app.logger.Infof("No pool info found at height %d", to)
		return nil
	}

	// TODO: snapshot for expected pools
	// e.g.) expected pools can be difference between pool of height 1000 and 900
	actual, err := app.ParsedPoolsInfo(from, to)
	if err != nil {
		return errors.Wrap(err, "dexApp.validate")
	}

	expectedPool := make(map[string]PoolInfo)
	for _, pool := range expected {
		expectedPool[pool.ContractAddr] = pool
	}
	for _, pool := range actual {
		exp, ok := expectedPool[pool.ContractAddr]
		if !ok {
			return errors.New(fmt.Sprintf("unexpected pool(%s) found", pool.ContractAddr))
		}
		for idx, expAsset := range exp.Assets {
			if expAsset.Amount != pool.Assets[idx].Amount {
				msg := fmt.Sprintf(
					"pool(%s) asset(%s) amount mismatch: actual(%s), expected(%s)", pool.ContractAddr, expAsset.Addr, pool.Assets[idx].Amount, expAsset.Amount,
				)
				return errors.New(msg)
			}
		}
		if exp.TotalShare != pool.TotalShare {
			msg := fmt.Sprintf(
				"pool(%s) total share mismatch: actual(%s), expected(%s)", pool.ContractAddr, pool.TotalShare, exp.TotalShare,
			)
			return errors.New(msg)
		}
		delete(expectedPool, pool.ContractAddr)
	}
	if len(expectedPool) > 0 {
		addrs := []string{}
		for _, pool := range expectedPool {
			addrs = append(addrs, pool.ContractAddr)
		}
		return errors.New(fmt.Sprintf("expected pools(%s) not found", addrs))
	}
	return nil
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
