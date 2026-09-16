package conx

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	pdex "github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/conx"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
)

type appImpl struct {
	dex.PairRepo
	Parsers *dex.PairParsers
	dex.DexMixin
	chainId string

	// state
	pairs       map[string]dex.Pair
	lpPairAddrs map[string]string
}

var _ dex.TargetApp = &appImpl{}

func New(repo dex.PairRepo, _ logging.Logger, c configs.ParserDexConfig) (dex.TargetApp, error) {
	finder, err := conx.CreateCreatePairRuleFinder(c.FactoryAddress)
	if err != nil {
		return nil, errors.Wrap(err, "conx.New")
	}

	parsers := &dex.PairParsers{
		CreatePairParser: parser.NewParser(finder, &createPairMapper{}),
		PairActionParser: nil,
		InitialProvide:   nil,
		WasmTransfer:     nil,
		Transfer:         nil,
	}

	pairs, err := repo.GetPairs()
	if err != nil {
		return nil, errors.Wrap(err, "conx.New")
	}

	lpPairAddrs := make(map[string]string)
	for _, p := range pairs {
		lpPairAddrs[p.LpAddr] = p.ContractAddr
	}

	return &appImpl{repo, parsers, dex.DexMixin{}, c.ChainId, pairs, lpPairAddrs}, nil
}

func (p *appImpl) ParseTxs(tx parser.RawTx, height uint64) ([]dex.ParsedTx, error) {
	txDtos := []dex.ParsedTx{}
	partialQuarantine := dex.NewPartialQuarantineRecorder(tx, height)
	createPairTxs, err := p.Parsers.CreatePairParser.Parse(tx.LogResults, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp}, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "conx.ParseTxs create_pair tx_hash=%s", tx.Hash)
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
	if len(createPairTxs) > 0 {
		if err := p.updatePairScopedParsers(height); err != nil {
			return nil, errors.Wrapf(err, "conx.ParseTxs refresh_pair_parsers tx_hash=%s", tx.Hash)
		}
	}

	pairTxs := []*dex.ParsedTx{}
	wasmTransferTxs := []*dex.ParsedTx{}
	transferTxs := []*dex.ParsedTx{}
	burnTxs := []*dex.ParsedTx{}
	for _, raw := range tx.LogResults {
		ptxs, err := p.Parsers.PairActionParser.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrapf(err, "conx.ParseTxs pair_action tx_hash=%s", tx.Hash)
		}
		pairTxs = append(pairTxs, ptxs...)

		// find initial provide to a pair
		if p.HasProvide(ptxs) {
			ipTxs, err := p.Parsers.InitialProvide.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
			if err != nil {
				return nil, errors.Wrapf(err, "conx.ParseTxs initial_provide tx_hash=%s", tx.Hash)
			}
			pairTxs = append(pairTxs, ipTxs...)
		}

		wtxs, err := p.Parsers.WasmTransfer.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			wrapped := errors.Wrapf(err, "conx.ParseTxs wasm_transfer tx_hash=%s", tx.Hash)
			if !partialQuarantine.Record("wasm_transfer", wrapped) {
				return nil, wrapped
			}
		}
		wasmTransferTxs = append(wasmTransferTxs, wtxs...)

		if raw.Type == eventlog.TransferType {
			// event log messages are not sorted well
			// bug tx: 8C4CF31E736AAC477F61704ECCBBB5A5ABBAA2A8A12576EFAA9F8546F1F60FE2 (cube_47-5)
			sorted, err := pdex.NormalizeTransferAttrs(raw.Attributes)
			if err != nil {
				return nil, errors.Wrapf(err, "conx.ParseTxs sort_transfer_attrs tx_hash=%s", tx.Hash)
			}
			raw.Attributes = sorted
		}
		transfers, err := p.Parsers.Transfer.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp}, tx.Sender)
		if err != nil {
			return nil, errors.Wrapf(err, "conx.ParseTxs transfer tx_hash=%s", tx.Hash)
		}
		transferTxs = append(transferTxs, transfers...)

		burns, err := p.Parsers.BurnParser.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrapf(err, "conx.ParseTxs burn tx_hash=%s", tx.Hash)
		}
		burnTxs = append(burnTxs, burns...)
	}

	for _, ptx := range pairTxs {
		ptx.Sender = tx.Sender
		txDtos = append(txDtos, *ptx)
	}

	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, append(wasmTransferTxs, transferTxs...))...)
	txDtos = append(txDtos, dex.CollectLpBurnTxs(burnTxs, p.lpPairAddrs)...)

	if err := partialQuarantine.Err(txDtos); err != nil {
		return txDtos, err
	}

	return txDtos, nil
}

func (p *appImpl) IsValidationExceptionCandidate(contractAddress string) bool {
	return false
}

func (p *appImpl) UpdateParsers(tokenExceptions map[string]bool, height uint64) error {
	if err := p.updatePairScopedParsers(height); err != nil {
		return err
	}

	// wasm transfer parser
	{
		wasmTransferFinder, err := conx.CreateWasmCommonTransferRuleFinder()
		if err != nil {
			return errors.Wrap(err, "updateParsers")
		}
		p.Parsers.WasmTransfer = parser.NewParser(
			wasmTransferFinder,
			&wasmTransferMapper{
				mixin:           transferMapperMixin{pdex.MapperMixin{}},
				pairSet:         p.pairs,
				tokenExceptions: tokenExceptions,
			},
		)
	}

	// transfer parser
	{
		transferRule, err := pdex.CreateTransferRuleFinder(nil)
		if err != nil {
			return errors.Wrap(err, "updateParsers")
		}
		p.Parsers.Transfer = parser.NewParser(transferRule, &transferMapper{pairSet: p.pairs})
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

// pair filters are built from a snapshot of p.pairs, so they must be rebuilt
// whenever a pair is created mid-height.
func (p *appImpl) updatePairScopedParsers(height uint64) error {
	pairFilter := make(map[string]bool)
	for k := range p.pairs {
		pairFilter[k] = true
	}

	pairFinder, err := conx.CreatePairAllRulesFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updatePairScopedParsers")
	}

	pairMapper, err := pairMapperBy(p.chainId, height, p.pairs)
	if err != nil {
		return errors.Wrap(err, "updatePairScopedParsers")
	}
	p.Parsers.PairActionParser = parser.NewParser(pairFinder, pairMapper)

	initialProvideFinder, err := pdex.CreatePairInitialProvideRuleFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updatePairScopedParsers")
	}
	p.Parsers.InitialProvide = parser.NewParser(initialProvideFinder, dex.NewInitialProvideMapper())

	return nil
}
