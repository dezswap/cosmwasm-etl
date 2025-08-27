package terraswap

import (
	"github.com/dezswap/cosmwasm-etl/pkg/terra/lcd"

	pdex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
	"github.com/pkg/errors"
)

type col5ChainDataAdapter struct {
	factoryAddress string
	mapper
	rpc rpc.Rpc
	lcd lcd.Lcd[cosmos45.LcdTxRes]
	terraswap.QueryClient
}

var _ chainDataAdapter = &col5ChainDataAdapter{}

func NewCol5Store(factoryAddress string, rpc rpc.Rpc, lcd lcd.Lcd[cosmos45.LcdTxRes], client terraswap.QueryClient) pdex.SourceDataStore {
	return NewBaseStore(
		rpc,
		client,
		&col5ChainDataAdapter{
			factoryAddress: factoryAddress,
			mapper:         &mapperImpl{},
			rpc:            rpc,
			lcd:            lcd,
			QueryClient:    client,
		})
}

func (a *col5ChainDataAdapter) AllPairs(height uint64) ([]pdex.Pair, error) {
	var pairs []pdex.Pair
	var startAfter []dex.AssetInfo = nil
	for {
		factoryRes, err := a.QueryPairs(a.factoryAddress, startAfter, height)
		if err != nil {
			return nil, errors.Wrap(err, "col5ChainDataAdapter.AllPairs")
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

func (a *col5ChainDataAdapter) TxSenderOf(hash string) (string, error) {
	res, err := a.lcd.Tx(hash)
	if err != nil {
		return "", errors.Wrap(err, "col5ChainDataAdapter.TxSenderOf")
	}

	for _, msg := range res.Tx.Body.Messages {
		if msg.Type == "/cosmwasm.wasm.v1.MsgExecuteContract" {
			return msg.Sender, nil
		}
	}

	return "", nil
}
