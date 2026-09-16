package classicv1

import (
	"fmt"

	pdex "github.com/dezswap/cosmwasm-etl/pkg/dex"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra/classicv1"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"

	"github.com/pkg/errors"
)

var _ parser.Mapper[dex.ParsedTx] = &pairMapper{}

type pairMapper struct {
	mixin   pdex.MapperMixin
	pairSet map[string]dex.Pair
}

// match implements mapper

// match implements mapper
func (m *pairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	if len(res) < classicv1.PairCommonMatchedLen {
		msg := fmt.Sprintf("results length must bigger than %d", classicv1.PairCommonMatchedLen)
		return nil, errors.New(msg)
	}
	pair, ok := m.pairSet[res[classicv1.PairAddrIdx].Value]
	if !ok {
		msg := fmt.Sprintf("pairMapper.MatchedToParsedTx no pair(%s)", res[classicv1.PairAddrIdx].Value)
		return nil, errors.New(msg)
	}

	action := classicv1.PairAction(res[classicv1.PairActionIdx].Value)
	switch action {
	case classicv1.SwapAction:
		return m.swapMatchedToParsedTx(res, pair)
	case classicv1.ProvideAction:
		return m.provideMatchedToParsedTx(res, pair)
	case classicv1.WithdrawAction:
		return m.withdrawMatchedToParsedTx(res, pair)
	}

	msg := fmt.Sprintf("action must be (%s, %s, %s)", classicv1.SwapAction, classicv1.ProvideAction, classicv1.WithdrawAction)
	return nil, errors.New(msg)
}

func (m *pairMapper) swapMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	var err error
	if err = m.mixin.CheckResult(res, classicv1.PairSwapMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	offerAsset := res[classicv1.PairSwapOfferAssetIdx].Value
	offerIdx := 0
	if pair.Assets[1] == offerAsset {
		offerIdx = 1
	}
	returnIdx := (offerIdx + 1) % 2

	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}

	assets[offerIdx].Amount = res[classicv1.PairSwapOfferAmountIdx].Value
	assets[returnIdx].Amount = fmt.Sprintf("-%s", res[classicv1.PairSwapReturnAmountIdx].Value)

	if assets[returnIdx].Amount, err = dex.AmountAdd(assets[returnIdx].Amount, res[classicv1.PairSwapTaxAmountIdx].Value); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	return []*dex.ParsedTx{{
		Type:             dex.Swap,
		ContractAddr:     res[classicv1.PairAddrIdx].Value,
		Assets:           assets,
		CommissionAmount: res[classicv1.PairSwapCommissionAmountIdx].Value,
		Meta: map[string]interface{}{
			res[classicv1.PairSwapTaxAmountIdx].Key: dex.Asset{
				Addr:   assets[returnIdx].Addr,
				Amount: res[classicv1.PairSwapTaxAmountIdx].Value,
			},
		},
	}}, nil
}

func (m *pairMapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.mixin.CheckResult(res, classicv1.PairProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.PairProvideMatchedLen")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[classicv1.PairProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []dex.Asset{assets[1], assets[0]}
	}

	return []*dex.ParsedTx{{
		Type:         dex.Provide,
		ContractAddr: res[classicv1.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[classicv1.PairProvideShareIdx].Value,
	}}, nil
}

func (m *pairMapper) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.mixin.CheckResult(res, classicv1.PairWithdrawMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[classicv1.PairWithdrawRefundAssetsIdx].Value)
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
		ContractAddr: res[classicv1.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{{Addr: assets[0].Addr, Amount: "0"}, {Addr: assets[1].Addr, Amount: "0"}},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[classicv1.PairWithdrawWithdrawShareIdx].Value,
		Meta: map[string]interface{}{
			"withdraw_assets": assets,
		},
	}}, nil

}
