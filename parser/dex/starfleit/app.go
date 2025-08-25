package starfleit

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	sf "github.com/dezswap/cosmwasm-etl/pkg/dex/starfleit"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
)

// runner for terraswap
type starfleitApp struct {
	dex.PairRepo
	Parsers *dex.PairParsers
	dex.DexMixin
	chainId string

	// state
	pairs map[string]dex.Pair
}

var _ dex.TargetApp = &starfleitApp{}

func New(repo dex.PairRepo, logger logging.Logger, c configs.ParserDexConfig, chainId string) (dex.TargetApp, error) {
	finder, err := sf.CreateCreatePairRuleFinder(c.ChainId)
	if err != nil {
		return nil, errors.Wrap(err, "NewApp")
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
		return nil, errors.Wrap(err, "starfleit.New")
	}

	return &starfleitApp{repo, parsers, dex.DexMixin{}, chainId, pairs}, nil
}

func (p *starfleitApp) ParseTxs(tx parser.RawTx, height uint64) ([]dex.ParsedTx, error) {
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
		ctx.Sender = tx.Sender
		txDtos = append(txDtos, *ctx)
	}

	if err := p.updateParsers(p.pairs, height); err != nil {
		return nil, errors.Wrap(err, "parseTxs")
	}

	pairTxs := []*dex.ParsedTx{}
	wasmTxs := []*dex.ParsedTx{}
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

		// find transfer from user
		wtxs, err := p.Parsers.WasmTransfer.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "parseTxs")
		}
		wasmTxs = append(wasmTxs, wtxs...)

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
	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, wasmTxs)...)
	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, transferTxs)...)

	return txDtos, nil
}

func (p *starfleitApp) IsValidationExceptionCandidate(contractAddress string) bool {
	return false
}

func (p *starfleitApp) updateParsers(pairs map[string]dex.Pair, height uint64) error {
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
	p.Parsers.PairActionParser = parser.NewParser[dex.ParsedTx](pairFinder, pairMapper)

	initialProvideFinder, err := sf.CreatePairInitialProvideRuleFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.InitialProvide = parser.NewParser[dex.ParsedTx](initialProvideFinder, dex.NewInitialProvideMapper())

	wasmTransferFinder, err := sf.CreateWasmCommonTransferRuleFinder()
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.WasmTransfer = parser.NewParser[dex.ParsedTx](wasmTransferFinder, &wasmTransferMapper{pairSet: pairs})

	transferRule, err := sf.CreateTransferRuleFinder()
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.Transfer = parser.NewParser[dex.ParsedTx](transferRule, &transferMapper{pairSet: pairs})
	return nil
}
