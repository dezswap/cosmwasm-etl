package parser

import (
	"math/rand"
	"reflect"

	"github.com/bxcodec/faker/v3"
)

// FakerCustomGenerator ...
func FakerCustomGenerator() {
	parserTxTypes := []string{string(CreatePair), string(Provide), string(Swap),
		string(Withdraw), string(Transfer)}
	_ = faker.AddProvider("parserTxType", func(v reflect.Value) (interface{}, error) {
		t := parserTxTypes[rand.Intn(len(parserTxTypes))]
		return t, nil
	})
}

func FakeParserAssets() []Asset {
	assets := []Asset{}
	_ = faker.FakeData(&assets)
	return assets
}

func FakeParserPoolInfoTxs() []PoolInfo {
	poolInfos := []PoolInfo{}
	_ = faker.FakeData(&poolInfos)
	for idx := range poolInfos {
		for len(poolInfos[idx].Assets) < 2 {
			_ = faker.FakeData(&poolInfos[idx].Assets)
		}
		poolInfos[idx].Assets = poolInfos[idx].Assets[0:2]
	}
	return poolInfos
}

func FakeParserParsedTxs() []ParsedTx {
	txs := []ParsedTx{}
	_ = faker.FakeData(&txs)
	for idx := range txs {
		for len(txs[idx].Assets) < 2 {
			_ = faker.FakeData(&txs[idx].Assets)
		}
		txs[idx].Assets = txs[idx].Assets[0:2]
	}
	return txs
}

func FakeParserPairs() []Pair {
	pairs := []Pair{}
	_ = faker.FakeData(&pairs)
	for idx := range pairs {
		for len(pairs[idx].Assets) < 2 {
			_ = faker.FakeData(&pairs[idx].Assets)
		}
		pairs[idx].Assets = pairs[idx].Assets[0:2]
	}
	return pairs
}
