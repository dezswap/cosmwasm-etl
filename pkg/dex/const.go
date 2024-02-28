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
	QUERY_DIFFERENT_HEIGHT_ERROR = errors.New("query different height")
)
