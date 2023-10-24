package dezswap

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	ds "github.com/dezswap/cosmwasm-etl/pkg/rules/dezswap"
	"github.com/pkg/errors"
)

// runner for terraswap
type dezswapApp struct {
	parser.PairRepo
	Parsers *parser.PairParsers
	parser.DexMixin
	chainId string
}

var _ parser.TargetApp = &dezswapApp{}

func New(repo parser.PairRepo, logger logging.Logger, c configs.ParserConfig, chainId string) (parser.TargetApp, error) {
	finder, err := ds.CreateCreatePairRuleFinder(c.ChainId)
	if err != nil {
		return nil, errors.Wrap(err, "NewApp")
	}

	parsers := &parser.PairParsers{
		CreatePairParser: parser.NewParser(finder, &createPairMapper{}),
		PairActionParser: nil,
		InitialProvide:   nil,
		WasmTransfer:     nil,
		Transfer:         nil,
	}

	return &dezswapApp{repo, parsers, parser.DexMixin{}, chainId}, nil
}

func (p *dezswapApp) ParseTxs(tx parser.RawTx, height uint64) ([]parser.ParsedTx, error) {
	pairs, err := p.GetPairs()
	if err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}

	txDtos := []parser.ParsedTx{}
	createPairTxs, err := p.Parsers.CreatePairParser.Parse(tx.Hash, tx.Timestamp, tx.LogResults, nil)
	if err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}
	for _, ctx := range createPairTxs {
		pairs[ctx.ContractAddr] = parser.Pair{
			ContractAddr: ctx.ContractAddr,
			LpAddr:       ctx.LpAddr,
			Assets:       []string{ctx.Assets[0].Addr, ctx.Assets[1].Addr},
		}
		ctx.Sender = tx.Sender
		txDtos = append(txDtos, *ctx)
	}

	if err := p.updateParsers(pairs, height); err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}

	pairTxs := []*parser.ParsedTx{}
	wasmTransferTxs := []*parser.ParsedTx{}
	transferTxs := []*parser.ParsedTx{}
	initialProvideTxs := []*parser.ParsedTx{}
	for _, raw := range tx.LogResults {
		ptxs, err := p.Parsers.PairActionParser.Parse(tx.Hash, tx.Timestamp, eventlog.LogResults{raw})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		pairTxs = append(pairTxs, ptxs...)

		// find initial provide to a pair
		ipTxs, err := p.Parsers.InitialProvide.Parse(tx.Hash, tx.Timestamp, eventlog.LogResults{raw})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		initialProvideTxs = append(initialProvideTxs, ipTxs...)

		wtxs, err := p.Parsers.WasmTransfer.Parse(tx.Hash, tx.Timestamp, eventlog.LogResults{raw})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		wasmTransferTxs = append(wasmTransferTxs, wtxs...)

		transfers, err := p.Parsers.Transfer.Parse(tx.Hash, tx.Timestamp, eventlog.LogResults{raw})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		transferTxs = append(transferTxs, transfers...)
	}
	for _, ptx := range pairTxs {
		ptx.Sender = tx.Sender
		txDtos = append(txDtos, *ptx)
	}

	for _, ipTx := range initialProvideTxs {
		ipTx.Sender = tx.Sender
		txDtos = append(txDtos, *ipTx)
	}

	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, wasmTransferTxs)...)
	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, transferTxs)...)

	return txDtos, nil
}

func (p *dezswapApp) updateParsers(pairs map[string]parser.Pair, height uint64) error {
	pairFilter := make(map[string]bool)
	for k := range pairs {
		pairFilter[k] = true
	}

	pairFinder, err := ds.CreatePairAllRulesFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}

	pairMapper, err := pairMapperBy(p.chainId, height, pairs)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.PairActionParser = parser.NewParser(pairFinder, pairMapper)

	initialProvideFinder, err := ds.CreatePairInitialProvideRuleFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.InitialProvide = parser.NewParser(initialProvideFinder, &initialProvideMapper{})

	wasmTransferFinder, err := ds.CreateWasmCommonTransferRuleFinder()
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.WasmTransfer = parser.NewParser(wasmTransferFinder, &wasmTransferMapper{pairSet: pairs})

	transferRule, err := ds.CreateTransferRuleFinder()
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.Transfer = parser.NewParser(transferRule, &transferMapper{pairSet: pairs})
	return nil
}
