package dezswap

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	ds "github.com/dezswap/cosmwasm-etl/pkg/dex/dezswap"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/pkg/errors"
)

var _ parser.Mapper[dex.ParsedTx] = &createPairMapper{}
var _ parser.Mapper[dex.ParsedTx] = &transferMapper{}
var _ parser.Mapper[dex.ParsedTx] = &wasmTransferMapper{}

type mapperMixin struct{}

type createPairMapper struct{ mapperMixin }

type transferMapperMixin struct {
	mapperMixin
}
type transferMapper struct {
	mixin   transferMapperMixin
	pairSet map[string]dex.Pair
}
type wasmTransferMapper struct {
	mixin   transferMapperMixin
	pairSet map[string]dex.Pair
}

// match implements mapper
func (m *createPairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.mapperMixin.checkResult(res, ds.CreatePairMatchedLen); err != nil {
		return nil, errors.Wrap(err, "createPairMapper.MatchedToParsedTx")
	}
	sortResult(res)
	assets := strings.Split(res[ds.FactoryPairIdx].Value, "-")
	if len(assets) != 2 {
		msg := fmt.Sprintf("expected assets length(%d)", 2)
		return nil, errors.New(msg)
	}

	return []*dex.ParsedTx{{
		Type:         dex.CreatePair,
		Sender:       "",
		ContractAddr: res[ds.FactoryPairAddrIdx].Value,
		Assets: [2]dex.Asset{
			{Addr: assets[0]},
			{Addr: assets[1]},
		},
		LpAddr:   res[ds.FactoryLpAddrIdx].Value,
		LpAmount: "",
	}}, nil
}

// match implements mapper
func (m *wasmTransferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	sortResult(res)
	action := res[ds.WasmCommonTransferActionIdx]

	switch action.Value {
	case ds.WasmTransferAction:
		return m.transferMatchedToParsedTx(res, optionals...)
	case ds.WasmTransferFromAction:
		return m.transferFromMatchedToParsedTx(res, optionals...)
	}

	msg := fmt.Sprintf("expected action(%s) or (%s)", ds.WasmTransferAction, ds.WasmTransferFromAction)
	return nil, errors.New(msg)
}

func (m *wasmTransferMapper) transferMatchedToParsedTx(res eventlog.MatchedResult, _ ...interface{}) ([]*dex.ParsedTx, error) {
	from := res[ds.WasmTransferFromIdx].Value
	to := res[ds.WasmTransferToIdx].Value

	pair, fromPair, err := m.mixin.pairBy(m.pairSet, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "no pair") {
			return nil, nil
		}

		return nil, errors.Wrap(err, "wasmTransferMapper.transferMatchedToParsedTx")
	}

	return m.matchedToParsedTx(
		pair, from, to, res[ds.WasmCommonTransferCw20AddrIdx].Value, res[ds.WasmTransferAmountIdx].Value, fromPair,
	)
}

func (m *wasmTransferMapper) transferFromMatchedToParsedTx(res eventlog.MatchedResult, _ ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.mixin.checkResult(res, ds.WasmTransferFromMatchedLen); err != nil {
		return nil, errors.Wrap(err, "wasmTransferMapper.transferFromMatchedToParsedTx")
	}
	from := res[ds.WasmTransferFromFromIdx].Value
	to := res[ds.WasmTransferFromToIdx].Value

	pair, fromPair, err := m.mixin.pairBy(m.pairSet, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "no pair") {
			return nil, nil
		}

		return nil, errors.Wrap(err, "wasmTransferMapper.transferFromMatchedToParsedTx")
	}

	return m.matchedToParsedTx(
		pair, from, to, res[ds.WasmCommonTransferCw20AddrIdx].Value, res[ds.WasmTransferFromAmountIdx].Value, fromPair,
	)
}

// match implements mapper
func (m *transferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.mixin.checkResult(res, ds.TransferMatchedLen); err != nil {
		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}
	from := res[ds.TransferSenderIdx].Value
	to := res[ds.TransferRecipientIdx].Value

	pair, fromPair, err := m.mixin.pairBy(m.pairSet, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "no pair") {
			return nil, nil
		}

		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}

	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	amountStrs := strings.Split(res[ds.TransferAmountIdx].Value, ",")
	if len(amountStrs) == 0 {
		return nil, errors.New("empty amount or wrong format(amounts separated by ,)")
	}
	for _, amountStr := range amountStrs {
		asset, err := dex.GetAssetFromAmountAssetString(amountStr)
		if err != nil {
			return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
		}
		idx := dex.IndexOf(pair.Assets, asset.Addr)
		if idx == -1 {
			msg := fmt.Sprintf("wrong asset(%s), pair(%s) assets(%s)", asset.Addr, pair.ContractAddr, pair.Assets)
			return nil, errors.New(msg)
		}
		if fromPair {
			asset.Amount = fmt.Sprintf("-%s", asset.Amount)
		}
		assets[idx] = asset
	}

	return []*dex.ParsedTx{{
		Type:         dex.Transfer,
		Sender:       res[ds.TransferSenderIdx].Value,
		ContractAddr: pair.ContractAddr,
		Assets:       assets,
		LpAddr:       "",
		LpAmount:     "",
		Meta: map[string]interface{}{
			"recipient": to,
		},
	}}, nil
}

func (*mapperMixin) checkResult(res eventlog.MatchedResult, expectedLen int) error {
	if len(res) != expectedLen {
		msg := fmt.Sprintf("expected results length(%d)", expectedLen)
		return errors.New(msg)
	}
	return nil
}

func (*wasmTransferMapper) matchedToParsedTx(pair *dex.Pair, from, to, targetToken, amount string, isFromPair bool) ([]*dex.ParsedTx, error) {
	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	idx := dex.IndexOf(pair.Assets, targetToken)
	if idx == -1 {
		msg := fmt.Sprintf("wrong asset(%s), pair(%s) assets(%s)", targetToken, pair.ContractAddr, pair.Assets)
		return nil, errors.New(msg)
	}
	if isFromPair {
		assets[idx].Amount = "-" + amount
	} else {
		assets[idx].Amount = amount
	}

	return []*dex.ParsedTx{{
		Type:         dex.Transfer,
		Sender:       from,
		ContractAddr: pair.ContractAddr,
		Assets:       assets,
		LpAddr:       "",
		LpAmount:     "",
		Meta: map[string]interface{}{
			"recipient": to,
		},
	}}, nil
}

func (*transferMapperMixin) pairBy(pairSet map[string]dex.Pair, from, to string) (*dex.Pair, bool, error) {
	fromPair, fromOk := pairSet[from]
	toPair, toOk := pairSet[to]

	if !fromOk && !toOk {
		msg := fmt.Sprintf("transferMapperMixin.pairBy no pair for (%s, %s)", from, to)
		return nil, fromOk, errors.New(msg)
	}

	if fromOk {
		return &fromPair, fromOk, nil
	}

	return &toPair, fromOk, nil
}

// sortResult sorts the result by key split by "_contract_address"
// @param res will be sorted
func sortResult(res eventlog.MatchedResult) {
	const sortSplitter = "_contract_address"
	sort := func(from, to int) {
		target := res[from:to]
		sort.Slice(target, func(i, j int) bool {
			return target[i].Key < target[j].Key
		})
	}
	prev := 0
	for idx, v := range res {
		if v.Key == sortSplitter {
			sort(prev, idx)
			prev = idx
		}
	}
	sort(prev, len(res))
}
