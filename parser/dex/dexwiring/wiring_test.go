package dexwiring

import (
	"testing"

	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/configs"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	ts "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/stretchr/testify/require"
)

var chainFixtures = map[string]struct {
	factory         string
	collectorBacked bool
}{
	"columbus-5":     {factory: string(ts.CLASSIC_V2_FACTORY)},
	"phoenix-1":      {factory: string(ts.MAINNET_FACTORY)},
	"cube_47-5":      {factory: "xpla1j4kgjl6h4rt96uddtzdxdu39h0mhn4vrtydufdrk4uxxnrpsnw2qug2yx2", collectorBacked: true},
	"dimension_37-1": {factory: "xpla1j33xdql0h4kpgj2mhggy4vutw655u90z7nyj4afhxgj4v5urtadq44e3vd", collectorBacked: true},
	"dorado-1":       {factory: "fetch1kmag3937lrl6dtsv29mlfsedzngl9egv5c3apnr468q50gu04zrqea398u", collectorBacked: true},
	"fetchhub-4":     {factory: "fetch1slz6c85kxp4ek5ufmcakfhnscv9r2snlemxgwz6cjhklgh7v2hms8rgt5v", collectorBacked: true},
}

// A chain reaching configs.Validate but missing a branch in one of the three wiring
// switches only fails at startup, so each of them has to handle every known chain.
func Test_NewTargetApp_WiresKnownChains(t *testing.T) {
	for chainId, fixture := range chainFixtures {
		t.Run(chainId, func(t *testing.T) {
			repo := &p_dex.RepoMock{}
			repo.On("GetPairs").Return(map[string]p_dex.Pair{}, nil)

			app, err := NewTargetApp(repo, logging.Discard, configs.ParserDexConfig{
				ChainId:        chainId,
				FactoryAddress: fixture.factory,
			})
			require.NoError(t, err)
			require.NotNil(t, app)
		})
	}
}

func Test_NewTargetReadStore_WiresKnownChains(t *testing.T) {
	for chainId, fixture := range chainFixtures {
		t.Run(chainId, func(t *testing.T) {
			// no grpc host, so the collector-backed chains take the s3 path
			store, err := NewTargetReadStore(configs.Config{}, configs.ParserDexConfig{ChainId: chainId})
			require.NoError(t, err)

			if fixture.collectorBacked {
				require.NotNil(t, store)
				return
			}
			require.Nil(t, store)
		})
	}
}

func Test_NewSourceDataStore_WiresKnownChains(t *testing.T) {
	for chainId, fixture := range chainFixtures {
		t.Run(chainId, func(t *testing.T) {
			dc := configs.ParserDexConfig{ChainId: chainId, FactoryAddress: fixture.factory}

			if !fixture.collectorBacked {
				// the terraswap branch dials the collector DB, which a unit test cannot provide.
				// An unusable factory address fails inside the branch, which still tells it apart
				// from the default one.
				dc.FactoryAddress = "terra1nope"
				_, err := NewSourceDataStore(dc, configs.RdbConfig{}, nil, logging.Discard)
				require.EqualError(t, err, "invalid factory address: terra1nope")
				return
			}

			_, err := NewSourceDataStore(dc, configs.RdbConfig{}, nil, logging.Discard)
			require.EqualError(t, err, "collector read store is required for chain id: "+chainId)

			store, err := NewSourceDataStore(dc, configs.RdbConfig{}, &datastore.ReadStoreMock{}, logging.Discard)
			require.NoError(t, err)
			require.NotNil(t, store)
		})
	}
}

func Test_Wiring_RejectsUnknownChainId(t *testing.T) {
	dc := configs.ParserDexConfig{ChainId: "mars-1", FactoryAddress: "terra1factory"}

	app, err := NewTargetApp(&p_dex.RepoMock{}, logging.Discard, dc)
	require.EqualError(t, err, "unsupported chain id: mars-1")
	require.Nil(t, app)

	readStore, err := NewTargetReadStore(configs.Config{}, dc)
	require.EqualError(t, err, "unsupported chain id: mars-1")
	require.Nil(t, readStore)

	source, err := NewSourceDataStore(dc, configs.RdbConfig{}, nil, logging.Discard)
	require.EqualError(t, err, "unsupported chain id: mars-1")
	require.Nil(t, source)
}
