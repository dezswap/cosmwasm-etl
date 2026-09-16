package asi

import (
	"fmt"
	"strings"

	pdex "github.com/dezswap/cosmwasm-etl/pkg/dex"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/asi"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
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
	if strings.HasPrefix(chainId, asi.TestnetPrefix) {
		if height < asi.TestnetV2Height {
			return &pairMapperImpl{base}, nil
		} else {
			return &pairMapperImpl{&pairV2Mapper{base}}, nil
		}
	} else if strings.HasPrefix(chainId, asi.MainnetPrefix) {
		if height < asi.MainnetV2Height {
			return &pairMapperImpl{base}, nil
		} else {
			return &pairMapperImpl{&pairV2Mapper{base}}, nil
		}
	}

	return nil, errors.New("chainId is not supported")
}

func (m *pairMapperImpl) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*dex.ParsedTx, error) {
	if len(res) < asi.PairCommonMatchedLen {
		msg := fmt.Sprintf("results length must bigger than %d", asi.PairCommonMatchedLen)
		return nil, errors.New(msg)
	}
	m.SortResult(res)
	pair, err := m.getPair(res[asi.PairAddrIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapperImpl.MatchedToParsedTx")
	}

	action := asi.PairAction(res[asi.PairActionIdx].Value)
	switch action {
	case asi.SwapAction:
		return m.swapMatchedToParsedTx(res, pair)
	case asi.ProvideAction:
		return m.provideMatchedToParsedTx(res, pair)
	case asi.WithdrawAction:
		return m.withdrawMatchedToParsedTx(res, pair)
	}

	msg := fmt.Sprintf("action must be (%s, %s, %s)", asi.SwapAction, asi.ProvideAction, asi.WithdrawAction)
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
	if err := m.CheckResult(res, asi.PairSwapMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	matchMap, err := eventlog.ResultToItemMapForKeys(
		res,
		pdex.PairSwapOfferAssetKey,
		pdex.PairSwapOfferAmountKey,
		pdex.PairSwapReturnAmountKey,
		pdex.PairSwapSenderKey,
		pdex.PairSwapCommissionAmountKey,
	)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapperMixin.swapMatchedToParsedTx")
	}

	offerAsset := matchMap[pdex.PairSwapOfferAssetKey]
	offerIdx := 0
	if pair.Assets[1] == offerAsset.Value {
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
		Sender:           matchMap[pdex.PairSwapSenderKey].Value,
		ContractAddr:     res[asi.PairAddrIdx].Value,
		Assets:           assets,
		CommissionAmount: matchMap[pdex.PairSwapCommissionAmountKey].Value,
	}}, nil
}

func (m *pairMapperMixin) provideMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.CheckResult(res, asi.PairProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.PairProvideMatchedLen")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[asi.PairProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []dex.Asset{assets[1], assets[0]}
	}

	return []*dex.ParsedTx{{
		Type:         dex.Provide,
		Sender:       res[asi.PairProvideSenderIdx].Value,
		ContractAddr: res[asi.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[asi.PairProvideShareIdx].Value,
	}}, nil
}

func (m *pairMapperMixin) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.CheckResult(res, asi.PairWithdrawMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[asi.PairWithdrawRefundAssetsIdx].Value)
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
		Sender:       res[asi.PairWithdrawSenderIdx].Value,
		ContractAddr: res[asi.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[asi.PairWithdrawWithdrawShareIdx].Value,
	}}, nil

}

func (m *pairV2Mapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair dex.Pair) ([]*dex.ParsedTx, error) {
	if err := m.CheckResult(res, asi.PairV2ProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.PairProvideMatchedLen")
	}

	assets, err := dex.GetAssetsFromAssetsString(res[asi.PairV2ProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.provideMatchedToParsedTx")
	}

	refundAssets, err := dex.GetAssetsFromAssetsString(res[asi.PairV2RefundAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.provideMatchedToParsedTx")
	}
	meta := map[string]interface{}{
		res[asi.PairV2RefundAssetsIdx].Key: refundAssets,
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []dex.Asset{assets[1], assets[0]}
	}

	return []*dex.ParsedTx{{
		Type:         dex.Provide,
		Sender:       res[asi.PairV2ProvideSenderIdx].Value,
		ContractAddr: res[asi.PairAddrIdx].Value,
		Assets:       [2]dex.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[asi.PairV2ProvideShareIdx].Value,
		Meta:         meta,
	}}, nil
}
