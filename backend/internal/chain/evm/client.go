// Package evm 基于 go-ethereum 的通用 EVM 链客户端实现。
package evm

import (
	"context"
	"fmt"
	"math/big"

	"copyflow/internal/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client 封装 ethclient，适用于 BSC、Ethereum 等 EVM 链。
type Client struct {
	chainID      int
	name         string
	nativeSymbol string
	rpc          *ethclient.Client
}

// NewClient 连接 RPC 并创建链客户端。
func NewClient(cfg config.ChainConfig) (*Client, error) {
	rpc, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("dial %s rpc: %w", cfg.Name, err)
	}
	return &Client{
		chainID:      cfg.ChainID,
		name:         cfg.Name,
		nativeSymbol: cfg.NativeSymbol,
		rpc:          rpc,
	}, nil
}

// ChainID 返回链 ID。
func (c *Client) ChainID() int { return c.chainID }

// Name 返回链名称。
func (c *Client) Name() string { return c.name }

// NativeSymbol 返回原生代币符号（如 BNB、ETH）。
func (c *Client) NativeSymbol() string { return c.nativeSymbol }

// Raw 返回底层 ethclient，供高级场景使用。
func (c *Client) Raw() *ethclient.Client { return c.rpc }

// BlockNumber 获取最新区块高度。
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	return c.rpc.BlockNumber(ctx)
}

// BlockByNumber 按区块号获取区块详情。
func (c *Client) BlockByNumber(ctx context.Context, number uint64) (*types.Block, error) {
	return c.rpc.BlockByNumber(ctx, new(big.Int).SetUint64(number))
}

// TransactionByHash 按哈希查询交易。
func (c *Client) TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	return c.rpc.TransactionByHash(ctx, hash)
}

// TransactionReceipt 获取交易回执。
func (c *Client) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	return c.rpc.TransactionReceipt(ctx, hash)
}

// SendTransaction 广播已签名交易。
func (c *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return c.rpc.SendTransaction(ctx, tx)
}

// PendingNonceAt 获取账户 pending nonce。
func (c *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return c.rpc.PendingNonceAt(ctx, account)
}

// SuggestGasPrice 获取建议 Gas 价格。
func (c *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return c.rpc.SuggestGasPrice(ctx)
}

// EstimateGas 估算交易 Gas 用量。
func (c *Client) EstimateGas(ctx context.Context, msg interface{}) (uint64, error) {
	call, ok := msg.(ethereum.CallMsg)
	if !ok {
		return 0, fmt.Errorf("invalid call msg type")
	}
	return c.rpc.EstimateGas(ctx, call)
}

// CallContract 只读调用合约。
func (c *Client) CallContract(ctx context.Context, call interface{}, blockNumber *big.Int) ([]byte, error) {
	msg, ok := call.(ethereum.CallMsg)
	if !ok {
		return nil, fmt.Errorf("invalid call msg type")
	}
	return c.rpc.CallContract(ctx, msg, blockNumber)
}

// FilterLogs 按条件过滤事件日志。
func (c *Client) FilterLogs(ctx context.Context, query interface{}) ([]types.Log, error) {
	q, ok := query.(ethereum.FilterQuery)
	if !ok {
		return nil, fmt.Errorf("invalid filter query type")
	}
	return c.rpc.FilterLogs(ctx, q)
}
