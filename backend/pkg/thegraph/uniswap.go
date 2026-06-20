package thegraph

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// UniswapV2SwapsQuery Uniswap V2 Swaps GraphQL 查询。
const UniswapV2SwapsQuery = `
query GetSwaps($first: Int!, $skip: Int!, $timestampGte: Int!, $timestampLt: Int!, $minAmountUSD: BigDecimal!) {
  swaps(
    first: $first
    skip: $skip
    orderBy: timestamp
    orderDirection: asc
    where: {
      timestamp_gte: $timestampGte
      timestamp_lt: $timestampLt
      amountUSD_gte: $minAmountUSD
    }
  ) {
    id
    transaction {
      id
      timestamp
      blockNumber
    }
    timestamp
    pair {
      id
      token0 {
        id
        symbol
      }
      token1 {
        id
        symbol
      }
    }
    sender
    to
    amount0In
    amount1In
    amount0Out
    amount1Out
    amountUSD
  }
}
`

// UniswapV3SwapsQuery Uniswap V3 Swaps GraphQL 查询。
const UniswapV3SwapsQuery = `
query GetSwaps($first: Int!, $timestamp_gte: Int!, $timestamp_lt: Int!, $minAmountUSD: String!, $lastID: String!) {
  swaps(
    first: $first
    orderBy: timestamp
    orderDirection: asc
    where: {
      timestamp_gte: $timestamp_gte
      timestamp_lt: $timestamp_lt
      amountUSD_gte: $minAmountUSD
      id_gt: $lastID
    }
  ) {
    id
    transaction {
      id
      timestamp
      blockNumber
    }
    timestamp
    pool {
      id
      token0 {
        id
        symbol
      }
      token1 {
        id
        symbol
      }
    }
    origin
    amount0
    amount1
    amountUSD
  }
}
`

// V2SwapsResponse Uniswap V2 查询响应。
type V2SwapsResponse struct {
	Swaps []struct {
		ID          string `json:"id"`
		Transaction struct {
			ID          string `json:"id"`
			Timestamp   string `json:"timestamp"`
			BlockNumber string `json:"blockNumber"`
		} `json:"transaction"`
		Timestamp  string `json:"timestamp"`
		Pair       struct {
			ID     string `json:"id"`
			Token0 struct {
				ID     string `json:"id"`
				Symbol string `json:"symbol"`
			} `json:"token0"`
			Token1 struct {
				ID     string `json:"id"`
				Symbol string `json:"symbol"`
			} `json:"token1"`
		} `json:"pair"`
		Sender     string `json:"sender"`
		To         string `json:"to"`
		Amount0In  string `json:"amount0In"`
		Amount1In  string `json:"amount1In"`
		Amount0Out string `json:"amount0Out"`
		Amount1Out string `json:"amount1Out"`
		AmountUSD  string `json:"amountUSD"`
	} `json:"swaps"`
}

// V3SwapsResponse Uniswap V3 查询响应。
type V3SwapsResponse struct {
	Swaps []struct {
		ID          string `json:"id"`
		Transaction struct {
			ID          string `json:"id"`
			Timestamp   string `json:"timestamp"`
			BlockNumber string `json:"blockNumber"`
		} `json:"transaction"`
		Timestamp string `json:"timestamp"`
		Pool      struct {
			ID     string `json:"id"`
			Token0 struct {
				ID     string `json:"id"`
				Symbol string `json:"symbol"`
			} `json:"token0"`
			Token1 struct {
				ID     string `json:"id"`
				Symbol string `json:"symbol"`
			} `json:"token1"`
		} `json:"pool"`
		Origin    string `json:"origin"`
		Amount0   string `json:"amount0"`
		Amount1   string `json:"amount1"`
		AmountUSD string `json:"amountUSD"`
	} `json:"swaps"`
}

// FetchUniswapV2Swaps 查询 Uniswap V2 Swaps（分页）。
func (c *Client) FetchUniswapV2Swaps(ctx context.Context, startTime, endTime time.Time, minAmountUSD float64, batchSize int) ([]V2SwapsResponse, error) {
	var allResults []V2SwapsResponse
	skip := 0
	
	for {
		vars := map[string]interface{}{
			"first":         batchSize,
			"skip":          skip,
			"timestampGte":  int(startTime.Unix()),
			"timestampLt":   int(endTime.Unix()),
			"minAmountUSD":  fmt.Sprintf("%f", minAmountUSD),
		}

		var resp V2SwapsResponse
		if err := c.Query(ctx, UniswapV2SwapsQuery, vars, &resp); err != nil {
			return allResults, fmt.Errorf("query V2 swaps (skip=%d): %w", skip, err)
		}

		if len(resp.Swaps) == 0 {
			break
		}

		allResults = append(allResults, resp)
		
		if len(resp.Swaps) < batchSize {
			break
		}
		
		skip += batchSize
		
		// The Graph 限制 skip <= 5000
		if skip >= 5000 {
			return allResults, fmt.Errorf("reached skip limit (5000), use time-based pagination")
		}
	}
	
	return allResults, nil
}

// FetchUniswapV3Swaps 查询 Uniswap V3 Swaps（使用 ID 游标分页）。
func (c *Client) FetchUniswapV3Swaps(ctx context.Context, startTime, endTime time.Time, minAmountUSD float64, batchSize int) ([]V3SwapsResponse, error) {
	var allResults []V3SwapsResponse
	lastID := ""
	
	for {
		vars := map[string]interface{}{
			"first":         batchSize,
			"timestamp_gte": int(startTime.Unix()),
			"timestamp_lt":  int(endTime.Unix()),
			"minAmountUSD":  fmt.Sprintf("%f", minAmountUSD),
			"lastID":        lastID,
		}

		var resp V3SwapsResponse
		if err := c.Query(ctx, UniswapV3SwapsQuery, vars, &resp); err != nil {
			return allResults, fmt.Errorf("query V3 swaps (lastID=%s): %w", lastID, err)
		}

		if len(resp.Swaps) == 0 {
			break
		}

		allResults = append(allResults, resp)
		lastID = resp.Swaps[len(resp.Swaps)-1].ID
		
		if len(resp.Swaps) < batchSize {
			break
		}
	}
	
	return allResults, nil
}

// ParseV2Swap 解析 V2 Swap 为标准化交易数据。
func ParseV2Swap(swap interface{}) (*ParsedSwap, error) {
	// 类型断言
	s, ok := swap.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid swap data type")
	}
	
	parsed := &ParsedSwap{
		DEXName:    "uniswap",
		DEXVersion: "v2",
	}
	
	// Transaction
	if tx, ok := s["transaction"].(map[string]interface{}); ok {
		parsed.TxHash = getString(tx, "id")
		if ts := getString(tx, "timestamp"); ts != "" {
			timestamp, _ := strconv.ParseInt(ts, 10, 64)
			parsed.Timestamp = time.Unix(timestamp, 0)
		}
		if bn := getString(tx, "blockNumber"); bn != "" {
			parsed.BlockNumber, _ = strconv.ParseUint(bn, 10, 64)
		}
	}
	
	// Pair & Tokens
	if pair, ok := s["pair"].(map[string]interface{}); ok {
		parsed.PoolAddress = getString(pair, "id")
		if token0, ok := pair["token0"].(map[string]interface{}); ok {
			parsed.Token0 = getString(token0, "id")
			parsed.Token0Symbol = getString(token0, "symbol")
		}
		if token1, ok := pair["token1"].(map[string]interface{}); ok {
			parsed.Token1 = getString(token1, "id")
			parsed.Token1Symbol = getString(token1, "symbol")
		}
	}
	
	// Amounts
	amount0In := getString(s, "amount0In")
	amount1In := getString(s, "amount1In")
	amount0Out := getString(s, "amount0Out")
	amount1Out := getString(s, "amount1Out")
	
	// 判断方向：in > 0 为输入，out > 0 为输出
	if isPositive(amount0In) {
		parsed.TokenIn = parsed.Token0
		parsed.TokenInSymbol = parsed.Token0Symbol
		parsed.AmountIn = amount0In
	} else if isPositive(amount1In) {
		parsed.TokenIn = parsed.Token1
		parsed.TokenInSymbol = parsed.Token1Symbol
		parsed.AmountIn = amount1In
	}
	
	if isPositive(amount0Out) {
		parsed.TokenOut = parsed.Token0
		parsed.TokenOutSymbol = parsed.Token0Symbol
		parsed.AmountOut = amount0Out
	} else if isPositive(amount1Out) {
		parsed.TokenOut = parsed.Token1
		parsed.TokenOutSymbol = parsed.Token1Symbol
		parsed.AmountOut = amount1Out
	}
	
	parsed.AmountUSD = getString(s, "amountUSD")
	parsed.WalletAddress = getString(s, "to") // V2 使用 'to' 作为钱包地址
	
	return parsed, nil
}

// ParseV3Swap 解析 V3 Swap 为标准化交易数据。
func ParseV3Swap(swap interface{}) (*ParsedSwap, error) {
	s, ok := swap.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid swap data type")
	}
	
	parsed := &ParsedSwap{
		DEXName:    "uniswap",
		DEXVersion: "v3",
	}
	
	// Transaction
	if tx, ok := s["transaction"].(map[string]interface{}); ok {
		parsed.TxHash = getString(tx, "id")
		if ts := getString(tx, "timestamp"); ts != "" {
			timestamp, _ := strconv.ParseInt(ts, 10, 64)
			parsed.Timestamp = time.Unix(timestamp, 0)
		}
		if bn := getString(tx, "blockNumber"); bn != "" {
			parsed.BlockNumber, _ = strconv.ParseUint(bn, 10, 64)
		}
	}
	
	// Pool & Tokens
	if pool, ok := s["pool"].(map[string]interface{}); ok {
		parsed.PoolAddress = getString(pool, "id")
		if token0, ok := pool["token0"].(map[string]interface{}); ok {
			parsed.Token0 = getString(token0, "id")
			parsed.Token0Symbol = getString(token0, "symbol")
		}
		if token1, ok := pool["token1"].(map[string]interface{}); ok {
			parsed.Token1 = getString(token1, "id")
			parsed.Token1Symbol = getString(token1, "symbol")
		}
	}
	
	// Amounts (V3 使用有符号数，负数为输入，正数为输出)
	amount0 := getString(s, "amount0")
	amount1 := getString(s, "amount1")
	
	if isNegative(amount0) {
		parsed.TokenIn = parsed.Token0
		parsed.TokenInSymbol = parsed.Token0Symbol
		parsed.AmountIn = strings.TrimPrefix(amount0, "-")
	} else if isPositive(amount0) {
		parsed.TokenOut = parsed.Token0
		parsed.TokenOutSymbol = parsed.Token0Symbol
		parsed.AmountOut = amount0
	}
	
	if isNegative(amount1) {
		parsed.TokenIn = parsed.Token1
		parsed.TokenInSymbol = parsed.Token1Symbol
		parsed.AmountIn = strings.TrimPrefix(amount1, "-")
	} else if isPositive(amount1) {
		parsed.TokenOut = parsed.Token1
		parsed.TokenOutSymbol = parsed.Token1Symbol
		parsed.AmountOut = amount1
	}
	
	parsed.AmountUSD = getString(s, "amountUSD")
	parsed.WalletAddress = getString(s, "origin") // V3 使用 'origin' 作为钱包地址
	
	return parsed, nil
}

// ParsedSwap 标准化的 Swap 数据结构。
type ParsedSwap struct {
	TxHash          string
	BlockNumber     uint64
	Timestamp       time.Time
	DEXName         string
	DEXVersion      string
	PoolAddress     string
	WalletAddress   string
	Token0          string
	Token1          string
	Token0Symbol    string
	Token1Symbol    string
	TokenIn         string
	TokenOut        string
	TokenInSymbol   string
	TokenOutSymbol  string
	AmountIn        string
	AmountOut       string
	AmountUSD       string
}

// Helper functions

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func isPositive(s string) bool {
	if s == "" || s == "0" {
		return false
	}
	f, ok := new(big.Float).SetString(s)
	if !ok {
		return false
	}
	return f.Sign() > 0
}

func isNegative(s string) bool {
	if s == "" || s == "0" {
		return false
	}
	f, ok := new(big.Float).SetString(s)
	if !ok {
		return false
	}
	return f.Sign() < 0
}
