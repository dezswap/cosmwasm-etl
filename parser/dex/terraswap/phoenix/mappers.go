package phoenix

import (
	"fmt"
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/phoenix"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"

	"github.com/pkg/errors"
)

var _ parser.Mapper[dex.ParsedTx] = &createPairMapper{}
var _ parser.Mapper[dex.ParsedTx] = &pairMapper{}
var _ parser.Mapper[dex.ParsedTx] = &transferMapper{}
var _ parser.Mapper[dex.ParsedTx] = &wasmCommonTransferMapper{}

type mapperMixin struct{}

type createPairMapper struct{ mapperMixin }
type pairMapper struct {
	mixin   mapperMixin
	pairSet map[string]dex.Pair
}

type transferMapper struct {
	pairSet map[string]dex.Pair
}
type wasmCommonTransferMapper struct {
	mixin   mapperMixin
	pairSet map[string]dex.Pair
}

// match implements mapper
func (m *createPairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.mapperMixin.checkResult(res, phoenix.CreatePairMatchedLen); err != nil {
		return nil, errors.Wrap(err, "createPairMapper.MatchedToParsedTx")
	}

	assets := strings.Split(res[phoenix.FactoryPairIdx].Value, "-")
	if len(assets) != 2 {
		msg := fmt.Sprintf("expected assets length(%d)", 2)
		return nil, errors.New(msg)
	}

	return []*dex.ParsedTx{{
		Type:         dex.CreatePair,
		Sender:       "",
		ContractAddr: res[phoenix.FactoryPairAddrIdx].Value,
		Assets: [2]dex.Asset{
			{Addr: assets[0]},
			{Addr: assets[1]},
		},
		LpAddr:   res[phoenix.FactoryLpAddrIdx].Value,
		LpAmount: "",
	}}, nil
}

// match implements mapper
func (m *pairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	pair, ok := m.pairSet[res[phoenix.PairAddrIdx].Value]
	if !ok {
		msg := fmt.Sprintf("pairMapper.MatchedToParsedTx no pair(%s)", res[phoenix.PairAddrIdx].Value)
		return nil, errors.New(msg)
	}

	action := phoenix.PairAction(res[phoenix.PairActionIdx].Value)
	switch action {
	case phoenix.SwapAction:
		return m.swapMatchedToParsedTx(res, pair)
	case phoenix.ProvideAction:
		return m.provideMatchedToParsedTx(res, pair)
	case phoenix.WithdrawAction:
		return m.withdrawMatchedToParsedTx(res, pair)
	}

	msg := fmt.Sprintf("action must be (%s, %s, %s)", phoenix.SwapAction, phoenix.ProvideAction, phoenix.WithdrawAction)
	return nil, errors.New(msg)
}

func (m *pairMapper) swapMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.mixin.checkResult(res); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	matchMap, err := resultToMap(res)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	offerAsset := matchMap[phoenix.PairSwapOfferAssetKey].Value
	offerIdx := 0
	if pair.Assets[1] == offerAsset {
		offerIdx = 1
	}
	returnIdx := (offerIdx + 1) % 2

	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}

	assets[offerIdx].Amount = matchMap[phoenix.PairSwapOfferAmountKey].Value
	assets[returnIdx].Amount = fmt.Sprintf("-%s", matchMap[phoenix.PairSwapReturnAmountKey].Value)

	return []*dex.ParsedTx{{
		Type:             dex.Swap,
		ContractAddr:     matchMap[phoenix.PairAddrKey].Value,
		Sender:           matchMap[phoenix.PairSwapSenderKey].Value,
		Assets:           assets,
		CommissionAmount: matchMap[phoenix.PairSwapCommissionAmountKey].Value,
		Meta:             nil,
	}}, nil
}

func (m *pairMapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.mixin.checkResult(res); err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	matchMap, err := resultToMap(res)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	assets, err := dex.GetAssetsFromAssetsString(matchMap[phoenix.PairProvideAssetsKey].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []dex.Asset{assets[1], assets[0]}
	}

	return []*dex.ParsedTx{{
		Type:         dex.Provide,
		ContractAddr: matchMap[phoenix.PairAddrKey].Value,
		Sender:       matchMap[phoenix.PairProvideSenderKey].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     matchMap[phoenix.PairProvideShareKey].Value,
	}}, nil
}

func (m *pairMapper) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.mixin.checkResult(res); err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	matchMap, err := resultToMap(res)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := dex.GetAssetsFromAssetsString(matchMap[phoenix.PairWithdrawRefundAssetsKey].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}
	for idx := range assets {
		assets[idx].Amount = fmt.Sprintf("-%s", assets[idx].Amount)
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []dex.Asset{assets[1], assets[0]}
	}

	return []*dex.ParsedTx{{
		Type:         dex.Withdraw,
		ContractAddr: matchMap[phoenix.PairAddrKey].Value,
		Sender:       matchMap[phoenix.PairWithdrawSenderKey].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     matchMap[phoenix.PairWithdrawWithdrawShareKey].Value,
	}}, nil

}

// match implements mapper
func (m *wasmCommonTransferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.mixin.checkResult(res); err != nil {
		return nil, errors.Wrap(err, "wasmCommonTransferMapper.MatchedToParsedTx")
	}

	matchMap, err := resultToMap(res)
	if err != nil {
		return nil, errors.Wrap(err, "wasmCommonTransferMapper.MatchedToParsedTx")
	}

	fp, fromPair := m.pairSet[matchMap[phoenix.WasmTransferFromKey].Value]
	tp, toPair := m.pairSet[matchMap[phoenix.WasmTransferToKey].Value]
	if fromPair && toPair {
		msg := fmt.Sprintf("cannot be both from and to, see the tx, result(%v)", res)
		return nil, errors.New(msg)
	}

	if !fromPair && !toPair {
		return nil, nil
	}

	pair := fp
	if toPair {
		pair = tp
	}

	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	meta := make(map[string]interface{})

	target := matchMap[phoenix.WasmTransferCw20AddrKey].Value
	idx := dex.IndexOf(pair.Assets, target)
	if idx == -1 {
		meta[target] = matchMap[phoenix.WasmTransferAmountKey].Value
	} else {
		for _, item := range res {
			if item.Key == "amount" {
				assets[idx].Amount = item.Value
			}
		}
	}

	if fromPair {
		assets[idx].Amount = fmt.Sprintf("-%s", assets[idx].Amount)
	}

	return []*dex.ParsedTx{{
		Type:         dex.Transfer,
		Sender:       matchMap[phoenix.WasmTransferFromKey].Value,
		ContractAddr: pair.ContractAddr,
		Assets:       assets,
		Meta:         meta,
	}}, nil

}

// match implements mapper
func (m *transferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	matchMap, err := resultToMap(res)
	if err != nil {
		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}

	fp, fromPair := m.pairSet[matchMap[phoenix.SortedTransferSenderKey].Value]
	tp, toPair := m.pairSet[matchMap[phoenix.SortedTransferRecipientKey].Value]

	if fromPair && toPair {
		msg := fmt.Sprintf("cannot be both from and to, see the tx, result(%v)", res)
		return nil, errors.New(msg)
	}

	if !fromPair && !toPair {
		return nil, nil
	}

	pair := fp
	if toPair {
		pair = tp
	}

	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	meta := make(map[string]interface{})

	amountStrs := strings.Split(matchMap[phoenix.SortedTransferAmountKey].Value, ",")
	for _, amountStr := range amountStrs {
		asset, err := dex.GetAssetFromAmountAssetString(amountStr)
		if err != nil {
			return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
		}
		idx := dex.IndexOf(pair.Assets, asset.Addr)
		if idx == -1 {
			meta[asset.Addr] = asset.Amount
		} else {
			assets[idx] = asset
		}
		if fromPair {
			assets[idx].Amount = fmt.Sprintf("-%s", assets[idx].Amount)
		}
	}

	return []*dex.ParsedTx{{
		Type:         dex.Transfer,
		Sender:       matchMap[phoenix.SortedTransferSenderKey].Value,
		ContractAddr: pair.ContractAddr,
		Assets:       assets,
		LpAddr:       "",
		LpAmount:     "",
		Meta:         meta,
	}}, nil
}

func (*mapperMixin) checkResult(res eventlog.MatchedResult, expectedLen ...int) error {
	if len(expectedLen) > 0 && len(res) != expectedLen[0] {
		msg := fmt.Sprintf("expected results length(%d)", expectedLen)
		return errors.New(msg)
	}
	for i, r := range res {
		if r.Value == "" {
			msg := fmt.Sprintf("matched result(%d) must not be empty", i)
			return errors.New(msg)
		}
	}
	return nil
}

func resultToMap(res eventlog.MatchedResult) (map[string]eventlog.MatchedItem, error) {
	m := make(map[string]eventlog.MatchedItem)
	for _, r := range res {
		if _, ok := m[r.Key]; ok {
			return nil, errors.New(fmt.Sprintf("duplicated key(%s)", r.Key))
		}
		m[r.Key] = r
	}
	return m, nil
}
