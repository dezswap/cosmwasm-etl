package columbusv1

import (
	"fmt"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	cv1 "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/columbusv1"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"

	"github.com/pkg/errors"
)

var _ parser.Mapper[dex.ParsedTx] = &pairMapper{}

type mapperMixin struct{}

type pairMapper struct {
	mixin   mapperMixin
	pairSet map[string]dex.Pair
}

// match implements mapper

// match implements mapper
func (m *pairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	pair, ok := m.pairSet[res[cv1.PairAddrIdx].Value]
	if !ok {
		msg := fmt.Sprintf("pairMapper.MatchedToParsedTx no pair(%s)", res[cv1.PairAddrIdx].Value)
		return nil, errors.New(msg)
	}

	action := cv1.PairAction(res[cv1.PairActionIdx].Value)
	switch action {
	case cv1.SwapAction:
		return m.swapMatchedToParsedTx(res, pair)
	case cv1.ProvideAction:
		return m.provideMatchedToParsedTx(res, pair)
	case cv1.WithdrawAction:
		return m.withdrawMatchedToParsedTx(res, pair)
	}

	msg := fmt.Sprintf("action must be (%s, %s, %s)", cv1.SwapAction, cv1.ProvideAction, cv1.WithdrawAction)
	return nil, errors.New(msg)
}

func (m *pairMapper) swapMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	var err error
	if err = m.mixin.checkResult(res, cv1.PairSwapMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	offerAsset := res[cv1.PairSwapOfferAssetIdx].Value
	offerIdx := 0
	if pair.Assets[1] == offerAsset {
		offerIdx = 1
	}
	returnIdx := (offerIdx + 1) % 2

	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}

	assets[offerIdx].Amount = res[cv1.PairSwapOfferAmountIdx].Value
	assets[returnIdx].Amount = fmt.Sprintf("-%s", res[cv1.PairSwapReturnAmountIdx].Value)

	if assets[returnIdx].Amount, err = dex.AmountAdd(assets[returnIdx].Amount, res[cv1.PairSwapTaxAmountIdx].Value); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	return []*dex.ParsedTx{{
		Type:             dex.Swap,
		ContractAddr:     res[cv1.PairAddrIdx].Value,
		Assets:           assets,
		CommissionAmount: res[cv1.PairSwapCommissionAmountIdx].Value,
		Meta: map[string]interface{}{
			res[cv1.PairSwapTaxAmountIdx].Key: dex.Asset{
				Addr:   assets[returnIdx].Addr,
				Amount: res[cv1.PairSwapTaxAmountIdx].Value,
			},
		},
	}}, nil
}

func (m *pairMapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.mixin.checkResult(res, cv1.PairProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.PairProvideMatchedLen")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[cv1.PairProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []dex.Asset{assets[1], assets[0]}
	}

	return []*dex.ParsedTx{{
		Type:         dex.Provide,
		ContractAddr: res[cv1.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[cv1.PairProvideShareIdx].Value,
	}}, nil
}

func (m *pairMapper) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.mixin.checkResult(res, cv1.PairWithdrawMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[cv1.PairWithdrawRefundAssetsIdx].Value)
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
		ContractAddr: res[cv1.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{{Addr: assets[0].Addr, Amount: "0"}, {Addr: assets[1].Addr, Amount: "0"}},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[cv1.PairWithdrawWithdrawShareIdx].Value,
		Meta: map[string]interface{}{
			"withdraw_assets": assets,
		},
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
