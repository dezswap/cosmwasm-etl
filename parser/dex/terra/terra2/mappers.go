package terra2

import (
	"fmt"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	pdex "github.com/dezswap/cosmwasm-etl/pkg/dex"
	dexterra "github.com/dezswap/cosmwasm-etl/pkg/dex/terra"
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
	if len(res) < dexterra.PairCommonMatchedLen {
		msg := fmt.Sprintf("results length must bigger than %d", dexterra.PairCommonMatchedLen)
		return nil, errors.New(msg)
	}
	pair, ok := m.pairSet[res[dexterra.PairAddrIdx].Value]
	if !ok {
		msg := fmt.Sprintf("pairMapper.MatchedToParsedTx no pair(%s)", res[dexterra.PairAddrIdx].Value)
		return nil, errors.New(msg)
	}

	action := dexterra.PairAction(res[dexterra.PairActionIdx].Value)
	switch action {
	case dexterra.SwapAction:
		return m.swapMatchedToParsedTx(res, pair)
	case dexterra.ProvideAction:
		return m.provideMatchedToParsedTx(res, pair)
	case dexterra.WithdrawAction:
		return m.withdrawMatchedToParsedTx(res, pair)
	}

	msg := fmt.Sprintf("action must be (%s, %s, %s)", dexterra.SwapAction, dexterra.ProvideAction, dexterra.WithdrawAction)
	return nil, errors.New(msg)
}

func (m *pairMapper) swapMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	matchMap, err := eventlog.ResultToItemMapForKeys(
		res,
		dexterra.PairAddrKey,
		pdex.PairSwapOfferAssetKey,
		pdex.PairSwapOfferAmountKey,
		pdex.PairSwapReturnAmountKey,
		pdex.PairSwapSenderKey,
		pdex.PairSwapCommissionAmountKey,
	)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	offerAsset := matchMap[pdex.PairSwapOfferAssetKey].Value
	offerIdx := 0
	if pair.Assets[1] == offerAsset {
		offerIdx = 1
	}
	returnIdx := (offerIdx + 1) % 2

	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}

	assets[offerIdx].Amount = matchMap[pdex.PairSwapOfferAmountKey].Value
	assets[returnIdx].Amount = fmt.Sprintf("-%s", matchMap[pdex.PairSwapReturnAmountKey].Value)

	return []*dex.ParsedTx{{
		Type:             dex.Swap,
		ContractAddr:     matchMap[dexterra.PairAddrKey].Value,
		Sender:           matchMap[pdex.PairSwapSenderKey].Value,
		Assets:           assets,
		CommissionAmount: matchMap[pdex.PairSwapCommissionAmountKey].Value,
		Meta:             nil,
	}}, nil
}

func (m *pairMapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	matchMap, err := eventlog.ResultToItemMapForKeys(
		res,
		dexterra.PairAddrKey,
		dexterra.PairProvideAssetsKey,
		dexterra.PairProvideSenderKey,
		dexterra.PairProvideShareKey,
		dexterra.PairProvideRefundAssetKey,
	)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	assets, err := dex.GetAssetsFromAssetsString(matchMap[dexterra.PairProvideAssetsKey].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}
	if assets[0].Addr != pair.Assets[0] {
		assets = []dex.Asset{assets[1], assets[0]}
	}

	meta := map[string]interface{}{}
	refundItem, ok := matchMap[dexterra.PairProvideRefundAssetKey]
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
		meta[dexterra.PairProvideRefundAssetKey] = refundAssets
	}

	return []*dex.ParsedTx{{
		Type:         dex.Provide,
		ContractAddr: matchMap[dexterra.PairAddrKey].Value,
		Sender:       matchMap[dexterra.PairProvideSenderKey].Value,
		Assets:       [2]dex.Asset(assets),
		LpAddr:       pair.LpAddr,
		LpAmount:     matchMap[dexterra.PairProvideShareKey].Value,
	}}, nil
}

func (m *pairMapper) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	matchMap, err := eventlog.ResultToItemMapForKeys(
		res,
		dexterra.PairAddrKey,
		dexterra.PairWithdrawRefundAssetsKey,
		dexterra.PairWithdrawSenderKey,
		dexterra.PairWithdrawWithdrawShareKey,
	)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := dex.GetAssetsFromAssetsString(matchMap[dexterra.PairWithdrawRefundAssetsKey].Value)
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
		ContractAddr: matchMap[dexterra.PairAddrKey].Value,
		Sender:       matchMap[dexterra.PairWithdrawSenderKey].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     matchMap[dexterra.PairWithdrawWithdrawShareKey].Value,
	}}, nil

}

// Apply refund asset to provided asset for cw20
// cw20 token is not refunded in provide event, it is transferred deducted amount to pair once.
// wasm message shows users requested amount rather than actual provided amount.
func (m *pairMapper) applyRefundAsset(provide []dex.Asset, refund []dex.Asset) (applied []dex.Asset, err error) {
	applied = make([]dex.Asset, len(provide))
	copy(applied, provide)

	for idx := range provide {
		if provide[idx].Addr != refund[idx].Addr {
			return nil, errors.New("provide and refund assets must be same order")
		}
		if !terra.IsCw20(provide[idx].Addr) {
			continue
		}

		amount, err := dex.ToBigInt(provide[idx].Amount)
		if err != nil {
			return nil, err
		}
		refundAmount, err := dex.ToBigInt(refund[idx].Amount)
		if err != nil {
			return nil, err
		}
		applied[idx].Amount = amount.Sub(amount, refundAmount).String()
	}

	return applied, nil
}
