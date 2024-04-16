package dex

import (
	"fmt"
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	dex_common "github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"

	"github.com/pkg/errors"
)

type mapperMixin struct{}

type factoryMapper struct{ mapperMixin }
type transferMapper struct {
	pairSet map[string]Pair
}
type wasmCommonTransferMapper struct {
	cw20AddrKey string
	pairSet     map[string]Pair
}

type initialProvideMapper struct{ mapperMixin }

func NewFactoryMapper() parser.Mapper[ParsedTx] {
	return &factoryMapper{mapperMixin{}}
}

// MatchedToParsedTx implements parser.Mapper.
func (m *factoryMapper) MatchedToParsedTx(res eventlog.MatchedResult, optional ...interface{}) ([]*ParsedTx, error) {
	if err := m.mapperMixin.checkResult(res, dex_common.CreatePairMatchedLen); err != nil {
		return nil, errors.Wrap(err, "factoryMapper.MatchedToParsedTx")
	}

	assets := strings.Split(res[dex_common.FactoryPairIdx].Value, "-")
	if len(assets) != 2 {
		msg := fmt.Sprintf("expected assets length(%d)", 2)
		return nil, errors.New(msg)
	}

	return []*ParsedTx{{
		Type:         CreatePair,
		Sender:       "",
		ContractAddr: res[dex_common.FactoryPairAddrIdx].Value,
		Assets: [2]Asset{
			{Addr: assets[0]},
			{Addr: assets[1]},
		},
		LpAddr:   res[dex_common.FactoryLpAddrIdx].Value,
		LpAmount: "",
	}}, nil
}

func NewWasmTransferMapper(cw20AddrKey string, pairSet map[string]Pair) parser.Mapper[ParsedTx] {
	return &wasmCommonTransferMapper{cw20AddrKey, pairSet}
}

// match implements mapper
func (m *wasmCommonTransferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*ParsedTx, error) {
	matchMap, err := eventlog.ResultToItemMap(res)
	if err != nil {
		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}

	sender, receiver := matchMap[dex_common.WasmTransferFromKey].Value, matchMap[dex_common.WasmTransferToKey].Value
	fp, fromPair := m.pairSet[sender]
	tp, toPair := m.pairSet[receiver]
	if !fromPair && !toPair {
		return nil, nil
	}

	var txs []*ParsedTx

	amount := matchMap[dex_common.WasmTransferAmountKey].Value
	cw20Addr := matchMap[m.cw20AddrKey].Value
	if fromPair {
		txs = append(txs, m.wasmTransferToParsedTx(fp, cw20Addr, sender, amount, true))
	}
	if toPair {
		txs = append(txs, m.wasmTransferToParsedTx(tp, cw20Addr, sender, amount, false))
	}

	return txs, nil
}

func (m *wasmCommonTransferMapper) wasmTransferToParsedTx(pair Pair, cw20Addr, from, amount string, fromPair bool) *ParsedTx {
	assets := [2]Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	meta := make(map[string]interface{})

	idx := IndexOf(pair.Assets, cw20Addr)
	if idx == -1 {
		meta[cw20Addr] = amount
	} else {
		assets[idx].Amount = amount
	}

	// outflow
	if fromPair {
		assets[idx].Amount = fmt.Sprintf("-%s", assets[idx].Amount)
	}

	return &ParsedTx{
		Type:         Transfer,
		Sender:       from,
		ContractAddr: pair.ContractAddr,
		Assets:       assets,
		Meta:         meta,
	}
}

func NewTransferMapper(pairSet map[string]Pair) parser.Mapper[ParsedTx] {
	return &transferMapper{pairSet}
}

// match implements mapper
func (m *transferMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*ParsedTx, error) {
	matchMap, err := eventlog.ResultToItemMap(res)
	if err != nil {
		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}

	sender, receiver := matchMap[dex_common.TransferSenderKey].Value, matchMap[dex_common.TransferRecipientKey].Value
	fp, fromPair := m.pairSet[sender]
	tp, toPair := m.pairSet[receiver]

	if !fromPair && !toPair {
		return nil, nil
	}

	assetsStr := matchMap[dex_common.TransferAmountKey].Value
	txs := []*ParsedTx{}
	if fromPair {
		tx, err := m.transferToParsedTx(fp, sender, assetsStr, true)
		if err != nil {
			return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
		}
		txs = append(txs, tx)
	}
	if toPair {
		tx, err := m.transferToParsedTx(tp, sender, assetsStr, false)
		if err != nil {
			return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
		}
		txs = append(txs, tx)
	}
	if len(txs) == 0 {
		txs = nil
	}

	return txs, nil
}

func (m transferMapper) transferToParsedTx(pair Pair, from, assetsStr string, fromPair bool) (*ParsedTx, error) {
	assets := [2]Asset{
		{Addr: pair.Assets[0]},
		{Addr: pair.Assets[1]},
	}
	meta := make(map[string]interface{})

	amountStrs := strings.Split(assetsStr, ",")
	for _, amountStr := range amountStrs {
		asset, err := GetAssetFromAmountAssetString(amountStr)
		if err != nil {
			return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
		}
		idx := IndexOf(pair.Assets, asset.Addr)
		if idx == -1 {
			meta[asset.Addr] = asset.Amount
		} else {
			assets[idx] = asset
		}
		if fromPair {
			assets[idx].Amount = fmt.Sprintf("-%s", assets[idx].Amount)
		}
	}

	return &ParsedTx{
		Type:         Transfer,
		Sender:       from,
		ContractAddr: pair.ContractAddr,
		Assets:       assets,
		LpAddr:       "",
		LpAmount:     "",
		Meta:         meta,
	}, nil
}

func NewInitialProvideMapper() parser.Mapper[ParsedTx] {
	return &initialProvideMapper{mapperMixin{}}
}

func (m *initialProvideMapper) MatchedToParsedTx(res eventlog.MatchedResult, optionals ...interface{}) ([]*ParsedTx, error) {
	if err := m.checkResult(res, dex_common.PairInitialProvideMatchedLen); err != nil {
		return nil, errors.Wrap(err, "initialProvideMapper.MatchedToParsedTx")
	}
	matchMap, err := eventlog.ResultToItemMap(res)
	if err != nil {
		return nil, errors.Wrap(err, "transferMapper.MatchedToParsedTx")
	}

	return []*ParsedTx{{
		Type:         InitialProvide,
		Sender:       "",
		ContractAddr: matchMap[dex_common.PairInitialProvideToKey].Value,
		Assets:       [2]Asset{{}, {}},
		LpAddr:       matchMap[dex_common.PairInitialProvideAddrKey].Value,
		LpAmount:     matchMap[dex_common.PairInitialProvideAmountKey].Value,
		Meta:         nil,
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
