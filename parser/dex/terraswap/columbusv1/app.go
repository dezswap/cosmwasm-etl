package columbusv1

import (
	"github.com/dezswap/cosmwasm-etl/parser"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	cv1 "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/columbusv1"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
)

// runner for terraswap
type terraswapApp struct {
	p_dex.PairRepo
	Parsers *p_dex.PairParsers
	p_dex.DexMixin
}

var _ p_dex.TargetApp = &terraswapApp{}

func New(repo p_dex.PairRepo, logger logging.Logger, c configs.ParserDexConfig) (p_dex.TargetApp, error) {
	finder, err := cv1.CreateCreatePairRuleFinder(c.FactoryAddress)
	if err != nil {
		return nil, errors.Wrap(err, "NewApp")
	}

	parsers := p_dex.PairParsers{
		CreatePairParser: parser.NewParser[p_dex.ParsedTx](finder, p_dex.NewFactoryMapper()),
		PairActionParser: nil,
		InitialProvide:   nil,
		WasmTransfer:     nil,
		Transfer:         nil,
	}

	return &terraswapApp{repo, &parsers, p_dex.DexMixin{}}, nil
}

func (p *terraswapApp) ParseTxs(tx parser.RawTx, height uint64) ([]p_dex.ParsedTx, error) {
	pairs, err := p.GetPairs()
	if err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}

	txDtos := []p_dex.ParsedTx{}
	createPairTxs, err := p.Parsers.CreatePairParser.Parse(tx.LogResults, p_dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}
	for _, ctx := range createPairTxs {
		pairs[ctx.ContractAddr] = p_dex.Pair{
			ContractAddr: ctx.ContractAddr,
			LpAddr:       ctx.LpAddr,
			Assets:       []string{ctx.Assets[0].Addr, ctx.Assets[1].Addr},
		}
		ctx.Sender = tx.Sender
		txDtos = append(txDtos, *ctx)
	}

	if err := p.updateParsers(pairs); err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}

	pairTxs := []*p_dex.ParsedTx{}
	wasmTxs := []*p_dex.ParsedTx{}
	transferTxs := []*p_dex.ParsedTx{}
	for _, raw := range tx.LogResults {
		if !dex.ParsableRules[string(raw.Type)] {
			continue
		}
		ptxs, err := p.Parsers.PairActionParser.Parse(eventlog.LogResults{raw}, p_dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		pairTxs = append(pairTxs, ptxs...)

		wtxs, err := p.Parsers.WasmTransfer.Parse(eventlog.LogResults{raw}, p_dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		wasmTxs = append(wasmTxs, wtxs...)

		transfers, err := p.Parsers.Transfer.Parse(eventlog.LogResults{raw}, p_dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
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

func (p *terraswapApp) updateParsers(pairs map[string]p_dex.Pair) error {
	pairFilter := make(map[string]bool)
	for k := range pairs {
		pairFilter[k] = true
	}

	pairFinder, err := cv1.CreatePairCommonRulesFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "createParsers")
	}
	p.Parsers.PairActionParser = parser.NewParser[p_dex.ParsedTx](pairFinder, &pairMapper{pairSet: pairs})

	wasmTransferFinder, err := cv1.CreateWasmCommonTransferRuleFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "createParsers")
	}
	p.Parsers.WasmTransfer = parser.NewParser[p_dex.ParsedTx](wasmTransferFinder, p_dex.NewWasmTransferMapper(dex.WasmTransferLegacyCw20AddrKey, pairs))

	transferRule, err := cv1.CreateTransferRuleFinder(nil)
	if err != nil {
		return errors.Wrap(err, "createParsers")
	}
	p.Parsers.Transfer = parser.NewParser[p_dex.ParsedTx](transferRule, p_dex.NewTransferMapper(pairs))
	return nil
}
