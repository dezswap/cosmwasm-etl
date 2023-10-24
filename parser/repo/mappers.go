package repo

import (
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
)

type mapper interface {
	toParsedTxModel(chainId string, height uint64, tx parser.ParsedTx) schemas.ParsedTx
	toPoolInfoModel(chainId string, height uint64, pool parser.PoolInfo) schemas.PoolInfo
	toPairModel(chainId string, pair parser.Pair) schemas.Pair

	toPairDto(pair schemas.Pair) parser.Pair
}

var _ mapper = &parserMapperImpl{}

type parserMapperImpl struct{}

// toPairModel implements mapper
func (*parserMapperImpl) toPairModel(chainId string, pair parser.Pair) schemas.Pair {
	return schemas.Pair{
		ChainId:  chainId,
		Contract: pair.ContractAddr,
		Asset0:   pair.Assets[0],
		Asset1:   pair.Assets[1],
		Lp:       pair.LpAddr,
	}
}

// toPairDto implements mapper
func (*parserMapperImpl) toPairDto(pair schemas.Pair) parser.Pair {
	return parser.Pair{
		ContractAddr: pair.Contract,
		Assets:       []string{pair.Asset0, pair.Asset1},
		LpAddr:       pair.Lp,
	}
}

// toParsedTxModel implements mapper
func (p *parserMapperImpl) toParsedTxModel(chainId string, height uint64, tx parser.ParsedTx) schemas.ParsedTx {
	lpAmount := p.emptyStringToZero(tx.LpAmount)
	if tx.Type == parser.Withdraw {
		lpAmount = "-" + lpAmount
	}
	commission := p.emptyStringToZero(tx.CommissionAmount)
	commission0, commission1 := "0", "0"
	if strings.HasPrefix(tx.Assets[0].Amount, "-") {
		commission0 = commission
	} else {
		commission1 = commission
	}
	return schemas.ParsedTx{
		ChainId:           chainId,
		Height:            height,
		Timestamp:         float64(tx.Timestamp.UTC().Unix()),
		Type:              tx.Type,
		Hash:              tx.Hash,
		Contract:          tx.ContractAddr,
		Asset0:            tx.Assets[0].Addr,
		Asset0Amount:      p.emptyStringToZero(tx.Assets[0].Amount),
		Asset1:            tx.Assets[1].Addr,
		Asset1Amount:      p.emptyStringToZero(tx.Assets[1].Amount),
		Lp:                tx.LpAddr,
		LpAmount:          lpAmount,
		Sender:            tx.Sender,
		Commission0Amount: commission0,
		Commission1Amount: commission1,
		CommissionAmount:  commission,
		Meta:              tx.Meta,
	}
}

// toPoolInfoModel implements mapper
func (*parserMapperImpl) toPoolInfoModel(chainId string, height uint64, pool parser.PoolInfo) schemas.PoolInfo {
	return schemas.PoolInfo{
		ChainId:      chainId,
		Height:       height,
		Contract:     pool.ContractAddr,
		Asset0Amount: pool.Assets[0].Amount,
		Asset1Amount: pool.Assets[1].Amount,
		LpAmount:     pool.TotalShare,
	}
}

func (p *parserMapperImpl) emptyStringToZero(amount string) string {
	if amount == "" {
		return "0"
	}
	return amount
}
