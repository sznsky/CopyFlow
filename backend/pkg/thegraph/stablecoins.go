// Package thegraph 提供 The Graph 子图查询客户端。
package thegraph

import "strings"

// EthereumStablecoins 以太坊主网稳定币/锚定资产（小写地址）。
var EthereumStablecoins = map[string]struct{}{
	"0xdac17f958d2ee523a2206206994597c13d831ec7": {}, // USDT
	"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": {}, // USDC
	"0x6b175474e89094c44da98b954eedeac495271d0f": {}, // DAI
	"0x853d955acef822db058eb8501599331710176347": {}, // FRAX
	"0x0000000000085d4780b73119b46063785270":     {}, // TUSD
	"0x8e870d67f660d95d5652b2837392e3c7b313c110": {}, // USDP
	"0x056fd409e1d7a124bd7017632916c56a5208":     {}, // GUSD
	"0x57ab1ec28d129707052df4df087df1f3e33":     {}, // sUSD
	"0x5f98805a4e8be255a32880fdec7f6728c6568ba0": {}, // LUSD
	"0x6c3ea903640685950962c7dc2038c542e0b8ce33": {}, // PYUSD
	"0x4c9edd5852cd905f086c759e8383e09bff1e68b3": {}, // USDe
	"0x45804880de22913dafe09f4980848ece6ecbaf78": {}, // PAXG (gold)
	"0x68749665ff8d2b11230db6bc8a65314972109723": {}, // XAUT (Tether Gold)
}

// EthereumQuoteTokens 用于判断买入/卖出方向的报价资产。
var EthereumQuoteTokens = map[string]struct{}{
	"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {}, // WETH
}

func init() {
	// 合并稳定币到报价资产集合
	for addr := range EthereumStablecoins {
		EthereumQuoteTokens[addr] = struct{}{}
	}
}

func normalizeAddr(addr string) string {
	return strings.ToLower(strings.TrimSpace(addr))
}

// IsStablecoin 判断是否为稳定币/锚定资产。
func IsStablecoin(addr string) bool {
	_, ok := EthereumStablecoins[normalizeAddr(addr)]
	return ok
}

// IsQuoteToken 判断是否为报价资产（稳定币或 WETH）。
func IsQuoteToken(addr string) bool {
	_, ok := EthereumQuoteTokens[normalizeAddr(addr)]
	return ok
}

// IsStablecoinSwap 判断是否为稳定币之间的互换（应过滤）。
func IsStablecoinSwap(tokenIn, tokenOut string) bool {
	return IsStablecoin(tokenIn) && IsStablecoin(tokenOut)
}

// ClassifyDirection 判断交易方向（买入/卖出）。
// 买入：用报价资产换非报价资产；卖出：反向。
// 返回 isBuy 和 ok（ok=false 表示无法判断方向）。
func ClassifyDirection(tokenIn, tokenOut string) (isBuy bool, ok bool) {
	inQuote := IsQuoteToken(tokenIn)
	outQuote := IsQuoteToken(tokenOut)
	
	if inQuote && !outQuote {
		return true, true // 买入
	}
	if !inQuote && outQuote {
		return false, true // 卖出
	}
	return false, false // 无法判断
}
