package dezswap

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	pdex "github.com/dezswap/cosmwasm-etl/pkg/dex"
	ds "github.com/dezswap/cosmwasm-etl/pkg/dex/dezswap"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
	"strings"
)

// runner for terraswap
type dezswapApp struct {
	dex.PairRepo
	Parsers *dex.PairParsers
	dex.DexMixin
	chainId string
}

var _ dex.TargetApp = &dezswapApp{}

func New(repo dex.PairRepo, _ logging.Logger, _ configs.ParserDexConfig, chainId string) (dex.TargetApp, error) {
	return &dezswapApp{repo, &dex.PairParsers{}, dex.DexMixin{}, chainId}, nil
}

func (p *dezswapApp) ParseTxs(tx parser.RawTx, height uint64) ([]dex.ParsedTx, error) {
	pairs, err := p.GetPairs()
	if err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}

	if err := p.updateParsers(pairs, height); err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}

	txDtos := []dex.ParsedTx{}
	createPairTxs, err := p.Parsers.CreatePairParser.Parse(tx.LogResults, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}
	for _, ctx := range createPairTxs {
		pairs[ctx.ContractAddr] = dex.Pair{
			ContractAddr: ctx.ContractAddr,
			LpAddr:       ctx.LpAddr,
			Assets:       []string{ctx.Assets[0].Addr, ctx.Assets[1].Addr},
		}
		ctx.Sender = tx.Sender
		txDtos = append(txDtos, *ctx)
	}

	pairTxs := []*dex.ParsedTx{}
	wasmTransferTxs := []*dex.ParsedTx{}
	transferTxs := []*dex.ParsedTx{}
	for _, raw := range tx.LogResults {
		ptxs, err := p.Parsers.PairActionParser.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		pairTxs = append(pairTxs, ptxs...)

		// find initial provide to a pair
		if p.HasProvide(ptxs) {
			ipTxs, err := p.Parsers.InitialProvide.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
			if err != nil {
				return nil, errors.Wrap(err, "parseTxs")
			}
			pairTxs = append(pairTxs, ipTxs...)
		}

		wtxs, err := p.Parsers.WasmTransfer.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		wasmTransferTxs = append(wasmTransferTxs, wtxs...)

		transfers, err := p.Parsers.Transfer.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		transferTxs = append(transferTxs, transfers...)
	}
	for _, ptx := range pairTxs {
		ptx.Sender = tx.Sender
		txDtos = append(txDtos, *ptx)
	}

	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, wasmTransferTxs)...)
	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, transferTxs)...)

	return txDtos, nil
}

func (p *dezswapApp) updateParsers(pairs map[string]dex.Pair, height uint64) error {
	postAttrLen, isUpdateHeight := getPostEventAttrLen(p.chainId, height)

	// create pair parser
	{
		if p.Parsers.CreatePairParser == nil || isUpdateHeight {
			finder, err := ds.CreateCreatePairRuleFinder(p.chainId)
			if err != nil {
				return errors.Wrap(err, "updateParsers")
			}

			p.Parsers.CreatePairParser = parser.NewParser[dex.ParsedTx](
				finder, &createPairMapper{MapperMixin: pdex.MapperMixin{PostEventAttrLen: postAttrLen}})
		}
	}

	pairFilter := make(map[string]bool)
	for k := range pairs {
		pairFilter[k] = true
	}

	// pair action parser
	{
		pairFinder, err := ds.CreatePairAllRulesFinder(pairFilter)
		if err != nil {
			return errors.Wrap(err, "updateParsers")
		}

		pairMapper, err := pairMapperBy(p.chainId, height, pairs)
		if err != nil {
			return errors.Wrap(err, "updateParsers")
		}
		p.Parsers.PairActionParser = parser.NewParser[dex.ParsedTx](pairFinder, pairMapper)
	}

	// initial provide parser
	{
		initialProvideFinder, err := pdex.CreatePairInitialProvideRuleFinder(pairFilter)
		if err != nil {
			return errors.Wrap(err, "updateParsers")
		}
		p.Parsers.InitialProvide = parser.NewParser[dex.ParsedTx](initialProvideFinder, dex.NewInitialProvideMapper(postAttrLen))
	}

	// wasm transfer parser
	{
		wasmTransferFinder, err := ds.CreateWasmCommonTransferRuleFinder()
		if err != nil {
			return errors.Wrap(err, "updateParsers")
		}
		p.Parsers.WasmTransfer = parser.NewParser[dex.ParsedTx](
			wasmTransferFinder, &wasmTransferMapper{mixin: transferMapperMixin{pdex.MapperMixin{PostEventAttrLen: postAttrLen}}, pairSet: pairs})
	}

	// transfer parser
	{
		transferRule, err := ds.CreateTransferRuleFinder()
		if err != nil {
			return errors.Wrap(err, "updateParsers")
		}
		p.Parsers.Transfer = parser.NewParser[dex.ParsedTx](transferRule, &transferMapper{pairSet: pairs})
	}

	return nil
}

func getPostEventAttrLen(chainId string, height uint64) (len int, isUpdateHeight bool) {
	if strings.HasPrefix(chainId, ds.TestnetPrefix) {
		isUpdateHeight = height == ds.TestnetSdkV50Height
		if height >= ds.TestnetSdkV50Height {
			len++
		}
	}

	return
}
