package starfleit

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	sf "github.com/dezswap/cosmwasm-etl/pkg/rules/starfleit"
	"github.com/pkg/errors"
)

// runner for terraswap
type starfleitApp struct {
	parser.PairRepo
	Parsers *parser.PairParsers
	parser.DexMixin
	chainId string
}

var _ parser.TargetApp = &starfleitApp{}

func New(repo parser.PairRepo, logger logging.Logger, c configs.ParserConfig, chainId string) (parser.TargetApp, error) {
	finder, err := sf.CreateCreatePairRuleFinder(c.ChainId)
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

	return &starfleitApp{repo, parsers, parser.DexMixin{}, chainId}, nil
}

func (p *starfleitApp) ParseTxs(tx parser.RawTx, height uint64) ([]parser.ParsedTx, error) {
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
	wasmTxs := []*parser.ParsedTx{}
	transferTxs := []*parser.ParsedTx{}
	for _, raw := range tx.LogResults {
		ptxs, err := p.Parsers.PairActionParser.Parse(tx.Hash, tx.Timestamp, eventlog.LogResults{raw})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		pairTxs = append(pairTxs, ptxs...)

		// find transfer from user
		wtxs, err := p.Parsers.WasmTransfer.Parse(tx.Hash, tx.Timestamp, eventlog.LogResults{raw})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		wasmTxs = append(wasmTxs, wtxs...)

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
	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, wasmTxs)...)
	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, transferTxs)...)

	return txDtos, nil
}

func (p *starfleitApp) updateParsers(pairs map[string]parser.Pair, height uint64) error {
	pairFilter := make(map[string]bool)
	for k := range pairs {
		pairFilter[k] = true
	}

	pairFinder, err := sf.CreatePairAllRulesFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}

	pairMapper, err := pairMapperBy(p.chainId, height, pairs)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.PairActionParser = parser.NewParser(pairFinder, pairMapper)

	wasmTransferFinder, err := sf.CreateWasmCommonTransferRuleFinder()
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.WasmTransfer = parser.NewParser(wasmTransferFinder, &wasmTransferMapper{pairSet: pairs})

	transferRule, err := sf.CreateTransferRuleFinder()
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.Transfer = parser.NewParser(transferRule, &transferMapper{pairSet: pairs})
	return nil
}
