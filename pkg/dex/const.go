package dex

import (
	"errors"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
)

var ParsableRules = map[string]bool{
	string(eventlog.TransferType):   true,
	string(eventlog.FromContract):   true,
	string(eventlog.WasmType):       true,
	string(eventlog.TaxPaymentType): true,
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

const (
	PairTaxPaymentReverseChargeIdx = iota
	PairTaxPaymentTaxAmountIdx
	PairTaxPaymentTaxMatchedLen
)

const (
	PairTaxPaymentReverseChargeKey = "reverse_charge"
	PairTaxPaymentTaxAmountKey     = "tax_amount"
)

const (
	BurnAddrIdx = iota
	BurnActionIdx
	BurnAmountIdx
	BurnFromIdx
	BurnMatchedLen
)

const (
	BurnAddrKey   = "_contract_address"
	BurnActionKey = "action"
	BurnAmountKey = "amount"
	BurnFromKey   = "from"
)

const (
	PairSwapAskAssetKey         = "ask_asset"
	PairSwapCommissionAmountKey = "commission_amount"
	PairSwapOfferAmountKey      = "offer_amount"
	PairSwapOfferAssetKey       = "offer_asset"
	PairSwapReceiverKey         = "receiver"
	PairSwapReturnAmountKey     = "return_amount"
	PairSwapTaxAmountKey        = "tax_amount" // columbus-5
	PairSwapSenderKey           = "sender"
	PairSwapSpreadAmountKey     = "spread_amount"
)
