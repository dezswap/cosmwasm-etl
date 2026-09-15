package classicv2

import (
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/lcd"
)

// Classic v2 and terra 2.0 answer queries the same way today, so this stays a thin
// wrapper over the shared client until one of them diverges.
func NewClient(lcd lcd.Lcd[cosmos45.LcdTxRes]) terra.QueryClient {
	return terra.NewCosmos45Client(lcd)
}
