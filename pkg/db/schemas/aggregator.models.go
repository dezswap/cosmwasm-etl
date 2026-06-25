package schemas

import (
	"time"

	"github.com/lib/pq"

	"github.com/dezswap/cosmwasm-etl/pkg/util"
)

type Account struct {
	Id        uint64
	Address   string
	CreatedAt float64
}

type LpHistory struct {
	Height     uint64  `json:"height"`
	PairId     uint64  `json:"pair_id"`
	ChainId    string  `json:"chain_id"`
	Liquidity0 string  `json:"liquidity0"`
	Liquidity1 string  `json:"liquidity1"`
	Timestamp  float64 `json:"timestamp"`
}

type Price struct {
	Height       uint64 `json:"height"`
	ChainId      string `json:"chain_id"`
	TokenId      uint64 `json:"token_id"`
	Price        string `json:"price"`
	PriceTokenId uint64 `json:"price_token_id"`
	RouteId      uint64 `json:"route_id"`
	TxId         uint64 `json:"tx_id"`
}

type Route struct {
	ChainId  string         `json:"chain_id"`
	Asset0   string         `json:"asset0"`
	Asset1   string         `json:"asset1"`
	HopCount int            `json:"hopCount"`
	Route    pq.StringArray `gorm:"type:varchar[]" json:"route"`
}

type ParsedTxWithPrice struct {
	PairId            uint64  `json:"pair_id"`
	ChainId           string  `json:"chain_id"`
	Asset0Amount      string  `json:"asset0_amount"`
	Asset1Amount      string  `json:"asset1_amount"`
	Asset0Liquidity   string  `json:"asset0_liquidity"`
	Asset1Liquidity   string  `json:"asset1_liquidity"`
	Commission0Amount string  `json:"commission0_amount"`
	Commission1Amount string  `json:"commission1_amount"`
	Price0            string  `json:"price0"`
	Price1            string  `json:"price1"`
	Decimals0         int64   `json:"decimals0"`
	Decimals1         int64   `json:"decimals1"`
	Height            uint64  `json:"height"`
	Timestamp         float64 `json:"timestamp"`
}

type PairStatsRecent struct {
	PairId             uint64  `json:"pair_id"`
	ChainId            string  `json:"chain_id"`
	Volume0            string  `json:"volume0"`
	Volume1            string  `json:"volume1"`
	Volume0InPrice     string  `json:"volume0_in_price"`
	Volume1InPrice     string  `json:"volume1_in_price"`
	Liquidity0         string  `json:"liquidity0"`
	Liquidity1         string  `json:"liquidity1"`
	Liquidity0InPrice  string  `json:"liquidity0_in_price"`
	Liquidity1InPrice  string  `json:"liquidity1_in_price"`
	Commission0        string  `json:"commission0"`
	Commission1        string  `json:"commission1"`
	Commission0InPrice string  `json:"commission0_in_price"`
	Commission1InPrice string  `json:"commission1_in_price"`
	PriceToken         string  `json:"price_token"`
	Height             uint64  `json:"height"`
	Timestamp          float64 `json:"timestamp"`
}

type PairStats30m struct {
	YearUtc            int     `json:"year_utc"`
	MonthUtc           int     `json:"month_utc"`
	DayUtc             int     `json:"day_utc"`
	HourUtc            int     `json:"hour_utc"`
	MinuteUtc          int     `json:"minute_utc"`
	PairId             uint64  `json:"pair_id"`
	ChainId            string  `json:"chain_id"`
	Volume0            string  `json:"volume0"`
	Volume1            string  `json:"volume1"`
	Volume0InPrice     string  `json:"volume0_in_price"`
	Volume1InPrice     string  `json:"volume1_in_price"`
	LastSwapPrice      string  `json:"last_swap_price"`
	Liquidity0         string  `json:"liquidity0"`
	Liquidity1         string  `json:"liquidity1"`
	Liquidity0InPrice  string  `json:"liquidity0_in_price"`
	Liquidity1InPrice  string  `json:"liquidity1_in_price"`
	Commission0        string  `json:"commission0"`
	Commission1        string  `json:"commission1"`
	Commission0InPrice string  `json:"commission0_in_price"`
	Commission1InPrice string  `json:"commission1_in_price"`
	PriceToken         string  `json:"price_token"`
	TxCnt              int     `json:"tx_cnt"`
	ProviderCnt        uint64  `json:"provider_cnt"`
	Timestamp          float64 `json:"timestamp"`
}

type AccountStats30m struct {
	YearUtc              int     `json:"year_utc"`
	MonthUtc             int     `json:"month_utc"`
	DayUtc               int     `json:"day_utc"`
	HourUtc              int     `json:"hour_utc"`
	MinuteUtc            int     `json:"minute_utc"`
	AccountId            uint64  `json:"account_id"`
	Address              string  `json:"address"`
	PairId               uint64  `json:"pair_id"`
	ChainId              string  `json:"chain_id"`
	TxCnt                uint64  `json:"tx_cnt"`
	SwapTxCnt            uint64  `json:"swap_tx_cnt"`
	ProvideTxCnt         uint64  `json:"provide_tx_cnt"`
	WithdrawTxCnt        uint64  `json:"withdraw_tx_cnt"`
	SwapVolumeInPrice    string  `json:"swap_volume_in_price"`
	ProvideValueInPrice  string  `json:"provide_value_in_price"`
	WithdrawValueInPrice string  `json:"withdraw_value_in_price"`
	NetFlowInPrice       string  `json:"net_flow_in_price"`
	PriceToken           string  `json:"price_token"`
	NetAsset0Amount      string  `json:"net_asset0_amount"`
	NetAsset1Amount      string  `json:"net_asset1_amount"`
	NetLpAmount          string  `json:"net_lp_amount"`
	Timestamp            float64 `json:"timestamp"`
}

func NewPairStat30min(chainId string, priceToken string, end time.Time, pairId uint64) PairStats30m {
	return PairStats30m{
		YearUtc:    end.Year(),
		MonthUtc:   int(end.Month()),
		DayUtc:     end.Day(),
		HourUtc:    end.Hour(),
		MinuteUtc:  end.Minute(),
		PairId:     pairId,
		ChainId:    chainId,
		PriceToken: priceToken,
		Timestamp:  util.ToEpoch(end),
	}
}

func NewAccountStat30min(chainId string, end time.Time, pairId uint64, accountId uint64, accountAddress string) AccountStats30m {
	return AccountStats30m{
		YearUtc:              end.Year(),
		MonthUtc:             int(end.Month()),
		DayUtc:               end.Day(),
		HourUtc:              end.Hour(),
		MinuteUtc:            end.Minute(),
		Timestamp:            util.ToEpoch(end),
		AccountId:            accountId,
		Address:              accountAddress,
		PairId:               pairId,
		ChainId:              chainId,
		SwapVolumeInPrice:    "0",
		ProvideValueInPrice:  "0",
		WithdrawValueInPrice: "0",
		NetFlowInPrice:       "0",
		NetAsset0Amount:      "0",
		NetAsset1Amount:      "0",
		NetLpAmount:          "0",
	}
}
