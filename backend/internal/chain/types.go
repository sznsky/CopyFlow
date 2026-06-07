// Package chain 抽象多链 EVM RPC 客户端，便于扩展新链。
package chain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Client EVM 链 RPC 操作接口，每条链一个实现。
type Client interface {
	ChainID() int
	Name() string
	NativeSymbol() string

	BlockNumber(ctx context.Context) (uint64, error)                                              // 最新区块高度
	BlockByNumber(ctx context.Context, number uint64) (*types.Block, error)                       // 按高度查区块
	TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error)  // 按哈希查交易
	TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error)             // 查交易回执
	SendTransaction(ctx context.Context, tx *types.Transaction) error                             // 广播交易
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)                   // 账户 nonce
	SuggestGasPrice(ctx context.Context) (*big.Int, error)                                      // 建议 Gas 价
	EstimateGas(ctx context.Context, msg interface{}) (uint64, error)                             // 估算 Gas
	CallContract(ctx context.Context, call interface{}, blockNumber *big.Int) ([]byte, error)     // 只读合约调用
	FilterLogs(ctx context.Context, query interface{}) ([]types.Log, error)                         // 过滤事件日志
}

// Registry 按 chainID 管理链客户端，扩展时注册新链即可。
type Registry struct {
	clients map[int]Client
}

// NewRegistry 创建空的链注册表。
func NewRegistry() *Registry {
	return &Registry{clients: make(map[int]Client)}
}

// Register 注册一条链的 RPC 客户端。
func (r *Registry) Register(c Client) {
	r.clients[c.ChainID()] = c
}

// Get 按 chainID 获取链客户端。
func (r *Registry) Get(chainID int) (Client, bool) {
	c, ok := r.clients[chainID]
	return c, ok
}

// All 返回所有已注册的链客户端。
func (r *Registry) All() []Client {
	out := make([]Client, 0, len(r.clients))
	for _, c := range r.clients {
		out = append(out, c)
	}
	return out
}
