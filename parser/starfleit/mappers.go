package starfleit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	sf "github.com/dezswap/cosmwasm-etl/pkg/rules/starfleit"
	"github.com/pkg/errors"
)

var _ parser.Mapper = &createPairMapper{}
var _ parser.Mapper = &transferMapper{}
var _ parser.Mapper = &wasmTransferMapper{}

type mapperMixin struct{}

type createPairMapper struct{ mapperMixin }

type transferMapperMixin struct {
	mapperMixin
}
type transferMapper struct {
	mixin   transferMapperMixin
	pairSet map[string]parser.Pair
}
type wasmTransferMapper struct {
	mixin   transferMapperMixin
	pairSet map[string]parser.Pair
}

// match implements mapper
func (m *createPairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	if err := m.mapperMixin.checkResult(res, sf.CreatePairMatchedLen); err != nil {
		return nil, errors.Wrap(err, "createPairMapper.MatchedToParsedTx")
	}
	sortResult(res)
	assets := strings.Split(res[sf.FactoryPairIdx].Value, "-")
	if len(assets) != 2 {
		msg := fmt.Sprintf("expected assets length(%d)", 2)
		return nil, errors.New(msg)
	}

	return &parser.ParsedTx{
		Type:         parser.CreatePair,
		Sender:       "",
		ContractAddr: res[sf.FactoryPairAddrIdx].Value,
		Assets: []parser.Asset{
			{Addr: assets[0]},
			{Addr: assets[1]},
		},
		LpAddr:   res[sf.FactoryLpAddrIdx].Value,
		LpAmount: "",
	}, nil
}

// match implements mapper
func (m *wasmTransferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	sortResult(res)
	action := res[sf.WasmCommonTransferActionIdx]

	switch action.Value {
	case sf.WasmV1TransferAction:
		return m.v1MatchedToParsedTx(res, optionals...)
	case sf.WasmV2TransferAction:
		return m.v2MatchedToParsedTx(res, optionals...)
	}

	msg := fmt.Sprintf("expected action(%s) or (%s)", sf.WasmV1TransferAction, sf.WasmV2TransferAction)
	return nil, errors.New(msg)
}

func (m *wasmTransferMapper) v1MatchedToParsedTx(res eventlog.MatchedResult, _ ...interface{}) (*parser.ParsedTx, error) {
	if err := m.mixin.checkResult(res, sf.WasmV1TransferMatchedLen); err != nil {
		return nil, errors.Wrap(err, "transferMapper.v1MatchedToParsedTx")
	}
	from := res[sf.WasmTransferFromIdx].Value
	to := res[sf.WasmTransferToIdx].Value

	pair, fromPair, err := m.mixin.pairBy(m.pairSet, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "no pair") {
			return nil, nil
		}

		return nil, errors.Wrap(err, "wasmTransferMapper.v1MatchedToParsedTx")
	}

	return m.matchedToParsedTx(
		pair, from, to, res[sf.WasmCommonTransferCw20AddrIdx].Value, res[sf.WasmTransferAmountIdx].Value, fromPair,
	)
}

func (m *wasmTransferMapper) v2MatchedToParsedTx(res eventlog.MatchedResult, _ ...interface{}) (*parser.ParsedTx, error) {
	if err := m.mixin.checkResult(res, sf.WasmV2TransferMatchedLen); err != nil {
		return nil, errors.Wrap(err, "wasmTransferMapper.v2MatchedToParsedTx")
	}
	from := res[sf.WasmV2TransferFromIdx].Value
	to := res[sf.WasmV2TransferToIdx].Value

	pair, fromPair, err := m.mixin.pairBy(m.pairSet, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "no pair") {
			return nil, nil
		}

		return nil, errors.Wrap(err, "wasmTransferMapper.v2MatchedToParsedTx")
	}

	return m.matchedToParsedTx(
		pair, from, to, res[sf.WasmCommonTransferCw20AddrIdx].Value, res[sf.WasmV2TransferAmountIdx].Value, fromPair,
	)
}

// match implements mapper
func (m *transferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	if err := m.mixin.checkResult(res, sf.TransferMatchedLen); err != nil {
		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}
	from := res[sf.TransferSenderIdx].Value
	to := res[sf.TransferRecipientIdx].Value

	pair, fromPair, err := m.mixin.pairBy(m.pairSet, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "no pair") {
			return nil, nil
		}

		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}

	assets := []parser.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	amountStrs := strings.Split(res[sf.TransferAmountIdx].Value, ",")
	if len(amountStrs) == 0 {
		return nil, errors.New("empty amount or wrong format(amounts separated by ,)")
	}
	for _, amountStr := range amountStrs {
		asset, err := parser.GetAssetFromAmountAssetString(amountStr)
		if err != nil {
			return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
		}
		idx := parser.IndexOf(pair.Assets, asset.Addr)
		if idx == -1 {
			msg := fmt.Sprintf("wrong asset(%s), pair(%s) assets(%s)", asset.Addr, pair.ContractAddr, pair.Assets)
			return nil, errors.New(msg)
		}
		if fromPair {
			asset.Amount = fmt.Sprintf("-%s", asset.Amount)
		}
		assets[idx] = asset
	}

	return &parser.ParsedTx{
		Type:         parser.Transfer,
		Sender:       res[sf.TransferSenderIdx].Value,
		ContractAddr: pair.ContractAddr,
		Assets:       assets,
		LpAddr:       "",
		LpAmount:     "",
		Meta: map[string]interface{}{
			"recipient": to,
		},
	}, nil
}

func (*mapperMixin) checkResult(res eventlog.MatchedResult, expectedLen int) error {
	if len(res) != expectedLen {
		msg := fmt.Sprintf("expected results length(%d)", expectedLen)
		return errors.New(msg)
	}
	return nil
}

func (*wasmTransferMapper) matchedToParsedTx(pair *parser.Pair, from, to, targetToken, amount string, isFromPair bool) (*parser.ParsedTx, error) {
	assets := []parser.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	idx := parser.IndexOf(pair.Assets, targetToken)
	if idx == -1 {
		msg := fmt.Sprintf("wrong asset(%s), pair(%s) assets(%s)", targetToken, pair.ContractAddr, pair.Assets)
		return nil, errors.New(msg)
	}
	if isFromPair {
		assets[idx].Amount = "-" + amount
	} else {
		assets[idx].Amount = amount
	}

	return &parser.ParsedTx{
		Type:         parser.Transfer,
		Sender:       from,
		ContractAddr: pair.ContractAddr,
		Assets:       assets,
		LpAddr:       "",
		LpAmount:     "",
		Meta: map[string]interface{}{
			"recipient": to,
		},
	}, nil
}

func (*transferMapperMixin) pairBy(pairSet map[string]parser.Pair, from, to string) (*parser.Pair, bool, error) {
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
