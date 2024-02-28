package phoenix

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/phoenix"
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

func New(repo p_dex.PairRepo, logger logging.Logger, c configs.ParserConfig) (p_dex.TargetApp, error) {
	finder, err := phoenix.CreateCreatePairRuleFinder(c.FactoryAddress)
	if err != nil {
		return nil, errors.Wrap(err, "NewApp")
	}

	parsers := p_dex.PairParsers{
		CreatePairParser: parser.NewParser[p_dex.ParsedTx](finder, &createPairMapper{}),
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
		return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
	}

	txDtos := []p_dex.ParsedTx{}
	createPairTxs, err := p.Parsers.CreatePairParser.Parse(tx.LogResults, p_dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
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
		return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
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
			return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
		}
		pairTxs = append(pairTxs, ptxs...)

		wtxs, err := p.Parsers.WasmTransfer.Parse(eventlog.LogResults{raw}, p_dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
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
		transfers, err := p.Parsers.Transfer.Parse(eventlog.LogResults{raw}, p_dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
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

func (p *terraswapApp) updateParsers(pairs map[string]p_dex.Pair) error {
	pairFilter := make(map[string]bool)
	for k := range pairs {
		pairFilter[k] = true
	}

	pairFinder, err := phoenix.CreatePairCommonRulesFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.PairActionParser = parser.NewParser[p_dex.ParsedTx](pairFinder, &pairMapper{pairSet: pairs})

	wasmTransferFinder, err := phoenix.CreateWasmCommonTransferRuleFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.WasmTransfer = parser.NewParser[p_dex.ParsedTx](wasmTransferFinder, &wasmCommonTransferMapper{pairSet: pairs})

	transferRule, err := phoenix.CreateSortedTransferRuleFinder(nil)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.Transfer = parser.NewParser[p_dex.ParsedTx](transferRule, &transferMapper{pairSet: pairs})
	return nil
}
