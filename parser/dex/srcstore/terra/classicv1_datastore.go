package terra

import (
	"github.com/dezswap/cosmwasm-etl/pkg/terra/lcd"

	pdex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/col4"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
	"github.com/pkg/errors"
)

type classicV1ChainDataAdapter struct {
	factoryAddress string
	mapper
	rpc rpc.Rpc
	lcd lcd.Lcd[col4.LcdTxRes]
	terra.QueryClient
}

var _ chainDataAdapter = &classicV1ChainDataAdapter{}

func NewClassicV1Store(factoryAddress string, rpc rpc.Rpc, lcd lcd.Lcd[col4.LcdTxRes], client terra.QueryClient) pdex.SourceDataStore {
	return NewBaseStore(
		rpc,
		client,
		&classicV1ChainDataAdapter{
			factoryAddress: factoryAddress,
			mapper:         &mapperImpl{},
			rpc:            rpc,
			lcd:            lcd,
			QueryClient:    client,
		})
}

func (a *classicV1ChainDataAdapter) AllPairs(height uint64) ([]pdex.Pair, error) {
	var pairs []pdex.Pair
	var startAfter []dex.AssetInfo = nil
	for {
		factoryRes, err := a.QueryPairs(a.factoryAddress, startAfter, height)
		if err != nil {
			return nil, errors.Wrap(err, "classicV1ChainDataAdapter.AllPairs")
		}

		if len(factoryRes.Pairs) == 0 {
			break
		}

		for _, pair := range factoryRes.Pairs {
			p := a.dexPairToPair(&pair)
			pairs = append(pairs, p)
		}
		startAfter = factoryRes.Pairs[len(factoryRes.Pairs)-1].AssetInfos[:]
	}

	return pairs, nil
}

func (a *classicV1ChainDataAdapter) TxSenderOf(hash string) (string, error) {
	res, err := a.lcd.Tx(hash)
	if err != nil {
		return "", errors.Wrap(err, "classicV1ChainDataAdapter.TxSenderOf")
	}

	for _, msg := range res.Tx.Value.Msg {
		if msg.Type == col4.LCD_TERRA_TX_MSG_WASM_TYPE {
			return msg.Value.Sender, nil
		}
	}

	return "", nil
}
