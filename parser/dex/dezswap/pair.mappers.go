package dezswap

import (
	"fmt"
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	pdex "github.com/dezswap/cosmwasm-etl/pkg/dex"
	ds "github.com/dezswap/cosmwasm-etl/pkg/dex/dezswap"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/xpla"
	"github.com/pkg/errors"
)

var _ parser.Mapper[dex.ParsedTx] = &pairMapperImpl{}
var _ pairMapper = &pairMapperImpl{}

type pairMapper interface {
	getPair(addr string) (dex.Pair, error)
	CheckResult(res eventlog.MatchedResult, expectedLen int) error
	SortResult(res eventlog.MatchedResult)
	swapMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error)
	provideMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error)
	withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error)
}

type pairMapperImpl struct {
	pairMapper
}

type pairMapperMixin struct {
	pdex.MapperMixin
	pairSet map[string]dex.Pair
}

type pairV2Mapper struct {
	*pairMapperMixin
}

func pairMapperBy(chainId string, height uint64, pairSet map[string]dex.Pair) (parser.Mapper[dex.ParsedTx], error) {
	base := &pairMapperMixin{pdex.MapperMixin{}, pairSet}
	if strings.HasPrefix(chainId, ds.TestnetPrefix) {
		if height < ds.TestnetV2Height {
			return &pairMapperImpl{base}, nil
		} else {
			return &pairMapperImpl{&pairV2Mapper{base}}, nil
		}
	} else if strings.HasPrefix(chainId, ds.MainnetPrefix) {
		if height < ds.MainnetV2Height {
			return &pairMapperImpl{base}, nil
		} else {
			return &pairMapperImpl{&pairV2Mapper{base}}, nil
		}
	}

	return nil, errors.New("chainId is not supported")
}

func (m *pairMapperImpl) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	if len(res) < ds.PairCommonMatchedLen {
		msg := fmt.Sprintf("results length must bigger than %d", ds.PairCommonMatchedLen)
		return nil, errors.New(msg)
	}
	m.SortResult(res)
	pair, err := m.getPair(res[ds.PairAddrIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapperImpl.MatchedToParsedTx")
	}

	action := ds.PairAction(res[ds.PairActionIdx].Value)
	switch action {
	case ds.SwapAction:
		return m.swapMatchedToParsedTx(res, pair)
	case ds.ProvideAction:
		return m.provideMatchedToParsedTx(res, pair)
	case ds.WithdrawAction:
		return m.withdrawMatchedToParsedTx(res, pair)
	}

	msg := fmt.Sprintf("action must be (%s, %s, %s)", ds.SwapAction, ds.ProvideAction, ds.WithdrawAction)
	return nil, errors.New(msg)
}

func (m *pairMapperMixin) getPair(addr string) (dex.Pair, error) {
	pair, ok := m.pairSet[addr]
	if !ok {
		msg := fmt.Sprintf("pairMapper.MatchedToParsedTx no pair(%s)", addr)
		return dex.Pair{}, errors.New(msg)
	}
	return pair, nil
}

func (m *pairMapperMixin) swapMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.CheckResult(res, ds.PairSwapMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	offerAsset := res[ds.PairSwapOfferAssetIdx].Value
	offerIdx := 0
	if pair.Assets[1] == offerAsset {
		offerIdx = 1
	}
	returnIdx := (offerIdx + 1) % 2

	assets := [2]dex.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}

	assets[offerIdx].Amount = res[ds.PairSwapOfferAmountIdx].Value
	assets[returnIdx].Amount = fmt.Sprintf("-%s", res[ds.PairSwapReturnAmountIdx].Value)

	return []*dex.ParsedTx{{
		Type:             dex.Swap,
		Sender:           res[ds.PairSwapSenderIdx].Value,
		ContractAddr:     res[ds.PairAddrIdx].Value,
		Assets:           assets,
		CommissionAmount: res[ds.PairSwapCommissionAmountIdx].Value,
	}}, nil
}

func (m *pairMapperMixin) provideMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.CheckResult(res, ds.PairProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.PairProvideMatchedLen")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[ds.PairProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []dex.Asset{assets[1], assets[0]}
	}

	return []*dex.ParsedTx{{
		Type:         dex.Provide,
		Sender:       res[ds.PairProvideSenderIdx].Value,
		ContractAddr: res[ds.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[ds.PairProvideShareIdx].Value,
	}}, nil
}

func (m *pairMapperMixin) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.CheckResult(res, ds.PairWithdrawMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[ds.PairWithdrawRefundAssetsIdx].Value)
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
		Sender:       res[ds.PairWithdrawSenderIdx].Value,
		ContractAddr: res[ds.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[ds.PairWithdrawWithdrawShareIdx].Value,
	}}, nil

}

func (m *pairV2Mapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.CheckResult(res, ds.PairV2ProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.PairProvideMatchedLen")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[ds.PairV2ProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.provideMatchedToParsedTx")
	}
	if assets[0].Addr != pair.Assets[0] {
		assets = []dex.Asset{assets[1], assets[0]}
	}

	refundAssets, err := dex.GetAssetsFromAssetsString(res[ds.PairV2RefundAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.provideMatchedToParsedTx")
	}
	if refundAssets[0].Addr != pair.Assets[0] {
		refundAssets = []dex.Asset{refundAssets[1], refundAssets[0]}
	}

	meta := map[string]interface{}{
		res[ds.PairV2RefundAssetsIdx].Key: refundAssets,
	}

	assets, err = m.applyRefundAsset(assets, refundAssets)
	if err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.provideMatchedToParsedTx")
	}

	return []*dex.ParsedTx{{
		Type:         dex.Provide,
		Sender:       res[ds.PairV2ProvideSenderIdx].Value,
		ContractAddr: res[ds.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[ds.PairV2ProvideShareIdx].Value,
		Meta:         meta,
	}}, nil
}

// Apply refund asset to provided asset for cw20
// cw20 token is not refunded in provide event, it is transferred deducted amount to pair once.
// wasm message shows users requested amount rather than actual provided amount.
func (m *pairV2Mapper) applyRefundAsset(provide []dex.Asset, refund []dex.Asset) (applied []dex.Asset, err error) {
	applied = make([]dex.Asset, len(provide))
	copy(applied, provide)

	for idx := range provide {
		if provide[idx].Addr != refund[idx].Addr {
			return nil, errors.New("provide and refund assets must be same order")
		}
		if !xpla.IsCw20(provide[idx].Addr) {
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
