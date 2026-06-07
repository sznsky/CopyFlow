// Package dex 抽象 DEX 交易解析与执行，便于扩展 V3、聚合器等。
package dex

import (
	"context"
	"math/big"

	"copyflow/internal/chain"
	"copyflow/internal/model"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// SwapInfo 从领头交易中解析出的 Swap 信息。
type SwapInfo struct {
	DEXType   string
	TokenIn   common.Address
	TokenOut  common.Address
	AmountIn  *big.Int
	AmountOut *big.Int
	Path      []common.Address
	IsBuy     bool // 是否为买入（原生币/稳定币 -> 代币）
}

// Parser 解析 DEX Swap 交易，每种 DEX 一个实现。
type Parser interface {
	Type() string
	RouterAddress() common.Address
	WrappedNative() common.Address

	// CanParse 判断交易是否发往本 DEX Router。
	CanParse(tx *types.Transaction) bool

	// ParseSwap 从交易 input 和 receipt 日志解析 Swap 信息。
	ParseSwap(ctx context.Context, client chain.Client, tx *types.Transaction, receipt *types.Receipt) (*SwapInfo, error)
}

// Executor 构造并发送跟单 Swap 交易。
type Executor interface {
	Type() string
	ExecuteBuy(ctx context.Context, client chain.Client, params ExecuteParams) (common.Hash, error) // 执行跟单买入
}

// ExecuteParams 跟单买入执行参数。
type ExecuteParams struct {
	PrivateKeyHex   string           // 跟单钱包私钥（hex）
	TokenOut        common.Address   // 目标代币
	AmountIn        *big.Int         // 投入原生币数量（wei）
	SlippageBps     int              // 滑点（基点，300 = 3%）
	LeaderAmountOut *big.Int         // 领头交易产出量，用于估算 minOut
	Path            []common.Address // Swap 路径
}

// Registry 管理 dex_type 到 Parser/Executor 的映射。
type Registry struct {
	parsers   map[string]Parser
	executors map[string]Executor
	byRouter  map[int]map[common.Address]Parser // chainID -> router -> parser
}

// NewRegistry 创建空的 DEX 注册表。
func NewRegistry() *Registry {
	return &Registry{
		parsers:   make(map[string]Parser),
		executors: make(map[string]Executor),
		byRouter:  make(map[int]map[common.Address]Parser),
	}
}

// RegisterParser 注册 DEX 解析器，并按 Router 地址建立索引。
func (r *Registry) RegisterParser(chainID int, p Parser) {
	r.parsers[p.Type()] = p
	if r.byRouter[chainID] == nil {
		r.byRouter[chainID] = make(map[common.Address]Parser)
	}
	r.byRouter[chainID][p.RouterAddress()] = p
}

// RegisterExecutor 注册 DEX 交易执行器。
func (r *Registry) RegisterExecutor(e Executor) {
	r.executors[e.Type()] = e
}

// Parser 按 DEX 类型获取解析器。
func (r *Registry) Parser(dexType string) (Parser, bool) {
	p, ok := r.parsers[dexType]
	return p, ok
}

// Executor 按 DEX 类型获取执行器。
func (r *Registry) Executor(dexType string) (Executor, bool) {
	e, ok := r.executors[dexType]
	return e, ok
}

// ParserByRouter 按链 ID 和 Router 合约地址查找解析器。
func (r *Registry) ParserByRouter(chainID int, router common.Address) (Parser, bool) {
	m, ok := r.byRouter[chainID]
	if !ok {
		return nil, false
	}
	p, ok := m[router]
	return p, ok
}

// LeaderTradeFromSwap 将 SwapInfo 转为数据库模型。
func LeaderTradeFromSwap(chainID int, leader string, txHash string, block uint64, swap *SwapInfo) *model.LeaderTrade {
	return &model.LeaderTrade{
		ChainID:       chainID,
		LeaderAddress: leader,
		TxHash:        txHash,
		DEXType:       swap.DEXType,
		TokenIn:       swap.TokenIn.Hex(),
		TokenOut:      swap.TokenOut.Hex(),
		AmountIn:      swap.AmountIn.String(),
		AmountOut:     swap.AmountOut.String(),
		BlockNumber:   block,
	}
}
