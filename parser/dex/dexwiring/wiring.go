package dexwiring

import (
	"fmt"

	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	collectorrepo "github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/configs"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	pds "github.com/dezswap/cosmwasm-etl/parser/dex/dezswap"
	"github.com/dezswap/cosmwasm-etl/parser/dex/srcstore"
	ts_srcstore "github.com/dezswap/cosmwasm-etl/parser/dex/srcstore/terraswap"
	psf "github.com/dezswap/cosmwasm-etl/parser/dex/starfleit"
	pts "github.com/dezswap/cosmwasm-etl/parser/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/httpclient"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/s3client"
)

// NewTargetApp builds the configured DEX target parser used by parser commands.
func NewTargetApp(repo p_dex.PairRepo, logger logging.Logger, c configs.ParserDexConfig) (p_dex.TargetApp, error) {
	switch c.TargetApp {
	case dex.Terraswap:
		return pts.New(repo, logger, c)
	case dex.Dezswap:
		return pds.New(repo, logger, c, c.ChainId)
	case dex.Starfleit:
		return psf.New(repo, logger, c, c.ChainId)
	default:
		return nil, fmt.Errorf("unknown target app: %s", c.TargetApp)
	}
}

// NewTargetReadStore builds the collector-backed raw read store required by the configured DEX.
func NewTargetReadStore(c configs.Config, dc configs.ParserDexConfig) (datastore.ReadStore, error) {
	switch dc.TargetApp {
	case dex.Terraswap:
		return nil, nil
	case dex.Dezswap, dex.Starfleit:
		return NewCollectorReadStore(c, dc)
	default:
		return nil, fmt.Errorf("unknown target app: %s", dc.TargetApp)
	}
}

// NewCollectorReadStore mirrors the production parser source selection for collector-backed DEXes.
func NewCollectorReadStore(c configs.Config, dc configs.ParserDexConfig) (datastore.ReadStore, error) {
	nodeConf := dc.NodeConfig
	if nodeConf.GrpcConfig.Host != "" {
		serviceDesc := grpc.GetServiceDesc("collector", nodeConf.GrpcConfig)

		store, err := datastore.New(c, serviceDesc, nil)
		if err != nil {
			return nil, err
		}
		if nodeConf.FailoverLcdHost != "" {
			failoverStore, err := datastore.New(
				c,
				serviceDesc,
				datastore.NewLcdClient(nodeConf.FailoverLcdHost, httpclient.New(dc.NodeConfig.HttpClientConfig)),
			)
			if err != nil {
				return nil, err
			}
			store = failoverStore
		}

		return datastore.NewReadStoreWithGrpc(dc.ChainId, store), nil
	}

	s3Client, err := s3client.NewClient(c.S3)
	if err != nil {
		return nil, err
	}
	return datastore.NewReadStore(dc.ChainId, s3Client), nil
}

// NewSourceDataStore builds the raw transaction source used by parser commands.
func NewSourceDataStore(dc configs.ParserDexConfig, rdbc configs.RdbConfig, readStore datastore.ReadStore, logger logging.Logger) (p_dex.SourceDataStore, error) {
	switch dc.TargetApp {
	case dex.Terraswap:
		fallback, err := ts_srcstore.NewFromConfig(dc.NodeConfig, dc.FactoryAddress)
		if err != nil {
			return nil, err
		}
		return srcstore.NewCollectorFallback(dc.ChainId, collectorrepo.New(rdbc), fallback, logger), nil
	case dex.Dezswap, dex.Starfleit:
		if readStore == nil {
			return nil, fmt.Errorf("collector read store is required for target app: %s", dc.TargetApp)
		}
		return srcstore.New(readStore), nil
	default:
		return nil, fmt.Errorf("unknown target app: %s", dc.TargetApp)
	}
}
