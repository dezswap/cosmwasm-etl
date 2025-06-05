package datastore

import (
	"cosmossdk.io/math"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"sigs.k8s.io/yaml"
)

const (
	TOKEN        = "token"
	NATIVE_TOKEN = "native_token"
)

type BlockTxsRaw struct {
	BlockId int64   `json:"block_id"`
	Txs     []TxRaw `json:"txs"`
}

type TxRaw struct {
	TxHash    string
	TxContent *txtypes.GetTxResponse
}

type BlockTxsDTO struct {
	BlockId int64   `json:"block_id"`
	Txs     []TxDTO `json:"txs"`
}

// Copied from the type definition of Cosmos SDK and revised the stringed integer
type TxDTO struct {
	Height    string                    `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	TxHash    string                    `protobuf:"bytes,2,opt,name=txhash,proto3" json:"txhash,omitempty"`
	Codespace string                    `protobuf:"bytes,3,opt,name=codespace,proto3" json:"codespace,omitempty"`
	Code      uint32                    `protobuf:"varint,4,opt,name=code,proto3" json:"code,omitempty"`
	Data      string                    `protobuf:"bytes,5,opt,name=data,proto3" json:"data,omitempty"`
	RawLog    string                    `protobuf:"bytes,6,opt,name=raw_log,json=rawLog,proto3" json:"raw_log,omitempty"`
	Logs      cosmossdk.ABCIMessageLogs `protobuf:"bytes,7,rep,name=logs,proto3,castrepeated=ABCIMessageLogs" json:"logs"`
	Info      string                    `protobuf:"bytes,8,opt,name=info,proto3" json:"info,omitempty"`
	GasWanted string                    `protobuf:"varint,9,opt,name=gas_wanted,json=gasWanted,proto3" json:"gas_wanted,omitempty"`
	GasUsed   string                    `protobuf:"varint,10,opt,name=gas_used,json=gasUsed,proto3" json:"gas_used,omitempty"`

	// the type of Tx will be replace when we can find the appropriate type
	Tx        *codectypes.Any   `protobuf:"bytes,11,opt,name=tx,proto3" json:"tx,omitempty"`
	Timestamp string            `protobuf:"bytes,12,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Events    []abcitypes.Event `protobuf:"bytes,13,rep,name=events,proto3" json:"events"`
}

func (tx *TxRaw) ToString() (string, error) {
	jsonByte, err := tx.MarshalJSON()
	if err != nil {
		return "", err
	}

	return string(jsonByte), nil
}

func (tx *TxRaw) MarshalJSON() ([]byte, error) {
	jsonByte, err := yaml.YAMLToJSON([]byte(tx.TxContent.GetTxResponse().String()))
	if err != nil {
		// check if the response is JSON
		_, err = yaml.JSONToYAML([]byte(tx.TxContent.GetTxResponse().String()))
		if err != nil {
			return nil, err
		} else {
			return []byte(tx.TxContent.GetTxResponse().String()), nil
		}
	} else {
		return jsonByte, nil
	}
}

type PairListResponse struct {
	Pairs []UnitPair `json:"pairs"`
}

type PairListDTO struct {
	Pairs map[string]UnitPairDTO `json:"pairs"`
}

type UnitPair struct {
	AssetInfo      [2]AssetInfo `json:"asset_infos"`
	ContractAddr   string       `json:"contract_addr"`
	LiquidityToken string       `json:"liquidity_token"`
	AssetDecimals  [2]uint      `json:"asset_decimals"`
}

type UnitPairDTO struct {
	AssetInfo      [2]AssetInfoWithType `json:"asset_infos"`
	ContractAddr   string               `json:"contract_addr"`
	LiquidityToken string               `json:"liquidity_token"`
	AssetDecimals  [2]uint              `json:"asset_decimals"`
}

func (pair *UnitPair) Convert() *UnitPairDTO {
	ret := &UnitPairDTO{}

	ret.AssetInfo = [2]AssetInfoWithType{
		*pair.AssetInfo[0].GetInfo(),
		*pair.AssetInfo[1].GetInfo(),
	}

	ret.ContractAddr = pair.ContractAddr
	ret.LiquidityToken = pair.LiquidityToken
	ret.AssetDecimals = pair.AssetDecimals

	return ret
}

type AssetInfo struct {
	Token *struct {
		ContractAddr string `json:"contract_addr"`
	} `json:"token,omitempty"`

	NativeToken *struct {
		Denom string `json:"denom"`
	} `json:"native_token,omitempty"`
}

type AssetInfoWithType struct {
	Type           string `json:"type"`
	DenomOrAddress string `json:"denom_or_address"`
}

func (assets *AssetInfo) GetInfo() *AssetInfoWithType {
	ret := &AssetInfoWithType{}

	if assets.Token != nil {
		ret.Type = TOKEN
		ret.DenomOrAddress = assets.Token.ContractAddr
		return ret
	} else {
		ret.Type = NATIVE_TOKEN
		ret.DenomOrAddress = assets.NativeToken.Denom
	}

	return ret
}

type PoolInfo struct {
	Assets     [2]PoolAssetInfo `json:"assets"`
	TotalShare *math.Int        `json:"total_share"`
}

type PoolAssetInfo struct {
	Info   AssetInfo `json:"info"`
	Amount *math.Int `json:"amount"`
}

type PoolAssetInfoWithType struct {
	Info   *AssetInfoWithType `json:"info"`
	Amount *math.Int          `json:"amount"`
}

func (pool *PoolAssetInfo) GetInfo() *PoolAssetInfoWithType {
	return &PoolAssetInfoWithType{
		Info:   pool.Info.GetInfo(),
		Amount: pool.Amount,
	}
}

type PoolInfoDTO struct {
	Assets     [2]PoolAssetInfoWithType `json:"assets"`
	TotalShare *math.Int                `json:"total_share"`
}

func (pool *PoolInfo) Convert() *PoolInfoDTO {
	ret := &PoolInfoDTO{}

	ret.Assets = [2]PoolAssetInfoWithType{
		*pool.Assets[0].GetInfo(),
		*pool.Assets[1].GetInfo(),
	}

	ret.TotalShare = pool.TotalShare

	return ret
}

type PoolInfoWithLpAddr struct {
	PoolInfoDTO
	LpAddr string `json:"lp_addr"`
}

type PoolInfoList struct {
	Pairs map[string]PoolInfoWithLpAddr `json:"pairs"`
}
