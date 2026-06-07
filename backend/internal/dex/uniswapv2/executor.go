package uniswapv2

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"copyflow/internal/chain"
	"copyflow/internal/dex"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Executor 通过 V2 Router 执行跟单买入。
type Executor struct {
	dexType       string
	router        common.Address
	wrappedNative common.Address
	routerABI     abi.ABI
}

// NewExecutor 创建 V2 DEX 跟单执行器。
func NewExecutor(dexType, router, wrappedNative string) (*Executor, error) {
	parsed, err := abi.JSON(strings.NewReader(RouterABI))
	if err != nil {
		return nil, err
	}
	return &Executor{
		dexType:       dexType,
		router:        common.HexToAddress(router),
		wrappedNative: common.HexToAddress(wrappedNative),
		routerABI:     parsed,
	}, nil
}

// Type 返回 DEX 类型标识。
func (e *Executor) Type() string { return e.dexType }

// ExecuteBuy 构造并广播跟单买入交易。
func (e *Executor) ExecuteBuy(ctx context.Context, client chain.Client, params dex.ExecuteParams) (common.Hash, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(params.PrivateKeyHex, "0x"))
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid private key: %w", err)
	}
	publicKeyECDSA, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return common.Hash{}, fmt.Errorf("invalid public key")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return common.Hash{}, err
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	path := params.Path
	if len(path) == 0 {
		path = []common.Address{e.wrappedNative, params.TokenOut}
	}

	// 通过 Router getAmountsOut 报价并应用滑点
	amountOutMin, err := e.quoteMinOut(ctx, client, params.AmountIn, path, params.SlippageBps)
	if err != nil {
		// 报价失败时降级为 0（MVP 容忍滑点，后续可改为直接失败）
		amountOutMin = big.NewInt(0)
	}
	deadline := big.NewInt(time.Now().Add(5 * time.Minute).Unix())

	data, err := e.routerABI.Pack("swapExactETHForTokens", amountOutMin, path, fromAddress, deadline)
	if err != nil {
		return common.Hash{}, err
	}

	tx := types.NewTransaction(nonce, e.router, params.AmountIn, 300000, gasPrice, data)
	chainID := big.NewInt(int64(client.ChainID()))
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return common.Hash{}, err
	}

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return common.Hash{}, err
	}
	return signedTx.Hash(), nil
}

// quoteMinOut 调用 Router.getAmountsOut 并按滑点计算 amountOutMin。
func (e *Executor) quoteMinOut(ctx context.Context, client chain.Client, amountIn *big.Int, path []common.Address, slippageBps int) (*big.Int, error) {
	data, err := e.routerABI.Pack("getAmountsOut", amountIn, path)
	if err != nil {
		return nil, err
	}
	router := e.router
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &router, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	values, err := e.routerABI.Unpack("getAmountsOut", out)
	if err != nil || len(values) == 0 {
		return nil, fmt.Errorf("unpack getAmountsOut failed")
	}
	amounts, ok := values[0].([]*big.Int)
	if !ok || len(amounts) == 0 {
		return nil, fmt.Errorf("invalid amounts")
	}
	expectedOut := amounts[len(amounts)-1]
	if slippageBps <= 0 {
		return expectedOut, nil
	}
	minOut := new(big.Int).Mul(expectedOut, big.NewInt(int64(10000-slippageBps)))
	minOut.Div(minOut, big.NewInt(10000))
	return minOut, nil
}
