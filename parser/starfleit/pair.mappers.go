package starfleit

import (
	"fmt"
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	sf "github.com/dezswap/cosmwasm-etl/pkg/rules/starfleit"
	"github.com/pkg/errors"
)

var _ parser.Mapper = &pairMapperImpl{}
var _ pairMapper = &pairMapperImpl{}

type pairMapper interface {
	getPair(addr string) (parser.Pair, error)
	checkResult(res eventlog.MatchedResult, expectedLen int) error
	swapMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error)
	provideMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error)
	withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error)
}

type pairMapperImpl struct {
	pairMapper
}

type pairMapperMixin struct {
	mapperMixin
	pairSet map[string]parser.Pair
}

type pairV2Mapper struct {
	*pairMapperMixin
}

func pairMapperBy(chainId string, height uint64, pairSet map[string]parser.Pair) (parser.Mapper, error) {
	base := &pairMapperMixin{mapperMixin{}, pairSet}
	if strings.HasPrefix(chainId, sf.TestnetPrefix) {
		if height < sf.TestnetV2Height {
			return &pairMapperImpl{base}, nil
		} else {
			return &pairMapperImpl{&pairV2Mapper{base}}, nil
		}
	} else if strings.HasPrefix(chainId, sf.MainnetPrefix) {
		if height < sf.MainnetV2Height {
			return &pairMapperImpl{base}, nil
		} else {
			return &pairMapperImpl{&pairV2Mapper{base}}, nil
		}
	}

	return nil, errors.New("chainId is not supported")
}

func (m *pairMapperImpl) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	if len(res) < sf.PairCommonMatchedLen {
		msg := fmt.Sprintf("results length must bigger than %d", sf.PairCommonMatchedLen)
		return nil, errors.New(msg)
	}
	sortResult(res)
	pair, err := m.getPair(res[sf.PairAddrIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapperImpl.MatchedToParsedTx")
	}

	action := sf.PairAction(res[sf.PairActionIdx].Value)
	switch action {
	case sf.SwapAction:
		return m.swapMatchedToParsedTx(res, pair)
	case sf.ProvideAction:
		return m.provideMatchedToParsedTx(res, pair)
	case sf.WithdrawAction:
		return m.withdrawMatchedToParsedTx(res, pair)
	}

	msg := fmt.Sprintf("action must be (%s, %s, %s)", sf.SwapAction, sf.ProvideAction, sf.WithdrawAction)
	return nil, errors.New(msg)
}

func (m *pairMapperMixin) getPair(addr string) (parser.Pair, error) {
	pair, ok := m.pairSet[addr]
	if !ok {
		msg := fmt.Sprintf("pairMapper.MatchedToParsedTx no pair(%s)", addr)
		return parser.Pair{}, errors.New(msg)
	}
	return pair, nil
}

func (m *pairMapperMixin) swapMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.checkResult(res, sf.PairSwapMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	offerAsset := res[sf.PairSwapOfferAssetIdx].Value
	offerIdx := 0
	if pair.Assets[1] == offerAsset {
		offerIdx = 1
	}
	returnIdx := (offerIdx + 1) % 2

	assets := [2]parser.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}

	assets[offerIdx].Amount = res[sf.PairSwapOfferAmountIdx].Value
	assets[returnIdx].Amount = fmt.Sprintf("-%s", res[sf.PairSwapReturnAmountIdx].Value)

	return &parser.ParsedTx{
		Type:             parser.Swap,
		Sender:           res[sf.PairSwapSenderIdx].Value,
		ContractAddr:     res[sf.PairAddrIdx].Value,
		Assets:           assets,
		CommissionAmount: res[sf.PairSwapCommissionAmountIdx].Value,
	}, nil
}

func (m *pairMapperMixin) provideMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.checkResult(res, sf.PairProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.PairProvideMatchedLen")
	}

	assets, err := parser.GetAssetsFromAssetsString(res[sf.PairProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []parser.Asset{assets[1], assets[0]}
	}

	return &parser.ParsedTx{
		Type:         parser.Provide,
		Sender:       res[sf.PairProvideSenderIdx].Value,
		ContractAddr: res[sf.PairAddrIdx].Value,
		Assets:       [2]parser.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[sf.PairProvideShareIdx].Value,
	}, nil
}

func (m *pairMapperMixin) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.checkResult(res, sf.PairWithdrawMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := parser.GetAssetsFromAssetsString(res[sf.PairWithdrawRefundAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}
	for idx := range assets {
		assets[idx].Amount = fmt.Sprintf("-%s", assets[idx].Amount)
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []parser.Asset{assets[1], assets[0]}
	}

	return &parser.ParsedTx{
		Type:         parser.Withdraw,
		Sender:       res[sf.PairWithdrawSenderIdx].Value,
		ContractAddr: res[sf.PairAddrIdx].Value,
		Assets:       [2]parser.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[sf.PairWithdrawWithdrawShareIdx].Value,
	}, nil

}

func (m *pairV2Mapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.checkResult(res, sf.PairV2ProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.PairProvideMatchedLen")
	}

	assets, err := parser.GetAssetsFromAssetsString(res[sf.PairV2ProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.provideMatchedToParsedTx")
	}

	refundAssets, err := parser.GetAssetsFromAssetsString(res[sf.PairV2RefundAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.provideMatchedToParsedTx")
	}
	meta := map[string]interface{}{
		res[sf.PairV2RefundAssetsIdx].Key: refundAssets,
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []parser.Asset{assets[1], assets[0]}
	}

	return &parser.ParsedTx{
		Type:         parser.Provide,
		Sender:       res[sf.PairV2ProvideSenderIdx].Value,
		ContractAddr: res[sf.PairAddrIdx].Value,
		Assets:       [2]parser.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[sf.PairV2ProvideShareIdx].Value,
		Meta:         meta,
	}, nil
}
