package dex

import (
	"errors"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
)

var ParsableRules = map[string]bool{
	string(eventlog.TransferType): true,
	string(eventlog.FromContract): true,
	string(eventlog.WasmType):     true,
}

var (
	PAIR_QUERY_POOL_STRING, _        = QueryToJsonStr[PoolInfoReq](PoolInfoReq{})
	PAIR_QUERY_POOL_BASE64_STRING, _ = QueryToBase64Str[PoolInfoReq](PoolInfoReq{})
)

var (
	ErrQueryDifferentHeight = errors.New("query different height")
	ErrEmptyEventValue      = errors.New("empty event value")
)

const (
	CreatePairMatchedLen = FactoryLpAddrIdx + 1
)

const (
	FactoryAddrIdx = iota
	FactoryActionIdx
	FactoryPairIdx
	FactoryPairAddrIdx
	FactoryLpAddrIdx
)

const (
	TransferAmountKey    = "amount"
	TransferRecipientKey = "recipient"
	TransferSenderKey    = "sender"
)

const (
	WasmTransferAction     = "transfer"
	WasmTransferFromAction = "transfer_from"
)

const (
	WasmTransferLegacyCw20AddrKey = "contract_address"
	WasmTransferCw20AddrKey       = "_contract_address"
	WasmTransferActionKey         = "action"
	WasmTransferAmountKey         = "amount"
	WasmTransferFromKey           = "from"
	WasmTransferToKey             = "to"

	WasmTransferTaxFlagPatternKey = "tax"
)

const (
	PairInitialProvideAddrIdx = iota
	PairInitialProvideActionIdx
	PairInitialProvideAmountIdx
	PairInitialProvideToIdx
	PairInitialProvideMatchedLen
)

const (
	PairInitialProvideAddrKey   = "_contract_address"
	PairInitialProvideActionKey = "action"
	PairInitialProvideAmountKey = "amount"
	PairInitialProvideToKey     = "to"
)
