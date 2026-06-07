// Package uniswapv2 实现 Uniswap V2 风格 DEX 的解析与执行（兼容 PancakeSwap V2）。
package uniswapv2

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"copyflow/internal/chain"
	"copyflow/internal/dex"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Parser 解析 V2 Router 的 swap 交易 input data。
type Parser struct {
	dexType       string
	router        common.Address
	wrappedNative common.Address
	routerABI     abi.ABI
}

// NewParser 创建 V2 DEX 交易解析器。
func NewParser(dexType, router, wrappedNative string) (*Parser, error) {
	parsed, err := abi.JSON(strings.NewReader(RouterABI))
	if err != nil {
		return nil, err
	}
	return &Parser{
		dexType:       dexType,
		router:        common.HexToAddress(router),
		wrappedNative: common.HexToAddress(wrappedNative),
		routerABI:     parsed,
	}, nil
}

// Type 返回 DEX 类型标识。
func (p *Parser) Type() string { return p.dexType }

// RouterAddress 返回 Router 合约地址。
func (p *Parser) RouterAddress() common.Address { return p.router }

// WrappedNative 返回包装原生币地址（WBNB/WETH）。
func (p *Parser) WrappedNative() common.Address { return p.wrappedNative }

// CanParse 判断交易是否发往本 DEX Router。
func (p *Parser) CanParse(tx *types.Transaction) bool {
	if tx.To() == nil {
		return false
	}
	return strings.EqualFold(tx.To().Hex(), p.router.Hex())
}

// ParseSwap 解析 Swap 交易，提取路径、金额和买卖方向。
func (p *Parser) ParseSwap(ctx context.Context, client chain.Client, tx *types.Transaction, receipt *types.Receipt) (*dex.SwapInfo, error) {
	if !p.CanParse(tx) {
		return nil, fmt.Errorf("tx not for router %s", p.router.Hex())
	}
	method, err := p.routerABI.MethodById(tx.Data()[:4])
	if err != nil {
		return nil, fmt.Errorf("unknown method: %w", err)
	}

	var swap *dex.SwapInfo
	switch method.Name {
	case "swapExactETHForTokens":
		swap, err = p.parseETHForTokens(tx, method)
	case "swapExactTokensForTokens":
		swap, err = p.parseTokensForTokens(tx, method)
	default:
		return nil, fmt.Errorf("unsupported swap method: %s", method.Name)
	}
	if err != nil || swap == nil {
		return swap, err
	}

	// 从 receipt 补充实际买入数量
	if receipt != nil {
		chainID := tx.ChainId()
		if chainID == nil {
			chainID = big.NewInt(0)
		}
		signer := types.LatestSignerForChainID(chainID)
		recipient, err := types.Sender(signer, tx)
		if err == nil {
			swap.AmountOut = amountOutFromReceipt(receipt, recipient, swap.TokenOut)
		}
	}
	return swap, nil
}

// parseETHForTokens 解析 swapExactETHForTokens（BNB/ETH 买入代币）。
func (p *Parser) parseETHForTokens(tx *types.Transaction, method *abi.Method) (*dex.SwapInfo, error) {
	values, err := method.Inputs.Unpack(tx.Data()[4:])
	if err != nil {
		return nil, err
	}
	if len(values) < 2 {
		return nil, fmt.Errorf("invalid input")
	}
	path, ok := values[1].([]common.Address)
	if !ok || len(path) < 2 {
		return nil, fmt.Errorf("invalid path")
	}
	amountIn := tx.Value()
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, fmt.Errorf("zero value")
	}
	return &dex.SwapInfo{
		DEXType:   p.dexType,
		TokenIn:   path[0],
		TokenOut:  path[len(path)-1],
		AmountIn:  amountIn,
		AmountOut: big.NewInt(0), // filled from receipt if needed; MVP uses ratio on amountIn
		Path:      path,
		IsBuy:     true,
	}, nil
}

// parseTokensForTokens 解析 swapExactTokensForTokens。
func (p *Parser) parseTokensForTokens(tx *types.Transaction, method *abi.Method) (*dex.SwapInfo, error) {
	values, err := method.Inputs.Unpack(tx.Data()[4:])
	if err != nil {
		return nil, err
	}
	if len(values) < 3 {
		return nil, fmt.Errorf("invalid input")
	}
	amountIn, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid amountIn")
	}
	path, ok := values[2].([]common.Address)
	if !ok || len(path) < 2 {
		return nil, fmt.Errorf("invalid path")
	}
	isBuy := path[0] == p.wrappedNative
	return &dex.SwapInfo{
		DEXType:   p.dexType,
		TokenIn:   path[0],
		TokenOut:  path[len(path)-1],
		AmountIn:  amountIn,
		AmountOut: big.NewInt(0),
		Path:      path,
		IsBuy:     isBuy,
	}, nil
}
