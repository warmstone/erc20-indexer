package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Chain    ChainConfig    `mapstructure:"chain"`
	Tokens   []string       `mapstructure:"tokens"`
	Database DatabaseConfig `mapstructure:"database"`
	API      APIConfig      `mapstructure:"api"`
}

type ChainConfig struct {
	ChainID       int64         `mapstructure:"chain_id"`
	RPCURL        string        `mapstructure:"rpc_url"`
	StartBlock    uint64        `mapstructure:"start_block"`
	Confirmations uint64        `mapstructure:"confirmations"`
	BatchSize     uint64        `mapstructure:"batch_size"`
	MinBatchSize  uint64        `mapstructure:"min_batch_size"`
	PollInterval  time.Duration `mapstructure:"poll_interval"`
	LogInterval   time.Duration `mapstructure:"log_interval"`
	LogRetries    int           `mapstructure:"log_retries"`
	LogBackoff    time.Duration `mapstructure:"log_retry_backoff"`
	BlockInterval time.Duration `mapstructure:"block_interval"`
	BlockRetries  int           `mapstructure:"block_retries"`
	BlockBackoff  time.Duration `mapstructure:"block_retry_backoff"`
	ReorgDepth    uint64        `mapstructure:"reorg_depth"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

type APIConfig struct {
	Addr string `mapstructure:"addr"`
}

func Load(path string) (Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("chain.confirmations", 24)
	v.SetDefault("chain.batch_size", 1000)
	v.SetDefault("chain.poll_interval", "10s")
	v.SetDefault("chain.reorg_depth", 64)
	v.SetDefault("api.addr", ":8080")

	if err := v.ReadInConfig(); err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
