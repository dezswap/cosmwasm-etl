package terraswap

import (
	"fmt"
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	t "github.com/dezswap/cosmwasm-etl/pkg/rules/terraswap"
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

type initialProvideMapper struct{ mapperMixin }

type transferMapper struct {
	mixin   mapperMixin
	pairSet map[string]parser.Pair
}
type wasmCommonTransferMapper struct {
	mixin   mapperMixin
	pairSet map[string]parser.Pair
}

// match implements mapper
func (m *createPairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	if err := m.mapperMixin.checkResult(res, t.CreatePairMatchedLen); err != nil {
		return nil, errors.Wrap(err, "createPairMapper.MatchedToParsedTx")
	}

	assets := strings.Split(res[t.FactoryPairIdx].Value, "-")
	if len(assets) != 2 {
		msg := fmt.Sprintf("expected assets length(%d)", 2)
		return nil, errors.New(msg)
	}

	return &parser.ParsedTx{
		Type:         parser.CreatePair,
		Sender:       "",
		ContractAddr: res[t.FactoryPairAddrIdx].Value,
		Assets: [2]parser.Asset{
			{Addr: assets[0]},
			{Addr: assets[1]},
		},
		LpAddr:   res[t.FactoryLpAddrIdx].Value,
		LpAmount: "",
	}, nil
}

// match implements mapper
func (m *pairMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	pair, ok := m.pairSet[res[t.PairAddrIdx].Value]
	if !ok {
		msg := fmt.Sprintf("pairMapper.MatchedToParsedTx no pair(%s)", res[t.PairAddrIdx].Value)
		return nil, errors.New(msg)
	}

	action := t.PairAction(res[t.PairActionIdx].Value)
	switch action {
	case t.SwapAction:
		return m.swapMatchedToParsedTx(res, pair)
	case t.ProvideAction:
		return m.provideMatchedToParsedTx(res, pair)
	case t.WithdrawAction:
		return m.withdrawMatchedToParsedTx(res, pair)
	}

	msg := fmt.Sprintf("action must be (%s, %s, %s)", t.SwapAction, t.ProvideAction, t.WithdrawAction)
	return nil, errors.New(msg)
}

func (m *pairMapper) swapMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.mixin.checkResult(res, t.PairSwapMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.swapMatchedToParsedTx")
	}

	offerAsset := res[t.PairSwapOfferAssetIdx].Value
	offerIdx := 0
	if pair.Assets[1] == offerAsset {
		offerIdx = 1
	}
	returnIdx := (offerIdx + 1) % 2

	assets := [2]parser.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}

	assets[offerIdx].Amount = res[t.PairSwapOfferAmountIdx].Value
	assets[returnIdx].Amount = fmt.Sprintf("-%s", res[t.PairSwapReturnAmountIdx].Value)

	return &parser.ParsedTx{
		Type:             parser.Swap,
		ContractAddr:     res[t.PairAddrIdx].Value,
		Assets:           assets,
		CommissionAmount: res[t.PairSwapCommissionAmountIdx].Value,
	}, nil
}

func (m *pairMapper) provideMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.mixin.checkResult(res, t.PairProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.PairProvideMatchedLen")
	}

	assets, err := parser.GetAssetsFromAssetsString(res[t.PairProvideAssetsIdx].Value)
	if err != nil {
		return nil, errors.Wrap(err, "pairMapper.provideMatchedToParsedTx")
	}

	if assets[0].Addr != pair.Assets[0] {
		assets = []parser.Asset{assets[1], assets[0]}
	}

	return &parser.ParsedTx{
		Type:         parser.Provide,
		ContractAddr: res[t.PairAddrIdx].Value,
		Assets:       [2]parser.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[t.PairProvideShareIdx].Value,
	}, nil
}

func (m *pairMapper) withdrawMatchedToParsedTx(res eventlog.MatchedResult, pair parser.Pair) (*parser.ParsedTx, error) {
	if err := m.mixin.checkResult(res, t.PairWithdrawMatchedLen); err != nil {
		return nil, errors.Wrap(err, "pairMapper.withdrawMatchedToParsedTx")
	}

	assets, err := parser.GetAssetsFromAssetsString(res[t.PairWithdrawRefundAssetsIdx].Value)
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
		ContractAddr: res[t.PairAddrIdx].Value,
		Assets:       [2]parser.Asset{assets[0], assets[1]},
		LpAddr:       pair.LpAddr,
		LpAmount:     res[t.PairWithdrawWithdrawShareIdx].Value,
	}, nil

}

func (m *initialProvideMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	if err := m.checkResult(res, t.PairInitialProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}
	return &parser.ParsedTx{
		Type:         parser.InitialProvide,
		Sender:       "",
		ContractAddr: res[t.PairInitialProvideToIdx].Value,
		Assets:       [2]parser.Asset{},
		LpAddr:       res[t.PairInitialProvideAddrIdx].Value,
		LpAmount:     res[t.PairInitialProvideAmountIdx].Value,
		Meta:         nil,
	}, nil
}

// match implements mapper
func (m *wasmCommonTransferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	if err := m.mixin.checkResult(res); err != nil {
		return nil, errors.Wrap(err, "wasmCommonTransferMapper.MatchedToParsedTx")
	}

	pair, ok := m.pairSet[res[t.WasmTransferToIdx].Value]
	if !ok {
		msg := fmt.Sprintf("wasmCommonTransferMapper.MatchedToParsedTx no pair(%s)", res[t.WasmTransferToIdx].Value)
		return nil, errors.New(msg)
	}
	assets := [2]parser.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	meta := make(map[string]interface{})

	target := res[t.WasmTransferCw20AddrIdx].Value
	idx := parser.IndexOf(pair.Assets, target)
	if idx == -1 {
		meta[target] = res[t.WasmTransferAmountIdx].Value
	} else {
		assets[idx].Amount = res[t.WasmTransferAmountIdx].Value
	}

	return &parser.ParsedTx{
		Type:         parser.Transfer,
		Sender:       res[t.WasmTransferFromIdx].Value,
		ContractAddr: res[t.WasmTransferToIdx].Value,
		Assets:       assets,
		Meta:         meta,
	}, nil

}

// match implements mapper
func (m *transferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) (*parser.ParsedTx, error) {
	if err := m.mixin.checkResult(res, t.TransferMatchedLen); err != nil {
		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}

	pair, ok := m.pairSet[res[t.TransferRecipientIdx].Value]
	if !ok {
		msg := fmt.Sprintf("transferMapper.MatchedToParsedTx no pair(%s)", res[t.TransferRecipientIdx].Value)
		return nil, errors.New(msg)
	}
	assets := [2]parser.Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	meta := make(map[string]interface{})

	amountStrs := strings.Split(res[t.TransferAmountIdx].Value, ",")
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
	}

	return &parser.ParsedTx{
		Type:         parser.Transfer,
		Sender:       res[t.TransferSenderIdx].Value,
		ContractAddr: res[t.TransferRecipientIdx].Value,
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
