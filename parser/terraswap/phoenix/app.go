package phoenix

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	ts "github.com/dezswap/cosmwasm-etl/pkg/rules/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/rules/terraswap/phoenix"
	"github.com/pkg/errors"
)

// runner for terraswap
type terraswapApp struct {
	parser.PairRepo
	Parsers *parser.PairParsers
	parser.DexMixin
}

var _ parser.TargetApp = &terraswapApp{}

func New(repo parser.PairRepo, logger logging.Logger, c configs.ParserConfig) (parser.TargetApp, error) {
	finder, err := phoenix.CreateCreatePairRuleFinder(c.FactoryAddress)
	if err != nil {
		return nil, errors.Wrap(err, "NewApp")
	}

	parsers := parser.PairParsers{
		CreatePairParser: parser.NewParser(finder, &createPairMapper{}),
		PairActionParser: nil,
		InitialProvide:   nil,
		WasmTransfer:     nil,
		Transfer:         nil,
	}

	return &terraswapApp{repo, &parsers, parser.DexMixin{}}, nil
}

func (p *terraswapApp) ParseTxs(tx parser.RawTx, height uint64) ([]parser.ParsedTx, error) {
	pairs, err := p.GetPairs()
	if err != nil {
		return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
	}

	txDtos := []parser.ParsedTx{}
	createPairTxs, err := p.Parsers.CreatePairParser.Parse(tx.Hash, tx.Timestamp, tx.LogResults, nil)
	if err != nil {
		return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
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

	if err := p.updateParsers(pairs); err != nil {
		return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
	}

	pairTxs := []*parser.ParsedTx{}
	wasmTxs := []*parser.ParsedTx{}
	transferTxs := []*parser.ParsedTx{}
	for _, raw := range tx.LogResults {
		if !ts.ParsableRules[string(raw.Type)] {
			continue
		}
		ptxs, err := p.Parsers.PairActionParser.Parse(tx.Hash, tx.Timestamp, eventlog.LogResults{raw})
		if err != nil {
			return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
		}
		pairTxs = append(pairTxs, ptxs...)

		wtxs, err := p.Parsers.WasmTransfer.Parse(tx.Hash, tx.Timestamp, eventlog.LogResults{raw})
		if err != nil {
			return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
		}
		wasmTxs = append(wasmTxs, wtxs...)

		if raw.Type == eventlog.TransferType {
			// event log messages are not sorted well
			// bug tx: C51473267BEF98BAE991C19AD8A5EFF6370BC64B63ACB68190170095C1AE0ABE
			filter := map[string]bool{
				phoenix.SortedTransferAmountKey: true, phoenix.SortedTransferRecipientKey: true, phoenix.SortedTransferSenderKey: true,
			}
			attrs, err := eventlog.SortAttributes(raw.Attributes, filter)
			if err != nil {
				return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
			}
			raw.Attributes = *attrs
		}
		transfers, err := p.Parsers.Transfer.Parse(tx.Hash, tx.Timestamp, eventlog.LogResults{raw})
		if err != nil {
			return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
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

func (p *terraswapApp) updateParsers(pairs map[string]parser.Pair) error {
	pairFilter := make(map[string]bool)
	for k := range pairs {
		pairFilter[k] = true
	}

	pairFinder, err := phoenix.CreatePairCommonRulesFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.PairActionParser = parser.NewParser(pairFinder, &pairMapper{pairSet: pairs})

	wasmTransferFinder, err := phoenix.CreateWasmCommonTransferRuleFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.WasmTransfer = parser.NewParser(wasmTransferFinder, &wasmCommonTransferMapper{pairSet: pairs})

	transferRule, err := phoenix.CreateSortedTransferRuleFinder(nil)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.Transfer = parser.NewParser(transferRule, &transferMapper{pairSet: pairs})
	return nil
}
