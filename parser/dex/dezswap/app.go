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
)

// runner for terraswap
type dezswapApp struct {
	dex.PairRepo
	Parsers *dex.PairParsers
	dex.DexMixin
	chainId string

	// state
	pairs       map[string]dex.Pair
	lpPairAddrs map[string]string
}

var _ dex.TargetApp = &dezswapApp{}

func New(repo dex.PairRepo, _ logging.Logger, c configs.ParserDexConfig, chainId string) (dex.TargetApp, error) {
	finder, err := ds.CreateCreatePairRuleFinder(c.FactoryAddress)
	if err != nil {
		return nil, errors.Wrap(err, "dezswap.New")
	}

	parsers := &dex.PairParsers{
		CreatePairParser: parser.NewParser[dex.ParsedTx](finder, &createPairMapper{}),
		PairActionParser: nil,
		InitialProvide:   nil,
		WasmTransfer:     nil,
		Transfer:         nil,
	}

	pairs, err := repo.GetPairs()
	if err != nil {
		return nil, errors.Wrap(err, "dezswap.New")
	}

	lpPairAddrs := make(map[string]string)
	for _, p := range pairs {
		lpPairAddrs[p.LpAddr] = p.ContractAddr
	}

	return &dezswapApp{repo, parsers, dex.DexMixin{}, chainId, pairs, lpPairAddrs}, nil
}

func (p *dezswapApp) ParseTxs(tx parser.RawTx, height uint64) ([]dex.ParsedTx, error) {
	txDtos := []dex.ParsedTx{}
	createPairTxs, err := p.Parsers.CreatePairParser.Parse(tx.LogResults, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}
	for _, ctx := range createPairTxs {
		p.pairs[ctx.ContractAddr] = dex.Pair{
			ContractAddr: ctx.ContractAddr,
			LpAddr:       ctx.LpAddr,
			Assets:       []string{ctx.Assets[0].Addr, ctx.Assets[1].Addr},
		}
		p.lpPairAddrs[ctx.LpAddr] = ctx.ContractAddr
		ctx.Sender = tx.Sender
		txDtos = append(txDtos, *ctx)
	}

	if err := p.updateParsers(p.pairs, height); err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}

	pairTxs := []*dex.ParsedTx{}
	wasmTransferTxs := []*dex.ParsedTx{}
	transferTxs := []*dex.ParsedTx{}
	burnTxs := []*dex.ParsedTx{}
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

		burns, err := p.Parsers.BurnParser.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		burnTxs = append(burnTxs, burns...)
	}

	for _, ptx := range pairTxs {
		ptx.Sender = tx.Sender
		txDtos = append(txDtos, *ptx)
	}

	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, wasmTransferTxs)...)
	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, transferTxs)...)
	txDtos = append(txDtos, p.collectLpBurnTxs(burnTxs)...)

	return txDtos, nil
}

// collectLpBurnTxs collects LP burn events and associates them with their pair contract.
// LP tokens can be burned directly (outside of withdraw_liquidity), so we need to track
// these burns separately and subtract the burned amount to keep pool calculations accurate.
func (p *dezswapApp) collectLpBurnTxs(burnTxs []*dex.ParsedTx) []dex.ParsedTx {
	lpBurnTxs := []dex.ParsedTx{}
	for _, t := range burnTxs {
		if pairAddr, ok := p.lpPairAddrs[t.LpAddr]; ok {
			t.ContractAddr = pairAddr
			lpBurnTxs = append(lpBurnTxs, *t)
		}
	}

	return lpBurnTxs
}

func (p *dezswapApp) IsValidationExceptionCandidate(contractAddress string) bool {
	return false
}

func (p *dezswapApp) updateParsers(pairs map[string]dex.Pair, height uint64) error {
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
		p.Parsers.PairActionParser = parser.NewParser(pairFinder, pairMapper)
	}

	// initial provide parser
	{
		initialProvideFinder, err := pdex.CreatePairInitialProvideRuleFinder(pairFilter)
		if err != nil {
			return errors.Wrap(err, "updateParsers")
		}
		p.Parsers.InitialProvide = parser.NewParser(initialProvideFinder, dex.NewInitialProvideMapper())
	}

	// wasm transfer parser
	{
		wasmTransferFinder, err := ds.CreateWasmCommonTransferRuleFinder()
		if err != nil {
			return errors.Wrap(err, "updateParsers")
		}
		p.Parsers.WasmTransfer = parser.NewParser(
			wasmTransferFinder, &wasmTransferMapper{mixin: transferMapperMixin{pdex.MapperMixin{}}, pairSet: pairs})
	}

	// transfer parser
	{
		transferRule, err := ds.CreateTransferRuleFinder()
		if err != nil {
			return errors.Wrap(err, "updateParsers")
		}
		p.Parsers.Transfer = parser.NewParser(transferRule, &transferMapper{pairSet: pairs})
	}

	// burn parser - to collect and parse LP burn event
	{
		burnRule, err := pdex.CreateBurnRuleFinder()
		if err != nil {
			return errors.Wrap(err, "updateParser")
		}
		p.Parsers.BurnParser = parser.NewParser(burnRule, dex.NewBurnMapper())
	}

	return nil
}
