package configs

import (
	"fmt"
)

const (
	defaultCollectorStartHeight          = 1
	defaultCollectorPollInterval         = 5
	defaultCollectorPoolSnapshotInterval = 1000
	defaultCollectorTipLagBlocks         = 2
)

type CollectorConfig struct {
	ChainId                    string     `mapstructure:"chainid"`
	PairFactoryContractAddress string     `mapstructure:"pair_factory_contract_address"`
	NodeConfig                 NodeConfig `mapstructure:"node"`
	FcdConfig                  FcdConfig  `mapstructure:"fcd"`
	StartHeight                uint64     `mapstructure:"start_height"`
	UntilHeight                uint64     `mapstructure:"until_height"`
	PollIntervalSec            uint64     `mapstructure:"poll_interval_sec"`
	// TipLagBlocks keeps the collector this many heights behind the node tip. The node
	// reports a height in /status before its block results are written, so following the
	// tip inclusive means retrying a height that is merely not ready yet. 0 disables it.
	TipLagBlocks         uint64 `mapstructure:"tip_lag_blocks"`
	PoolSnapshotInterval uint   `mapstructure:"pool_snapshot_interval"`
}

type FcdConfig struct {
	Url             string   `mapstructure:"url"`
	TargetAddresses []string `mapstructure:"target_addresses"`
}

func defaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		NodeConfig: NodeConfig{
			HttpClientConfig: defaultHttpClientConfig,
		},
		StartHeight:          defaultCollectorStartHeight,
		PollIntervalSec:      defaultCollectorPollInterval,
		TipLagBlocks:         defaultCollectorTipLagBlocks,
		PoolSnapshotInterval: defaultCollectorPoolSnapshotInterval,
	}
}

func (c CollectorConfig) Validate() error {
	if c.ChainId == "" {
		return fmt.Errorf("missing chain id: set collector.chainid")
	}

	if c.PairFactoryContractAddress == "" {
		return fmt.Errorf("missing pair factory contract address: set collector.pair_factory_contract_address")
	}

	if c.NodeConfig.RestClientConfig.RpcHost == "" {
		return fmt.Errorf("missing RPC host: set collector.node.rest.rpc")
	}

	if c.NodeConfig.RestClientConfig.LcdHost == "" {
		return fmt.Errorf("missing LCD host: set collector.node.rest.lcd")
	}

	if c.StartHeight == 0 {
		return fmt.Errorf("invalid start height: set collector.start_height to a value greater than 0")
	}

	if c.PollIntervalSec == 0 {
		return fmt.Errorf("invalid poll interval: set collector.poll_interval_sec to a value greater than 0")
	}

	return nil
}
