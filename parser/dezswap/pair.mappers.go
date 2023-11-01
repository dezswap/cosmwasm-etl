package dezswap

import (
	"fmt"
	"strings"

	"math/big"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/rules/dezswap"
	ds "github.com/dezswap/cosmwasm-etl/pkg/rules/dezswap"
	"github.com/dezswap/cosmwasm-etl/pkg/xpla"
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
	if strings.HasPrefix(chainId, dezswap.TestnetPrefix) {
		if height < dezswap.TestnetV2Height {
			return &pairMapperImpl{base}, nil
		} else {
			return &pairMapperImpl{&pairV2Mapper{base}}, nil
		}
	} else if strings.HasPrefix(chainId, dezswap.MainnetPrefix) {
		if height < dezswap.MainnetV2Height {
			return &pairMapperImpl{base}, nil
		} else {
			return &pairMapperImpl{&pairV2Mapper{base}}, nil
		}
	}

	return nil, errors.New("chainId is not supported")
}

func (m *pairMapperImpl) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	if len(res) < ds.PairCommonMatchedLen {
		msg := fmt.Sprintf("results length must bigger than %d", ds.PairCommonMatchedLen)
		return nil, errors.New(msg)
	}
	sortResult(res)
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

func (m *pairMapperMixin) getPair(addr string) (parser.Pair, error) {
	pair, ok := m.pairSet[addr]
	if !ok {
		msg := fmt.Sprintf("pairMapper.MatchedToParsedTx no pair(%s)", addr)
		return parser.Pair{}, errors.New(msg)
	}
	return pair, nil
}

func (m *pairMapperMixin) swapMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.checkResult(res, ds.PairSwapMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	offerAsset := res[ds.PairSwapOfferAssetIdx].Value
	offerIdx := 0
	if pair.Assets[1] == offerAsset {
		offerIdx = 1
	}
	returnIdx := (offerIdx + 1) % 2

	assets := [2]parser.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}

	assets[offerIdx].Amount = res[ds.PairSwapOfferAmountIdx].Value
	assets[returnIdx].Amount = fmt.Sprintf("-%s", res[ds.PairSwapReturnAmountIdx].Value)

	return &parser.ParsedTx{
		Type:             parser.Swap,
		Sender:           res[ds.PairSwapSenderIdx].Value,
		ContractAddr:     res[ds.PairAddrIdx].Value,
		Assets:           assets,
		CommissionAmount: res[ds.PairSwapCommissionAmountIdx].Value,
	}, nil
}

func (m *pairMapperMixin) provideMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.checkResult(res, ds.PairProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.PairProvideMatchedLen")
	}

	assets, err := parser.GetAssetsFromAssetsString(res[ds.PairProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []parser.Asset{assets[1], assets[0]}
	}

	return &parser.ParsedTx{
		Type:         parser.Provide,
		Sender:       res[ds.PairProvideSenderIdx].Value,
		ContractAddr: res[ds.PairAddrIdx].Value,
		Assets:       [2]parser.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[ds.PairProvideShareIdx].Value,
	}, nil
}

func (m *pairMapperMixin) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.checkResult(res, ds.PairWithdrawMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := parser.GetAssetsFromAssetsString(res[ds.PairWithdrawRefundAssetsIdx].Value)
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
		Sender:       res[ds.PairWithdrawSenderIdx].Value,
		ContractAddr: res[ds.PairAddrIdx].Value,
		Assets:       [2]parser.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[ds.PairWithdrawWithdrawShareIdx].Value,
	}, nil

}

func (m *pairV2Mapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.checkResult(res, ds.PairV2ProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.PairProvideMatchedLen")
	}

	assets, err := parser.GetAssetsFromAssetsString(res[ds.PairV2ProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.provideMatchedToParsedTx")
	}
	if assets[0].Addr != pair.Assets[0] {
		assets = []parser.Asset{assets[1], assets[0]}
	}

	refundAssets, err := parser.GetAssetsFromAssetsString(res[ds.PairV2RefundAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.provideMatchedToParsedTx")
	}
	if refundAssets[0].Addr != pair.Assets[0] {
		refundAssets = []parser.Asset{refundAssets[1], refundAssets[0]}
	}

	meta := map[string]interface{}{
		res[ds.PairV2RefundAssetsIdx].Key: refundAssets,
	}

	assets, err = m.applyRefundAsset(assets, refundAssets)
	if err != nil {
		return nil, errors.Wrap(err, "v2PairMapper.provideMatchedToParsedTx")
	}

	return &parser.ParsedTx{
		Type:         parser.Provide,
		Sender:       res[ds.PairV2ProvideSenderIdx].Value,
		ContractAddr: res[ds.PairAddrIdx].Value,
		Assets:       [2]parser.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[ds.PairV2ProvideShareIdx].Value,
		Meta:         meta,
	}, nil
}

// Apply refund asset to provided asset for cw20
// cw20 token is not refunded in provide event, it is transferred deducted amount to pair once.
// wasm message shows users requested amount rather than actual provided amount.
func (m *pairV2Mapper) applyRefundAsset(provide []parser.Asset, refund []parser.Asset) (applied []parser.Asset, err error) {
	toBigInt := func(amount string) (*big.Int, error) {
		amountBigInt, ok := big.NewInt(0).SetString(amount, 10)
		if !ok {
			return nil, errors.New("invalid amount")
		}
		return amountBigInt, nil
	}
	applied = make([]parser.Asset, len(provide))
	copy(applied, provide)

	for idx := range provide {
		if provide[idx].Addr != refund[idx].Addr {
			return nil, errors.New("provide and refund assets must be same order")
		}
		if !xpla.IsCw20(provide[idx].Addr) {
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
