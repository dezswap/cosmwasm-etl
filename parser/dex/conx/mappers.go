package conx

import (
	"fmt"
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	pdex "github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/conx"
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
	mixin           transferMapperMixin
	pairSet         map[string]dex.Pair
	tokenExceptions map[string]bool
}

// match implements mapper
func (m *createPairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.CheckResult(res, conx.CreatePairMatchedLen); err != nil {
		return nil, errors.Wrap(err, "createPairMapper.MatchedToParsedTx")
	}
	m.SortResult(res)
	assets := strings.Split(res[conx.FactoryPairIdx].Value, "-")
	if len(assets) != 2 {
		msg := fmt.Sprintf("expected assets length(%d)", 2)
		return nil, errors.New(msg)
	}

	return []*dex.ParsedTx{{
		Type:         dex.CreatePair,
		Sender:       "",
		ContractAddr: res[conx.FactoryPairAddrIdx].Value,
		Assets: [2]dex.Asset{
			{Addr: assets[0]},
			{Addr: assets[1]},
		},
		LpAddr:   res[conx.FactoryLpAddrIdx].Value,
		LpAmount: "",
	}}, nil
}

// match implements mapper
func (m *wasmTransferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	m.mixin.SortResult(res)

	var cw20Addr string
	if len(res) > conx.WasmCommonTransferCw20AddrIdx {
		cw20Addr = res[conx.WasmCommonTransferCw20AddrIdx].Value
		if m.tokenExceptions[cw20Addr] {
			return nil, nil
		}
	}

	if err := m.mixin.CheckResult(res, conx.WasmCommonTransferMatchedLen); err != nil {
		return nil, errors.Wrap(err, "wasmTransferMapper.MatchedToParsedTx")
	}
	action := res[conx.WasmCommonTransferActionIdx]

	switch action.Value {
	case conx.WasmTransferAction:
		return m.transferMatchedToParsedTx(res, optionals...)
	case conx.WasmTransferFromAction:
		return m.transferFromMatchedToParsedTx(res, optionals...)
	}

	msg := fmt.Sprintf("expected action(%s) or (%s)", conx.WasmTransferAction, conx.WasmTransferFromAction)
	return nil, errors.New(msg)
}

func (m *wasmTransferMapper) transferMatchedToParsedTx(res eventlog.MatchedResult, _ ...interface{}) ([]*dex.ParsedTx, error) {
	// Validate expected keys exist (filter out CW1155 transfers)
	// see. https://explorer.xpla.io/mainnet/address/xpla18xsgaqx66wkljvnjcu57pfwq4dtjv4gay662j2pnhy46lvmzqycsxdtz54
	if len(res) <= conx.WasmTransferToIdx {
		return nil, nil
	}
	if res[conx.WasmTransferFromIdx].Key != "from" {
		return nil, nil
	}

	from := res[conx.WasmTransferFromIdx].Value
	to := res[conx.WasmTransferToIdx].Value

	pair, fromPair, err := m.mixin.pairBy(m.pairSet, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "no pair") {
			return nil, nil
		}

		return nil, errors.Wrap(err, "wasmTransferMapper.transferMatchedToParsedTx")
	}

	return m.matchedToParsedTx(
		pair, from, to, res[conx.WasmCommonTransferCw20AddrIdx].Value, res[conx.WasmTransferAmountIdx].Value, fromPair,
	)
}

func (m *wasmTransferMapper) transferFromMatchedToParsedTx(res eventlog.MatchedResult, _ ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.mixin.CheckResult(res, conx.WasmTransferFromMatchedLen); err != nil {
		return nil, errors.Wrap(err, "wasmTransferMapper.transferFromMatchedToParsedTx")
	}
	from := res[conx.WasmTransferFromFromIdx].Value
	to := res[conx.WasmTransferFromToIdx].Value

	pair, fromPair, err := m.mixin.pairBy(m.pairSet, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "no pair") {
			return nil, nil
		}

		return nil, errors.Wrap(err, "wasmTransferMapper.transferFromMatchedToParsedTx")
	}

	return m.matchedToParsedTx(
		pair, from, to, res[conx.WasmCommonTransferCw20AddrIdx].Value, res[conx.WasmTransferFromAmountIdx].Value, fromPair,
	)
}

// match implements mapper
func (m *transferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.mixin.CheckResult(res, conx.TransferMatchedLen); err != nil {
		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}
	matchMap, err := eventlog.ResultToItemMapForKeys(
		res,
		pdex.TransferSenderKey,
		pdex.TransferRecipientKey,
		pdex.TransferAmountKey,
	)
	if err != nil {
		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}
	from := matchMap[pdex.TransferSenderKey].Value
	if from == "" {
		from = dex.TransferFallbackSender(optionals...)
	}
	to := matchMap[pdex.TransferRecipientKey].Value

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
	amountValue := matchMap[pdex.TransferAmountKey].Value
	if amountValue == "" {
		return nil, errors.New("empty amount")
	}
	amountStrs := strings.Split(amountValue, ",")
	for _, amountStr := range amountStrs {
		if amountStr == "" {
			continue
		}
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
