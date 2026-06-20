// Package model 定义数据库实体与业务常量。
package model

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	CopyModeFixed = "fixed" // 固定金额跟单
	CopyModeRatio = "ratio" // 等比例跟单

	DEXPancakeV2 = "pancake_v2" // PancakeSwap V2（BSC）
	DEXUniswapV2 = "uniswap_v2" // Uniswap V2（Ethereum）
	DEXUniswapV3 = "uniswap_v3" // 预留：Uniswap V3

	TradeStatusPending   = "pending"   // 待执行
	TradeStatusSubmitted = "submitted" // 已广播
	TradeStatusSuccess   = "success"   // 链上确认成功
	TradeStatusFailed    = "failed"    // 执行失败
	TradeStatusSkipped   = "skipped"   // 策略跳过
)

// User 用户表，支持邮箱或钱包登录。
type User struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	WalletAddress *string   `gorm:"size:42;uniqueIndex" json:"wallet_address"`
	Email         *string   `gorm:"size:128;uniqueIndex" json:"email"`
	PasswordHash  string    `gorm:"size:255" json:"-"`
	Nonce         string    `gorm:"size:64" json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EmailVerification 邮箱验证码记录。
type EmailVerification struct {
	ID        uint64    `gorm:"primaryKey"`
	Email     string    `gorm:"size:128;index"`
	Code      string    `gorm:"size:6"`
	Purpose   string    `gorm:"size:16"` // register
	ExpiresAt time.Time
	Used      bool      `gorm:"default:false"`
	CreatedAt time.Time
}

// CopyWallet 跟单专用钱包，私钥加密存储。
type CopyWallet struct {
	ID                    uint64    `gorm:"primaryKey" json:"id"`
	UserID                uint64    `gorm:"index;uniqueIndex:uk_user_chain" json:"user_id"`
	ChainID               int       `gorm:"uniqueIndex:uk_user_chain" json:"chain_id"`
	Address               string    `gorm:"size:42" json:"address"`
	EncryptedPrivateKey   string    `gorm:"type:text" json:"-"`
	IsActive              bool      `gorm:"default:true" json:"is_active"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// CopyConfig 跟单策略配置。
type CopyConfig struct {
	ID            uint64          `gorm:"primaryKey" json:"id"`
	UserID        uint64          `gorm:"index" json:"user_id"`
	ChainID       int             `json:"chain_id"`
	DEXType       string          `gorm:"size:32" json:"dex_type"`
	LeaderAddress string          `gorm:"size:42" json:"leader_address"`
	CopyMode      string          `gorm:"size:16;default:ratio" json:"copy_mode"`
	CopyAmount    decimal.Decimal `gorm:"type:decimal(36,18)" json:"copy_amount"`
	MaxPerTrade   decimal.Decimal `gorm:"type:decimal(36,18)" json:"max_per_trade"`
	SlippageBps   int             `gorm:"default:300" json:"slippage_bps"`
	IsActive      bool            `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// LeaderTrade 监听到的领头地址 DEX 买入交易。
type LeaderTrade struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	ChainID       int       `gorm:"uniqueIndex:uk_leader_trade" json:"chain_id"`
	LeaderAddress string    `gorm:"size:42;index" json:"leader_address"`
	TxHash        string    `gorm:"size:66;uniqueIndex:uk_leader_trade" json:"tx_hash"`
	DEXType       string    `gorm:"size:32" json:"dex_type"`
	TokenIn       string    `gorm:"size:42" json:"token_in"`
	TokenOut      string    `gorm:"size:42" json:"token_out"`
	AmountIn      string    `gorm:"size:78" json:"amount_in"`
	AmountOut     string    `gorm:"size:78" json:"amount_out"`
	BlockNumber   uint64    `json:"block_number"`
	DetectedAt    time.Time `json:"detected_at"`
}

// CopyTrade 用户跟单执行记录。
type CopyTrade struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	UserID        uint64    `gorm:"index" json:"user_id"`
	ConfigID      uint64    `json:"config_id"`
	LeaderTradeID uint64    `json:"leader_trade_id"`
	TxHash        *string   `gorm:"size:66" json:"tx_hash"`
	Status        string    `gorm:"size:16;default:pending" json:"status"`
	AmountIn      string    `gorm:"size:78" json:"amount_in"`
	TokenOut      string    `gorm:"size:42" json:"token_out"`
	GasUsed       *uint64   `json:"gas_used"`
	ErrorMsg      *string   `gorm:"type:text" json:"error_msg"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	LeaderTrade *LeaderTrade `gorm:"foreignKey:LeaderTradeID" json:"leader_trade,omitempty"`
}

// ChainCursor 区块扫描游标，记录每条链已处理到的区块高度。
type ChainCursor struct {
	ID        uint64    `gorm:"primaryKey"`
	ChainID   int       `gorm:"uniqueIndex"`
	LastBlock uint64    `gorm:"default:0"`
	UpdatedAt time.Time
}

// SmartWallet 聪明钱包评分表。
type SmartWallet struct {
	ID            uint64          `gorm:"primaryKey" json:"id"`
	WalletAddress string          `gorm:"size:42;uniqueIndex:uk_wallet_chain" json:"wallet_address"`
	ChainID       int             `gorm:"uniqueIndex:uk_wallet_chain" json:"chain_id"`
	Score         decimal.Decimal `gorm:"type:decimal(10,2);default:0" json:"score"`
	TotalPNL      decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"total_pnl"`
	WinRate       decimal.Decimal `gorm:"type:decimal(5,2);default:0" json:"win_rate"`
	ProfitLossRatio decimal.Decimal `gorm:"type:decimal(10,2);default:0" json:"profit_loss_ratio"`
	MaxDrawdown   decimal.Decimal `gorm:"type:decimal(10,2);default:0" json:"max_drawdown"`
	MainstreamRatio decimal.Decimal `gorm:"type:decimal(5,2);default:0" json:"mainstream_ratio"`
	TradeFrequency decimal.Decimal `gorm:"type:decimal(10,2);default:0" json:"trade_frequency"`
	TotalTrades   int             `gorm:"default:0" json:"total_trades"`
	WinningTrades int             `gorm:"default:0" json:"winning_trades"`
	TotalVolume   decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"total_volume"`
	EvaluationStartDate time.Time `json:"evaluation_start_date"`
	EvaluationEndDate   time.Time `json:"evaluation_end_date"`
	RankPosition  *int            `json:"rank_position"`
	IsTopWallet   bool            `gorm:"default:false" json:"is_top_wallet"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// WalletTrade 钱包交易历史（从 Dune Analytics 同步）。
type WalletTrade struct {
	ID              uint64          `gorm:"primaryKey" json:"id"`
	WalletAddress   string          `gorm:"size:42;index:idx_wallet" json:"wallet_address"`
	ChainID         int             `gorm:"index:idx_chain" json:"chain_id"`
	TxHash          string          `gorm:"size:66;uniqueIndex:uk_tx" json:"tx_hash"`
	BlockNumber     uint64          `json:"block_number"`
	BlockTime       time.Time       `gorm:"index:idx_block_time" json:"block_time"`
	DEXName         string          `gorm:"size:32" json:"dex_name"`
	DEXVersion      string          `gorm:"size:8;column:dex_version" json:"dex_version"`
	PoolAddress     string          `gorm:"size:42" json:"pool_address"`
	TokenIn         string          `gorm:"size:42" json:"token_in"`
	TokenOut        string          `gorm:"size:42;uniqueIndex:uk_tx" json:"token_out"`
	TokenInSymbol   *string         `gorm:"size:32" json:"token_in_symbol"`
	TokenOutSymbol  *string         `gorm:"size:32" json:"token_out_symbol"`
	AmountIn        decimal.Decimal `gorm:"type:decimal(36,18)" json:"amount_in"`
	AmountOut       decimal.Decimal `gorm:"type:decimal(36,18)" json:"amount_out"`
	AmountUSD       decimal.Decimal `gorm:"type:decimal(36,2);index:idx_amount" json:"amount_usd"`
	IsBuy           bool            `json:"is_buy"`
	PnlUSD          *decimal.Decimal `gorm:"column:pnl_usd;type:decimal(36,18)" json:"pnl_usd"`
	PnlPercent      *decimal.Decimal `gorm:"column:pnl_percent;type:decimal(10,2)" json:"pnl_percent"`
	HoldingDurationHours *int       `json:"holding_duration_hours"`
	CreatedAt       time.Time       `json:"created_at"`
}

// TokenSignal 代币信号聚合。
type TokenSignal struct {
	ID                uint64          `gorm:"primaryKey" json:"id"`
	TokenAddress      string          `gorm:"size:42;uniqueIndex:uk_token_chain_period" json:"token_address"`
	TokenSymbol       *string         `gorm:"size:32" json:"token_symbol"`
	TokenName         *string         `gorm:"size:128" json:"token_name"`
	ChainID           int             `gorm:"uniqueIndex:uk_token_chain_period;index:idx_chain" json:"chain_id"`
	SmartWalletCount  int             `gorm:"default:0;index:idx_wallet_count" json:"smart_wallet_count"`
	TotalBuyVolume    decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"total_buy_volume"`
	AvgBuyAmount      decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"avg_buy_amount"`
	FirstBuyTime      time.Time       `json:"first_buy_time"`
	LastBuyTime       time.Time       `json:"last_buy_time"`
	ConsensusScore    decimal.Decimal `gorm:"type:decimal(10,2);default:0;index:idx_consensus" json:"consensus_score"`
	PriceUSD          *decimal.Decimal `gorm:"type:decimal(36,18)" json:"price_usd"`
	MarketCap         *decimal.Decimal `gorm:"type:decimal(36,2)" json:"market_cap"`
	LiquidityUSD      *decimal.Decimal `gorm:"type:decimal(36,2)" json:"liquidity_usd"`
	SignalStartDate   time.Time       `gorm:"uniqueIndex:uk_token_chain_period" json:"signal_start_date"`
	SignalEndDate     time.Time       `json:"signal_end_date"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// TokenSignalDetail 代币信号详情。
type TokenSignalDetail struct {
	ID            uint64          `gorm:"primaryKey" json:"id"`
	SignalID      uint64          `gorm:"index:idx_signal" json:"signal_id"`
	WalletAddress string          `gorm:"size:42;index:idx_wallet" json:"wallet_address"`
	WalletScore   decimal.Decimal `gorm:"type:decimal(10,2)" json:"wallet_score"`
	TradeID       uint64          `json:"trade_id"`
	BuyAmountUSD  decimal.Decimal `gorm:"type:decimal(36,18)" json:"buy_amount_usd"`
	BuyTime       time.Time       `json:"buy_time"`
	CreatedAt     time.Time       `json:"created_at"`
}

// DuneSyncLog Dune Analytics 数据同步记录。
type DuneSyncLog struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	QueryID         string    `gorm:"size:64;index:idx_query" json:"query_id"`
	QueryName       *string   `gorm:"size:128" json:"query_name"`
	ChainID         int       `json:"chain_id"`
	SyncType        string    `gorm:"size:32;index:idx_sync_type" json:"sync_type"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	RecordsFetched  int       `gorm:"default:0" json:"records_fetched"`
	RecordsInserted int       `gorm:"default:0" json:"records_inserted"`
	RecordsUpdated  int       `gorm:"default:0" json:"records_updated"`
	Status          string    `gorm:"size:16;index:idx_status" json:"status"`
	ErrorMessage    *string   `gorm:"type:text" json:"error_message"`
	CreatedAt       time.Time `json:"created_at"`
}
