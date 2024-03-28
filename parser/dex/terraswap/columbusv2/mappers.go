package columbusv2

import (
	"fmt"
	"math/big"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/columbusv2"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/terra"

	"github.com/pkg/errors"
)

var _ parser.Mapper[dex.ParsedTx] = &pairMapper{}

type pairMapper struct {
	pairSet map[string]dex.Pair
}

// match implements mapper
func (m *pairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	pair, ok := m.pairSet[res[columbusv2.PairAddrIdx].Value]
	if !ok {
		msg := fmt.Sprintf("pairMapper.MatchedToParsedTx no pair(%s)", res[columbusv2.PairAddrIdx].Value)
		return nil, errors.New(msg)
	}

	action := columbusv2.PairAction(res[columbusv2.PairActionIdx].Value)
	switch action {
	case columbusv2.SwapAction:
		return m.swapMatchedToParsedTx(res, pair)
	case columbusv2.ProvideAction:
		return m.provideMatchedToParsedTx(res, pair)
	case columbusv2.WithdrawAction:
		return m.withdrawMatchedToParsedTx(res, pair)
	}

	msg := fmt.Sprintf("action must be (%s, %s, %s)", columbusv2.SwapAction, columbusv2.ProvideAction, columbusv2.WithdrawAction)
	return nil, errors.New(msg)
}

func (m *pairMapper) swapMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	matchMap, err := eventlog.ResultToItemMap(res)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	offerAsset := matchMap[columbusv2.PairSwapOfferAssetKey].Value
	offerIdx := 0
	if pair.Assets[1] == offerAsset {
		offerIdx = 1
	}
	returnIdx := (offerIdx + 1) % 2

	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}

	assets[offerIdx].Amount = matchMap[columbusv2.PairSwapOfferAmountKey].Value
	assets[returnIdx].Amount = fmt.Sprintf("-%s", matchMap[columbusv2.PairSwapReturnAmountKey].Value)

	tax, ok := matchMap[columbusv2.PairSwapTaxAmountKey]
	if ok {
		if assets[returnIdx].Amount, err = dex.AmountAdd(assets[returnIdx].Amount, tax.Value); err != nil {
			return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
		}
	}

	return []*dex.ParsedTx{{
		Type:             dex.Swap,
		ContractAddr:     matchMap[columbusv2.PairAddrKey].Value,
		Sender:           matchMap[columbusv2.PairSwapSenderKey].Value,
		Assets:           assets,
		CommissionAmount: matchMap[columbusv2.PairSwapCommissionAmountKey].Value,
		Meta:             nil,
	}}, nil
}

func (m *pairMapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	matchMap, err := eventlog.ResultToItemMap(res)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	assets, err := dex.GetAssetsFromAssetsString(matchMap[columbusv2.PairProvideAssetsKey].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}
	if assets[0].Addr != pair.Assets[0] {
		assets = []dex.Asset{assets[1], assets[0]}
	}

	meta := map[string]interface{}{}
	refundItem, ok := matchMap[columbusv2.PairProvideRefundAssetKey]
	if ok {
		refundAssets, err := dex.GetAssetsFromAssetsString(refundItem.Value)
		if err != nil {
			return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
		}
		if refundAssets[0].Addr != pair.Assets[0] {
			refundAssets = []dex.Asset{refundAssets[1], refundAssets[0]}
		}

		assets, err = m.applyRefundAsset(assets, refundAssets)
		if err != nil {
			return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
		}
		meta[columbusv2.PairProvideRefundAssetKey] = refundAssets
	}

	return []*dex.ParsedTx{{
		Type:         dex.Provide,
		ContractAddr: matchMap[columbusv2.PairAddrKey].Value,
		Sender:       matchMap[columbusv2.PairProvideSenderKey].Value,
		Assets:       [2]dex.Asset(assets),
		LpAddr:       pair.LpAddr,
		LpAmount:     matchMap[columbusv2.PairProvideShareKey].Value,
	}}, nil
}

func (m *pairMapper) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	matchMap, err := eventlog.ResultToItemMap(res)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := dex.GetAssetsFromAssetsString(matchMap[columbusv2.PairWithdrawRefundAssetsKey].Value)
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
		ContractAddr: matchMap[columbusv2.PairAddrKey].Value,
		Sender:       matchMap[columbusv2.PairWithdrawSenderKey].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     matchMap[columbusv2.PairWithdrawWithdrawShareKey].Value,
	}}, nil

}

// Apply refund asset to provided asset for cw20
// cw20 token is not refunded in provide event, it is transferred deducted amount to pair once.
// wasm message shows users requested amount rather than actual provided amount.
func (m *pairMapper) applyRefundAsset(provide []dex.Asset, refund []dex.Asset) (applied []dex.Asset, err error) {
	toBigInt := func(amount string) (*big.Int, error) {
		amountBigInt, ok := big.NewInt(0).SetString(amount, 10)
		if !ok {
			return nil, errors.New("invalid amount")
		}
		return amountBigInt, nil
	}
	applied = make([]dex.Asset, len(provide))
	copy(applied, provide)

	for idx := range provide {
		if provide[idx].Addr != refund[idx].Addr {
			return nil, errors.New("provide and refund assets must be same order")
		}
		if !terra.IsCw20(provide[idx].Addr) {
			continue
		}

		amount, err := toBigInt(provide[idx].Amount)
		if err != nil {
			return nil, err
		}
		refundAmount, err := toBigInt(refund[idx].Amount)
		if err != nil {
			return nil, err
		}
		applied[idx].Amount = amount.Sub(amount, refundAmount).String()
	}

	return applied, nil
}
