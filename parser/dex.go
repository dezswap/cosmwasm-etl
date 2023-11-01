package parser

import (
	"fmt"
	"strings"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
)

type PairParsers struct {
	CreatePairParser Parser
	PairActionParser Parser
	InitialProvide   Parser
	WasmTransfer     Parser
	Transfer         Parser
}

// common mixin logic for all dex apps
type dexApp struct {
	TargetApp
	SourceDataStore
	Repo
	chainId string
	logger  logging.Logger

	sameHeightTolerance uint
	lastSrcHeight       uint64
	sameHeightCount     uint
}

type DexMixin struct{}

var _ Dex = &dexApp{}

func NewDexApp(app TargetApp, srcStore SourceDataStore, repo Repo, logger logging.Logger, c configs.ParserConfig) *dexApp {
	return &dexApp{
		TargetApp:           app,
		SourceDataStore:     srcStore,
		Repo:                repo,
		logger:              logger,
		chainId:             c.ChainId,
		sameHeightTolerance: c.SameHeightTolerance,
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

		poolInfos, err := app.GetPoolInfos(cur)
		if err != nil {
			return errors.Wrap(err, "app.Run")
		}

		if err := app.insert(cur, parsedTxs, poolInfos); err != nil {
			return errors.Wrap(err, "app.Run")
		}
	}
	app.lastSrcHeight = srcHeight

	return nil
}

// insert implements parser
func (p *dexApp) insert(height uint64, txs []ParsedTx, pools []PoolInfo) error {
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

	err := p.Repo.Insert(height, txs, pools, pairDtos)
	if err != nil {
		return errors.Wrap(err, "insert")
	}

	return nil
}

func (p *DexMixin) HasProvide(pairTxs []*ParsedTx) bool {
	for _, tx := range pairTxs {
		if tx.Type == Provide {
			return true
		}
	}
	return false
}

func (p *DexMixin) RemoveDuplicatedTxs(pairTxs []*ParsedTx, transferTxs []*ParsedTx) []ParsedTx {
	txs := []ParsedTx{}
	for idx, tx := range transferTxs {
		duplicated := false
		for i := 0; i < len(pairTxs) && !duplicated; i++ {
			duplicated = p.isDuplicatedTx(pairTxs[i], tx)
		}
		if !duplicated {
			txs = append(txs, *transferTxs[idx])
		}
	}
	return txs
}

func (p *DexMixin) isDuplicatedTx(ptx *ParsedTx, transfer *ParsedTx) bool {
	isSameAssetAmount := func(a1, a2 Asset) bool {
		return a1.Addr == a2.Addr && a1.Amount == a2.Amount
	}
	return (ptx.ContractAddr == transfer.ContractAddr || ptx.ContractAddr == transfer.Sender) && (isSameAssetAmount(ptx.Assets[0], transfer.Assets[0]) || isSameAssetAmount(ptx.Assets[1], transfer.Assets[1]))
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
