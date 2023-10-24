package configs

import (
	"time"

	"github.com/spf13/viper"
)

type AggregatorConfig struct {
	ChainId    string
	PriceToken string
	StartTs    time.Time // e.g. 2006-01-02 15:04:05
	CleanDups  bool
	SrcDb      RdbConfig
	DestDb     RdbConfig
	Router     RouterConfig
}

func aggregatorConfig(v *viper.Viper) AggregatorConfig {
	return AggregatorConfig{
		ChainId:    v.GetString("aggregator.chainId"),
		PriceToken: v.GetString("aggregator.priceToken"),
		StartTs:    v.GetTime("aggregator.startTs"),
		CleanDups:  v.GetBool("aggregator.cleanDups"),
		SrcDb: RdbConfig{
			Host:     v.GetString("aggregator.srcDb.host"),
			Port:     v.GetInt("aggregator.srcDb.port"),
			Database: v.GetString("aggregator.srcDb.database"),
			Username: v.GetString("aggregator.srcDb.username"),
			Password: v.GetString("aggregator.srcDb.password"),
		},
		DestDb: RdbConfig{
			Host:     v.GetString("aggregator.destDb.host"),
			Port:     v.GetInt("aggregator.destDb.port"),
			Database: v.GetString("aggregator.destDb.database"),
			Username: v.GetString("aggregator.destDb.username"),
			Password: v.GetString("aggregator.destDb.password"),
		},
		Router: RouterConfig{
			Name:       v.GetString("aggregator.router.name"),
			RouterAddr: v.GetString("aggregator.router.router_addr"),
			MaxPathLen: v.GetUint("aggregator.router.max_path_len"),
			WriteDb:    v.GetBool("aggregator.router.write_db"),
		},
	}
}
