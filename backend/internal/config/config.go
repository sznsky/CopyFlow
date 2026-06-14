// Package config 加载应用配置，支持 YAML 文件与环境变量覆盖。
package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config 应用总配置，链和 DEX 可通过配置扩展。
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Auth       AuthConfig
	Email      EmailConfig
	Worker     WorkerConfig
	Dune       DuneConfig
	SmartMoney SmartMoneyConfig
	Chains     []ChainConfig
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Port string
	Mode string // debug | release
}

// DatabaseConfig 数据库连接配置。
type DatabaseConfig struct {
	DSN string
}

// AuthConfig 认证与加密配置。
type AuthConfig struct {
	JWTSecret          string
	WalletEncryptKey   string // AES-256 密钥，须 32 字节
	NonceExpireMinutes int
}

// EmailConfig 邮件发送配置（未启用时验证码打印到日志）。
type EmailConfig struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// WorkerConfig 链上监听 Worker 配置。
type WorkerConfig struct {
	Enabled         bool
	PollIntervalSec int
	Confirmations   uint64
}

// DuneConfig Dune Analytics API 配置。
type DuneConfig struct {
	APIKey  string
	QueryID int
	Enabled bool
}

// SmartMoneyConfig 聪明钱分析配置。
type SmartMoneyConfig struct {
	Enabled          bool
	ChainID          int
	MinAmountUSD     float64
	TopWalletCount   int
	MinWalletScore   float64
	SignalDays       int
	SyncIntervalHours int
}

// ChainConfig 单条链的 RPC 与 DEX 配置。
type ChainConfig struct {
	ChainID     int
	Name        string
	RPCURL      string
	Enabled     bool
	NativeSymbol string
	DEXes       []DEXConfig
}

// DEXConfig 单个 DEX 的合约地址配置。
type DEXConfig struct {
	Type           string // pancake_v2, uniswap_v2, uniswap_v3
	RouterAddress  string
	FactoryAddress string
	WrappedNative  string
	Enabled        bool
}

// Load 从 YAML 和环境变量加载配置。
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("./backend/config")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults()
	_ = viper.ReadInConfig()

	cfg := &Config{
		Server: ServerConfig{
			Port: viper.GetString("server.port"),
			Mode: viper.GetString("server.mode"),
		},
		Database: DatabaseConfig{
			DSN: viper.GetString("database.dsn"),
		},
		Auth: AuthConfig{
			JWTSecret:          viper.GetString("auth.jwt_secret"),
			WalletEncryptKey:   viper.GetString("auth.wallet_encrypt_key"),
			NonceExpireMinutes: viper.GetInt("auth.nonce_expire_minutes"),
		},
		Email: EmailConfig{
			Enabled:  viper.GetBool("email.enabled"),
			Host:     viper.GetString("email.host"),
			Port:     viper.GetInt("email.port"),
			Username: viper.GetString("email.username"),
			Password: viper.GetString("email.password"),
			From:     viper.GetString("email.from"),
		},
		Worker: WorkerConfig{
			Enabled:         viper.GetBool("worker.enabled"),
			PollIntervalSec: viper.GetInt("worker.poll_interval_sec"),
			Confirmations:   viper.GetUint64("worker.confirmations"),
		},
		Dune: DuneConfig{
			APIKey:  viper.GetString("dune.api_key"),
			QueryID: viper.GetInt("dune.query_id"),
			Enabled: viper.GetBool("dune.enabled"),
		},
		SmartMoney: SmartMoneyConfig{
			Enabled:          viper.GetBool("smartmoney.enabled"),
			ChainID:          viper.GetInt("smartmoney.chain_id"),
			MinAmountUSD:     viper.GetFloat64("smartmoney.min_amount_usd"),
			TopWalletCount:   viper.GetInt("smartmoney.top_wallet_count"),
			MinWalletScore:   viper.GetFloat64("smartmoney.min_wallet_score"),
			SignalDays:       viper.GetInt("smartmoney.signal_days"),
			SyncIntervalHours: viper.GetInt("smartmoney.sync_interval_hours"),
		},
	}

	cfg.Chains = loadChains()
	return cfg, nil
}

// setDefaults 设置配置项默认值。
func setDefaults() {
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.dsn", "copyflow:copyflow@tcp(127.0.0.1:3306)/copyflow?charset=utf8mb4&parseTime=True&loc=Local")
	viper.SetDefault("auth.jwt_secret", "change-me-in-production")
	viper.SetDefault("auth.wallet_encrypt_key", "0123456789abcdef0123456789abcdef")
	viper.SetDefault("auth.nonce_expire_minutes", 10)
	viper.SetDefault("worker.enabled", true)
	viper.SetDefault("worker.poll_interval_sec", 3)
	viper.SetDefault("worker.confirmations", 1)

	// Dune Analytics defaults
	viper.SetDefault("dune.enabled", false)
	viper.SetDefault("dune.query_id", 0)

	// Smart Money defaults
	viper.SetDefault("smartmoney.enabled", false)
	viper.SetDefault("smartmoney.chain_id", 1)
	viper.SetDefault("smartmoney.min_amount_usd", 1000)
	viper.SetDefault("smartmoney.top_wallet_count", 20)
	viper.SetDefault("smartmoney.min_wallet_score", 60)
	viper.SetDefault("smartmoney.signal_days", 3)
	viper.SetDefault("smartmoney.sync_interval_hours", 24)

	// BSC mainnet defaults (MVP primary chain)
	viper.SetDefault("chains.bsc.chain_id", 56)
	viper.SetDefault("chains.bsc.name", "BSC")
	viper.SetDefault("chains.bsc.rpc_url", "https://bsc-dataseed.binance.org")
	viper.SetDefault("chains.bsc.enabled", true)
	viper.SetDefault("chains.bsc.native_symbol", "BNB")
	viper.SetDefault("chains.bsc.dex.pancake_v2.router", "0x10ED43C718714eb63d5aA57B78B54704E256024E")
	viper.SetDefault("chains.bsc.dex.pancake_v2.factory", "0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73")
	viper.SetDefault("chains.bsc.dex.pancake_v2.wrapped_native", "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c")
	viper.SetDefault("chains.bsc.dex.pancake_v2.enabled", true)

	// Ethereum (stub-ready, disabled by default in MVP)
	viper.SetDefault("chains.ethereum.chain_id", 1)
	viper.SetDefault("chains.ethereum.name", "Ethereum")
	viper.SetDefault("chains.ethereum.rpc_url", "https://eth.llamarpc.com")
	viper.SetDefault("chains.ethereum.enabled", false)
	viper.SetDefault("chains.ethereum.native_symbol", "ETH")
	viper.SetDefault("chains.ethereum.dex.uniswap_v2.router", "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")
	viper.SetDefault("chains.ethereum.dex.uniswap_v2.factory", "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f")
	viper.SetDefault("chains.ethereum.dex.uniswap_v2.wrapped_native", "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	viper.SetDefault("chains.ethereum.dex.uniswap_v2.enabled", false)
}

// loadChains 从 viper 加载各链及 DEX 配置。
func loadChains() []ChainConfig {
	var chains []ChainConfig

	if viper.IsSet("chains.bsc.chain_id") || true {
		chains = append(chains, ChainConfig{
			ChainID:      viper.GetInt("chains.bsc.chain_id"),
			Name:         viper.GetString("chains.bsc.name"),
			RPCURL:       viper.GetString("chains.bsc.rpc_url"),
			Enabled:      viper.GetBool("chains.bsc.enabled"),
			NativeSymbol: viper.GetString("chains.bsc.native_symbol"),
			DEXes: []DEXConfig{
				{
					Type:           "pancake_v2",
					RouterAddress:  viper.GetString("chains.bsc.dex.pancake_v2.router"),
					FactoryAddress: viper.GetString("chains.bsc.dex.pancake_v2.factory"),
					WrappedNative:  viper.GetString("chains.bsc.dex.pancake_v2.wrapped_native"),
					Enabled:        viper.GetBool("chains.bsc.dex.pancake_v2.enabled"),
				},
			},
		})
	}

	chains = append(chains, ChainConfig{
		ChainID:      viper.GetInt("chains.ethereum.chain_id"),
		Name:         viper.GetString("chains.ethereum.name"),
		RPCURL:       viper.GetString("chains.ethereum.rpc_url"),
		Enabled:      viper.GetBool("chains.ethereum.enabled"),
		NativeSymbol: viper.GetString("chains.ethereum.native_symbol"),
		DEXes: []DEXConfig{
			{
				Type:           "uniswap_v2",
				RouterAddress:  viper.GetString("chains.ethereum.dex.uniswap_v2.router"),
				FactoryAddress: viper.GetString("chains.ethereum.dex.uniswap_v2.factory"),
				WrappedNative:  viper.GetString("chains.ethereum.dex.uniswap_v2.wrapped_native"),
				Enabled:        viper.GetBool("chains.ethereum.dex.uniswap_v2.enabled"),
			},
		},
	})

	return chains
}

// GetChain 按 chainID 查找链配置，未找到返回 nil。
func (c *Config) GetChain(chainID int) *ChainConfig {
	for i := range c.Chains {
		if c.Chains[i].ChainID == chainID {
			return &c.Chains[i]
		}
	}
	return nil
}
