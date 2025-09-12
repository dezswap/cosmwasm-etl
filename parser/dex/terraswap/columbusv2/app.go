package columbusv2

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	pdex "github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/columbusv2"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
)

// runner for terraswap
type terraswapApp struct {
	dex.PairRepo
	Parsers *dex.PairParsers
	dex.DexMixin

	// state
	pairs         map[string]dex.Pair
	flaggedAssets map[string]bool
}

var _ dex.TargetApp = &terraswapApp{}

func New(repo dex.PairRepo, logger logging.Logger, c configs.ParserDexConfig) (dex.TargetApp, error) {
	finder, err := columbusv2.CreateCreatePairRuleFinder(c.FactoryAddress)
	if err != nil {
		return nil, errors.Wrap(err, "NewApp")
	}

	parsers := dex.PairParsers{
		CreatePairParser: parser.NewParser(finder, dex.NewFactoryMapper()),
		PairActionParser: nil,
		InitialProvide:   nil,
		WasmTransfer:     nil,
		Transfer:         nil,
	}

	pairs, err := repo.GetPairs()
	if err != nil {
		return nil, errors.Wrap(err, "columbusv2.New")
	}

	return &terraswapApp{repo, &parsers, dex.DexMixin{}, pairs, make(map[string]bool)}, nil
}

func (p *terraswapApp) ParseTxs(tx parser.RawTx, height uint64) ([]dex.ParsedTx, error) {

	txDtos := []dex.ParsedTx{}
	createPairTxs, err := p.Parsers.CreatePairParser.Parse(tx.LogResults, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
	}
	for _, ctx := range createPairTxs {
		p.pairs[ctx.ContractAddr] = dex.Pair{
			ContractAddr: ctx.ContractAddr,
			LpAddr:       ctx.LpAddr,
			Assets:       []string{ctx.Assets[0].Addr, ctx.Assets[1].Addr},
		}
		ctx.Sender = tx.Sender
		txDtos = append(txDtos, *ctx)
	}

	if err := p.updateParsers(p.pairs); err != nil {
		return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
	}

	pairTxs := []*dex.ParsedTx{}
	wasmTxs := []*dex.ParsedTx{}
	transferTxs := []*dex.ParsedTx{}
	for _, raw := range tx.LogResults {
		if !pdex.ParsableRules[string(raw.Type)] {
			continue
		}
		ptxs, err := p.Parsers.PairActionParser.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
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
			return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
		}
		wasmTxs = append(wasmTxs, wtxs...)

		if raw.Type == eventlog.TransferType {
			// event log messages are not sorted well
			// bug tx: C51473267BEF98BAE991C19AD8A5EFF6370BC64B63ACB68190170095C1AE0ABE
			filter := []string{
				columbusv2.SortedTransferAmountKey, columbusv2.SortedTransferRecipientKey, columbusv2.SortedTransferSenderKey,
			}
			attrs, err := eventlog.SortAttributes(raw.Attributes, filter)
			if err != nil {
				return nil, errors.Wrap(err, "phoenix.terraswapApp.ParseTxs")
			}
			raw.Attributes = *attrs
		}
		transfers, err := p.Parsers.Transfer.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
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

func (p *terraswapApp) IsValidationExceptionCandidate(contractAddress string) bool {
	return p.flaggedAssets[contractAddress]
}

func (p *terraswapApp) updateParsers(pairs map[string]dex.Pair) error {
	pairFilter := make(map[string]bool)
	for k := range pairs {
		pairFilter[k] = true
	}

	pairFinder, err := columbusv2.CreatePairCommonRulesFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.PairActionParser = parser.NewParser[dex.ParsedTx](pairFinder, &pairMapper{pairSet: pairs})

	initialProvideFinder, err := pdex.CreatePairInitialProvideRuleFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.InitialProvide = parser.NewParser[dex.ParsedTx](initialProvideFinder, dex.NewInitialProvideMapper())

	wasmTransferFinder, err := columbusv2.CreateWasmCommonTransferRuleFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.WasmTransfer = parser.NewParser[dex.ParsedTx](wasmTransferFinder, dex.NewWasmTransferMapper(pdex.WasmTransferCw20AddrKey, pairs, p.flaggedAssets))

	transferRule, err := columbusv2.CreateSortedTransferRuleFinder(nil)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.Transfer = parser.NewParser[dex.ParsedTx](transferRule, dex.NewTransferMapper(pairs))
	return nil
}
