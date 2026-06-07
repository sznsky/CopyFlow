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

// User 用户表，以钱包地址作为身份标识。
type User struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	WalletAddress string    `gorm:"size:42;uniqueIndex" json:"wallet_address"`
	Nonce         string    `gorm:"size:64" json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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
