package columbus_v1

import (
	"fmt"
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/columbus_v1"
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
	if err := m.mapperMixin.checkResult(res, columbus_v1.CreatePairMatchedLen); err != nil {
		return nil, errors.Wrap(err, "createPairMapper.MatchedToParsedTx")
	}

	assets := strings.Split(res[columbus_v1.FactoryPairIdx].Value, "-")
	if len(assets) != 2 {
		msg := fmt.Sprintf("expected assets length(%d)", 2)
		return nil, errors.New(msg)
	}

	return []*dex.ParsedTx{{
		Type:         dex.CreatePair,
		Sender:       "",
		ContractAddr: res[columbus_v1.FactoryPairAddrIdx].Value,
		Assets: [2]dex.Asset{
			{Addr: assets[0]},
			{Addr: assets[1]},
		},
		LpAddr:   res[columbus_v1.FactoryLpAddrIdx].Value,
		LpAmount: "",
	}}, nil
}

// match implements mapper
func (m *pairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	pair, ok := m.pairSet[res[columbus_v1.PairAddrIdx].Value]
	if !ok {
		msg := fmt.Sprintf("pairMapper.MatchedToParsedTx no pair(%s)", res[columbus_v1.PairAddrIdx].Value)
		return nil, errors.New(msg)
	}

	action := columbus_v1.PairAction(res[columbus_v1.PairActionIdx].Value)
	switch action {
	case columbus_v1.SwapAction:
		return m.swapMatchedToParsedTx(res, pair)
	case columbus_v1.ProvideAction:
		return m.provideMatchedToParsedTx(res, pair)
	case columbus_v1.WithdrawAction:
		return m.withdrawMatchedToParsedTx(res, pair)
	}

	msg := fmt.Sprintf("action must be (%s, %s, %s)", columbus_v1.SwapAction, columbus_v1.ProvideAction, columbus_v1.WithdrawAction)
	return nil, errors.New(msg)
}

func (m *pairMapper) swapMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	var err error
	if err = m.mixin.checkResult(res, columbus_v1.PairSwapMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	offerAsset := res[columbus_v1.PairSwapOfferAssetIdx].Value
	offerIdx := 0
	if pair.Assets[1] == offerAsset {
		offerIdx = 1
	}
	returnIdx := (offerIdx + 1) % 2

	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}

	assets[offerIdx].Amount = res[columbus_v1.PairSwapOfferAmountIdx].Value
	assets[returnIdx].Amount = fmt.Sprintf("-%s", res[columbus_v1.PairSwapReturnAmountIdx].Value)

	if assets[returnIdx].Amount, err = dex.AmountAdd(assets[returnIdx].Amount, res[columbus_v1.PairSwapTaxAmountIdx].Value); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	return []*dex.ParsedTx{{
		Type:             dex.Swap,
		ContractAddr:     res[columbus_v1.PairAddrIdx].Value,
		Assets:           assets,
		CommissionAmount: res[columbus_v1.PairSwapCommissionAmountIdx].Value,
		Meta: map[string]interface{}{
			res[columbus_v1.PairSwapTaxAmountIdx].Key: dex.Asset{
				Addr:   assets[returnIdx].Addr,
				Amount: res[columbus_v1.PairSwapTaxAmountIdx].Value,
			},
		},
	}}, nil
}

func (m *pairMapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.mixin.checkResult(res, columbus_v1.PairProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.PairProvideMatchedLen")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[columbus_v1.PairProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []dex.Asset{assets[1], assets[0]}
	}

	return []*dex.ParsedTx{{
		Type:         dex.Provide,
		ContractAddr: res[columbus_v1.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[columbus_v1.PairProvideShareIdx].Value,
	}}, nil
}

func (m *pairMapper) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.mixin.checkResult(res, columbus_v1.PairWithdrawMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[columbus_v1.PairWithdrawRefundAssetsIdx].Value)
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
		ContractAddr: res[columbus_v1.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{{Addr: assets[0].Addr, Amount: "0"}, {Addr: assets[1].Addr, Amount: "0"}},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[columbus_v1.PairWithdrawWithdrawShareIdx].Value,
		Meta: map[string]interface{}{
			"withdraw_assets": assets,
		},
	}}, nil

}

// match implements mapper
func (m *wasmCommonTransferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	if err := m.mixin.checkResult(res); err != nil {
		return nil, errors.Wrap(err, "wasmCommonTransferMapper.MatchedToParsedTx")
	}

	fp, fromPair := m.pairSet[res[columbus_v1.WasmTransferFromIdx].Value]
	tp, toPair := m.pairSet[res[columbus_v1.WasmTransferToIdx].Value]
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

	target := res[columbus_v1.WasmTransferCw20AddrIdx].Value
	idx := dex.IndexOf(pair.Assets, target)
	if idx == -1 {
		meta[target] = res[columbus_v1.WasmTransferAmountIdx].Value
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
		Sender:       res[columbus_v1.WasmTransferFromIdx].Value,
		ContractAddr: pair.ContractAddr,
		Assets:       assets,
		Meta:         meta,
	}}, nil

}

// match implements mapper
func (m *transferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	fp, fromPair := m.pairSet[res[columbus_v1.TransferSenderIdx].Value]
	tp, toPair := m.pairSet[res[columbus_v1.TransferRecipientIdx].Value]

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

	amountStrs := strings.Split(res[columbus_v1.TransferAmountIdx].Value, ",")
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
		Sender:       res[columbus_v1.TransferSenderIdx].Value,
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
