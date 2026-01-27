package starfleit

import (
	"fmt"
	"strings"

	pdex "github.com/dezswap/cosmwasm-etl/pkg/dex"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	sf "github.com/dezswap/cosmwasm-etl/pkg/dex/starfleit"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/pkg/errors"
)

var _ parser.Mapper[dex.ParsedTx] = &createPairMapper{}
var _ parser.Mapper[dex.ParsedTx] = &transferMapper{}
var _ parser.Mapper[dex.ParsedTx] = &wasmTransferMapper{}

type createPairMapper struct{ pdex.MapperMixin }

type transferMapperMixin struct {
	pdex.MapperMixin
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
	if err := m.CheckResult(res, sf.CreatePairMatchedLen); err != nil {
		return nil, errors.Wrap(err, "createPairMapper.MatchedToParsedTx")
	}
	m.SortResult(res)
	assets := strings.Split(res[sf.FactoryPairIdx].Value, "-")
	if len(assets) != 2 {
		msg := fmt.Sprintf("expected assets length(%d)", 2)
		return nil, errors.New(msg)
	}

	return []*dex.ParsedTx{{
		Type:         dex.CreatePair,
		Sender:       "",
		ContractAddr: res[sf.FactoryPairAddrIdx].Value,
		Assets: [2]dex.Asset{
			{Addr: assets[0]},
			{Addr: assets[1]},
		},
		LpAddr:   res[sf.FactoryLpAddrIdx].Value,
		LpAmount: "",
	}}, nil
}

// match implements mapper
func (m *wasmTransferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	m.mixin.SortResult(res)
	action := res[sf.WasmCommonTransferActionIdx]

	switch action.Value {
	case sf.WasmTransferAction:
		return m.v1MatchedToParsedTx(res, optionals...)
	case sf.WasmTransferFromAction:
		return m.v2MatchedToParsedTx(res, optionals...)
	}

	msg := fmt.Sprintf("expected action(%s) or (%s)", sf.WasmTransferAction, sf.WasmTransferFromAction)
	return nil, errors.New(msg)
}

func (m *wasmTransferMapper) v1MatchedToParsedTx(res eventlog.MatchedResult, _ ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.mixin.CheckResult(res, sf.WasmV1TransferMatchedLen); err != nil {
		return nil, errors.Wrap(err, "wasmTransferMapper.v1MatchedToParsedTx")
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

func (m *wasmTransferMapper) v2MatchedToParsedTx(res eventlog.MatchedResult, _ ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.mixin.CheckResult(res, sf.WasmV2TransferMatchedLen); err != nil {
		return nil, errors.Wrap(err, "wasmTransferMapper.v2MatchedToParsedTx")
	}
	from := res[sf.WasmTransferFromFromIdx].Value
	to := res[sf.WasmTransferFromToIdx].Value

	pair, fromPair, err := m.mixin.pairBy(m.pairSet, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "no pair") {
			return nil, nil
		}

		return nil, errors.Wrap(err, "wasmTransferMapper.v2MatchedToParsedTx")
	}

	return m.matchedToParsedTx(
		pair, from, to, res[sf.WasmCommonTransferCw20AddrIdx].Value, res[sf.WasmTransferFromAmountIdx].Value, fromPair,
	)
}

// match implements mapper
func (m *transferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.mixin.CheckResult(res, sf.TransferMatchedLen); err != nil {
		// skip empty value result
		// see. https://www.mintscan.io/fetchai/tx/C0B649ABBB5C04B8A01567C1E14635856E50CEA22B4A7BDA66F91D2CA8275BA2
		if errors.As(err, &pdex.ErrEmptyEventValue) {
			return []*dex.ParsedTx{}, nil
		}
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

	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	amountStrs := strings.Split(res[sf.TransferAmountIdx].Value, ",")
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
		Sender:       res[sf.TransferSenderIdx].Value,
		ContractAddr: pair.ContractAddr,
		Assets:       assets,
		LpAddr:       "",
		LpAmount:     "",
		Meta: map[string]interface{}{
			"recipient": to,
		},
	}}, nil
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
