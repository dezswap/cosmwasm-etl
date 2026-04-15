package columbusv2

import (
	"strconv"

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
	lpPairAddrs   map[string]string
	flaggedAssets map[string]bool
}

var _ dex.TargetApp = &terraswapApp{}

func New(repo dex.PairRepo, logger logging.Logger, c configs.ParserDexConfig) (dex.TargetApp, error) {
	finder, err := columbusv2.CreateCreatePairRuleFinder(c.FactoryAddress)
	if err != nil {
		return nil, errors.Wrap(err, "NewApp")
	}

	taxPaymentRule, err := columbusv2.CreateTaxPaymentRuleFinder()
	if err != nil {
		return nil, errors.Wrap(err, "NewApp")
	}

	parsers := dex.PairParsers{
		CreatePairParser: parser.NewParser(finder, dex.NewFactoryMapper()),
		PairActionParser: nil,
		InitialProvide:   nil,
		WasmTransfer:     nil,
		Transfer:         nil,
		TaxPaymentParser: parser.NewParser(taxPaymentRule, dex.NewTaxPaymentMapper()),
	}

	pairs, err := repo.GetPairs()
	if err != nil {
		return nil, errors.Wrap(err, "columbusv2.New")
	}

	lpPairAddrs := make(map[string]string)
	for _, p := range pairs {
		lpPairAddrs[p.LpAddr] = p.ContractAddr
	}

	return &terraswapApp{repo, &parsers, dex.DexMixin{}, pairs, lpPairAddrs, make(map[string]bool)}, nil
}

func (p *terraswapApp) ParseTxs(tx parser.RawTx, height uint64) ([]dex.ParsedTx, error) {
	txDtos := []dex.ParsedTx{}
	createPairTxs, err := p.Parsers.CreatePairParser.Parse(tx.LogResults, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "columbusv2.terraswapApp.ParseTxs")
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

	pairTxs := []*dex.ParsedTx{}
	wasmTxs := []*dex.ParsedTx{}
	taxTxs := []*dex.ParsedTx{}
	transferTxs := []*dex.ParsedTx{}
	burnTxs := []*dex.ParsedTx{}
	for _, raw := range tx.LogResults {
		if !pdex.ParsableRules[string(raw.Type)] {
			continue
		}
		ptxs, err := p.Parsers.PairActionParser.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "columbusv2.terraswapApp.ParseTxs")
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
			return nil, errors.Wrap(err, "columbusv2.terraswapApp.ParseTxs")
		}
		wasmTxs = append(wasmTxs, wtxs...)

		tTxs, err := p.Parsers.TaxPaymentParser.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "columbusv2.terraswapApp.ParseTxs")
		}
		taxTxs = append(taxTxs, tTxs...)

		if raw.Type == eventlog.TransferType {
			// event log messages are not sorted well
			// bug tx: C51473267BEF98BAE991C19AD8A5EFF6370BC64B63ACB68190170095C1AE0ABE
			filter := []string{
				columbusv2.SortedTransferAmountKey, columbusv2.SortedTransferRecipientKey, columbusv2.SortedTransferSenderKey,
			}
			attrs, err := eventlog.SortAttributes(raw.Attributes, filter)
			if err != nil {
				return nil, errors.Wrap(err, "columbusv2.terraswapApp.ParseTxs")
			}
			raw.Attributes = *attrs
		}
		transfers, err := p.Parsers.Transfer.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "columbusv2.terraswapApp.ParseTxs")
		}
		transferTxs = append(transferTxs, transfers...)

		burns, err := p.Parsers.BurnParser.Parse(eventlog.LogResults{raw}, dex.ParsedTx{Hash: tx.Hash, Timestamp: tx.Timestamp})
		if err != nil {
			return nil, errors.Wrap(err, "columbusv2.terraswapApp.ParseTxs")
		}
		burnTxs = append(burnTxs, burns...)
	}

	// Originally, tax_amount in wasm was used to deduct the tax(parser.dex.terraswap.columbusv2.pairMapper.swapMatchedToParsedTx).
	// tax_payment log replaced tax_amount attribute for more detail, so it will be used to deduct here instead.
	// e.g. 2B99CFA6D1FB1029A28DCCABD753B5F43517B89CE31BF855235D47C16A7D2FB0
	if len(taxTxs) > 0 {
		pairAddrs := p.getPairAddrs(pairTxs)
		pairTransferTxs := p.getPairTransfers(pairAddrs, transferTxs)
		pairTxs = p.deductTaxFromPairTxs(taxTxs, pairTransferTxs, pairTxs)
	}

	for _, ptx := range pairTxs {
		ptx.Sender = tx.Sender
		txDtos = append(txDtos, *ptx)
	}

	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, wasmTxs)...)
	txDtos = append(txDtos, p.RemoveDuplicatedTxs(pairTxs, transferTxs)...)
	txDtos = append(txDtos, dex.CollectLpBurnTxs(burnTxs, p.lpPairAddrs)...)

	return txDtos, nil
}

func (p *terraswapApp) getPairAddrs(pairTxs []*dex.ParsedTx) []string {
	seen := make(map[string]bool)
	addrs := []string{}
	for _, t := range pairTxs {
		if !seen[t.ContractAddr] {
			seen[t.ContractAddr] = true
			addrs = append(addrs, t.ContractAddr)
		}
	}
	return addrs
}

// getPairTransfers returns transferTxs whose ContractAddr is in pairAddrs.
func (p *terraswapApp) getPairTransfers(pairAddrs []string, transferTxs []*dex.ParsedTx) []*dex.ParsedTx {
	pairAddrSet := make(map[string]bool, len(pairAddrs))
	for _, addr := range pairAddrs {
		pairAddrSet[addr] = true
	}

	pairTransferTxs := []*dex.ParsedTx{}
	for _, trf := range transferTxs {
		if pairAddrSet[trf.ContractAddr] {
			pairTransferTxs = append(pairTransferTxs, trf)
		}
	}
	return pairTransferTxs
}

// deductTaxFromPairTxs adjusts pairTxs amounts by deducting tax.
// Matching rule: pairTransferTx(net) + taxTx(tax) == pairTx(gross)
// e.g. pairTx=-1000, tax=10 -> net=-990; confirmed by pairTransferTx=-990
func (p *terraswapApp) deductTaxFromPairTxs(taxTxs, pairTransferTxs, pairTxs []*dex.ParsedTx) []*dex.ParsedTx {
	// pre-index pairTxs by (contractAddr, assetIdx, absAmount) for O(1) lookup
	type pairKey struct {
		addr      string
		assetIdx  int
		absAmount int64
	}
	pairTxIdx := make(map[pairKey]int)
	for i, t := range pairTxs {
		for j, asset := range t.Assets {
			if asset.Amount == "" {
				continue
			}
			amount, _ := strconv.ParseInt(asset.Amount, 10, 64)
			if amount < 0 {
				amount = -amount
			}
			pairTxIdx[pairKey{t.ContractAddr, j, amount}] = i
		}
	}

	// queue tax amounts by asset addr (consumed one by one)
	taxQueue := make(map[string][]int64)
	for _, t := range taxTxs {
		amount, _ := strconv.ParseInt(t.Assets[0].Amount, 10, 64)
		taxQueue[t.Assets[0].Addr] = append(taxQueue[t.Assets[0].Addr], amount)
	}

	for _, tx := range pairTransferTxs {
		// pick the non-empty asset; each transfer has only one side
		var netAsset dex.Asset
		var assetIdx int
		if tx.Assets[0].Amount != "" {
			netAsset, assetIdx = tx.Assets[0], 0
		} else {
			netAsset, assetIdx = tx.Assets[1], 1
		}

		taxes := taxQueue[netAsset.Addr]
		if len(taxes) == 0 {
			continue
		}

		netAmount, _ := strconv.ParseInt(netAsset.Amount, 10, 64)
		if netAmount < 0 {
			netAmount = -netAmount
		}

		for j, taxAmount := range taxes {
			key := pairKey{tx.ContractAddr, assetIdx, netAmount + taxAmount} // gross = net + tax
			if i, ok := pairTxIdx[key]; ok {
				pairTxs[i].Assets[assetIdx].Amount = netAsset.Amount
				delete(pairTxIdx, key)
				taxQueue[netAsset.Addr] = append(taxes[:j], taxes[j+1:]...)
				break
			}
		}
	}

	return pairTxs
}

func (p *terraswapApp) IsValidationExceptionCandidate(contractAddress string) bool {
	return p.flaggedAssets[contractAddress]
}

func (p *terraswapApp) UpdateParsers(tokenExceptions map[string]bool, height uint64) error {
	pairFilter := make(map[string]bool)
	for k := range p.pairs {
		pairFilter[k] = true
	}

	pairFinder, err := columbusv2.CreatePairCommonRulesFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.PairActionParser = parser.NewParser(pairFinder, &pairMapper{pairSet: p.pairs})

	initialProvideFinder, err := pdex.CreatePairInitialProvideRuleFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.InitialProvide = parser.NewParser(initialProvideFinder, dex.NewInitialProvideMapper())

	wasmTransferFinder, err := columbusv2.CreateWasmCommonTransferRuleFinder(pairFilter)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.WasmTransfer = parser.NewParser(
		wasmTransferFinder,
		dex.NewWasmTransferMapper(
			pdex.WasmTransferCw20AddrKey,
			p.pairs,
			p.flaggedAssets,
			tokenExceptions,
		),
	)

	transferRule, err := columbusv2.CreateSortedTransferRuleFinder(nil)
	if err != nil {
		return errors.Wrap(err, "updateParsers")
	}
	p.Parsers.Transfer = parser.NewParser(transferRule, dex.NewTransferMapper(p.pairs))

	// burn parser - to collect and parse LP burn event
	{
		burnRule, err := columbusv2.CreateBurnRuleFinder()
		if err != nil {
			return errors.Wrap(err, "updateParser")
		}
		p.Parsers.BurnParser = parser.NewParser(burnRule, dex.NewBurnMapper())
	}

	return nil
}
