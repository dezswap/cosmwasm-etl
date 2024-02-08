package phoenix

import (
	"fmt"
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	ts "github.com/dezswap/cosmwasm-etl/pkg/rules/terraswap/phoenix"

	"github.com/pkg/errors"
)

var _ parser.Mapper = &createPairMapper{}
var _ parser.Mapper = &pairMapper{}
var _ parser.Mapper = &transferMapper{}
var _ parser.Mapper = &wasmCommonTransferMapper{}

type mapperMixin struct{}

type createPairMapper struct{ mapperMixin }
type pairMapper struct {
	mixin   mapperMixin
	pairSet map[string]parser.Pair
}

type transferMapper struct {
	pairSet map[string]parser.Pair
}
type wasmCommonTransferMapper struct {
	mixin   mapperMixin
	pairSet map[string]parser.Pair
}

// match implements mapper
func (m *createPairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	if err := m.mapperMixin.checkResult(res, ts.CreatePairMatchedLen); err != nil {
		return nil, errors.Wrap(err, "createPairMapper.MatchedToParsedTx")
	}

	assets := strings.Split(res[ts.FactoryPairIdx].Value, "-")
	if len(assets) != 2 {
		msg := fmt.Sprintf("expected assets length(%d)", 2)
		return nil, errors.New(msg)
	}

	return &parser.ParsedTx{
		Type:         parser.CreatePair,
		Sender:       "",
		ContractAddr: res[ts.FactoryPairAddrIdx].Value,
		Assets: [2]parser.Asset{
			{Addr: assets[0]},
			{Addr: assets[1]},
		},
		LpAddr:   res[ts.FactoryLpAddrIdx].Value,
		LpAmount: "",
	}, nil
}

// match implements mapper
func (m *pairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	pair, ok := m.pairSet[res[ts.PairAddrIdx].Value]
	if !ok {
		msg := fmt.Sprintf("pairMapper.MatchedToParsedTx no pair(%s)", res[ts.PairAddrIdx].Value)
		return nil, errors.New(msg)
	}

	action := ts.PairAction(res[ts.PairActionIdx].Value)
	switch action {
	case ts.SwapAction:
		return m.swapMatchedToParsedTx(res, pair)
	case ts.ProvideAction:
		return m.provideMatchedToParsedTx(res, pair)
	case ts.WithdrawAction:
		return m.withdrawMatchedToParsedTx(res, pair)
	}

	msg := fmt.Sprintf("action must be (%s, %s, %s)", ts.SwapAction, ts.ProvideAction, ts.WithdrawAction)
	return nil, errors.New(msg)
}

func (m *pairMapper) swapMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	var err error
	if err = m.mixin.checkResult(res, ts.PairSwapMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	offerAsset := res[ts.PairSwapOfferAssetIdx].Value
	offerIdx := 0
	if pair.Assets[1] == offerAsset {
		offerIdx = 1
	}
	returnIdx := (offerIdx + 1) % 2

	assets := [2]parser.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}

	assets[offerIdx].Amount = res[ts.PairSwapOfferAmountIdx].Value
	assets[returnIdx].Amount = fmt.Sprintf("-%s", res[ts.PairSwapReturnAmountIdx].Value)

	return &parser.ParsedTx{
		Type:             parser.Swap,
		ContractAddr:     res[ts.PairAddrIdx].Value,
		Sender:           res[ts.PairSwapSenderIdx].Value,
		Assets:           assets,
		CommissionAmount: res[ts.PairSwapCommissionAmountIdx].Value,
		Meta:             nil,
	}, nil
}

func (m *pairMapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.mixin.checkResult(res, ts.PairProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.PairProvideMatchedLen")
	}

	assets, err := parser.GetAssetsFromAssetsString(res[ts.PairProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []parser.Asset{assets[1], assets[0]}
	}

	return &parser.ParsedTx{
		Type:         parser.Provide,
		ContractAddr: res[ts.PairAddrIdx].Value,
		Sender:       res[ts.PairProvideSenderIdx].Value,
		Assets:       [2]parser.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[ts.PairProvideShareIdx].Value,
	}, nil
}

func (m *pairMapper) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.mixin.checkResult(res, ts.PairWithdrawMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := parser.GetAssetsFromAssetsString(res[ts.PairWithdrawRefundAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}
	for idx := range assets {
		assets[idx].Amount, err = parser.AmountMul(assets[idx].Amount, "0.9939285487078243")
		if err != nil {
			return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
		}
		assets[idx].Amount = fmt.Sprintf("-%s", assets[idx].Amount)
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []parser.Asset{assets[1], assets[0]}
	}

	return &parser.ParsedTx{
		Type:         parser.Withdraw,
		ContractAddr: res[ts.PairAddrIdx].Value,
		Sender:       res[ts.PairWithdrawSenderIdx].Value,
		Assets:       [2]parser.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[ts.PairWithdrawWithdrawShareIdx].Value,
	}, nil

}

// match implements mapper
func (m *wasmCommonTransferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	if err := m.mixin.checkResult(res); err != nil {
		return nil, errors.Wrap(err, "wasmCommonTransferMapper.MatchedToParsedTx")
	}

	fp, fromPair := m.pairSet[res[ts.WasmTransferFromIdx].Value]
	tp, toPair := m.pairSet[res[ts.WasmTransferToIdx].Value]
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

	assets := [2]parser.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	meta := make(map[string]interface{})

	target := res[ts.WasmTransferCw20AddrIdx].Value
	idx := parser.IndexOf(pair.Assets, target)
	if idx == -1 {
		meta[target] = res[ts.WasmTransferAmountIdx].Value
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

	return &parser.ParsedTx{
		Type:         parser.Transfer,
		Sender:       res[ts.WasmTransferFromIdx].Value,
		ContractAddr: res[ts.WasmTransferToIdx].Value,
		Assets:       assets,
		Meta:         meta,
	}, nil

}

// match implements mapper
func (m *transferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	fp, fromPair := m.pairSet[res[ts.TransferSenderIdx].Value]
	tp, toPair := m.pairSet[res[ts.TransferRecipientIdx].Value]

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

	assets := [2]parser.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	meta := make(map[string]interface{})

	amountStrs := strings.Split(res[ts.TransferAmountIdx].Value, ",")
	for _, amountStr := range amountStrs {
		asset, err := parser.GetAssetFromAmountAssetString(amountStr)
		if err != nil {
			return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
		}
		idx := parser.IndexOf(pair.Assets, asset.Addr)
		if idx == -1 {
			meta[asset.Addr] = asset.Amount
		} else {
			assets[idx] = asset
		}
		if fromPair {
			assets[idx].Amount = fmt.Sprintf("-%s", assets[idx].Amount)
		}
	}

	return &parser.ParsedTx{
		Type:         parser.Transfer,
		Sender:       res[ts.TransferSenderIdx].Value,
		ContractAddr: pair.ContractAddr,
		Assets:       assets,
		LpAddr:       "",
		LpAmount:     "",
		Meta:         meta,
	}, nil
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
