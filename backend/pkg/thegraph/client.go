// Package thegraph 提供 The Graph GraphQL 客户端。
package thegraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client The Graph GraphQL 客户端。
type Client struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

// NewClient 创建 The Graph 客户端。
func NewClient(endpoint, apiKey string) *Client {
	return &Client{
		endpoint: endpoint,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GraphQLRequest GraphQL 请求体。
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse GraphQL 响应体。
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// Query 执行 GraphQL 查询。
func (c *Client) Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var gqlResp GraphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	if err := json.Unmarshal(gqlResp.Data, result); err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}

	return nil
}

// Swap The Graph swap 事件结构（通用字段）。
type Swap struct {
	ID          string `json:"id"`
	Transaction struct {
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
	} `json:"transaction"`
	Timestamp   string `json:"timestamp"`
	Sender      string `json:"sender,omitempty"`
	Origin      string `json:"origin,omitempty"`
	To          string `json:"to,omitempty"`
	Amount0     string `json:"amount0"`
	Amount1     string `json:"amount1"`
	AmountUSD   string `json:"amountUSD"`
	Token0      struct {
		ID     string `json:"id"`
		Symbol string `json:"symbol"`
	} `json:"token0"`
	Token1 struct {
		ID     string `json:"id"`
		Symbol string `json:"symbol"`
	} `json:"token1"`
}

// V3Swap Uniswap V3 Swap 结构（继承 Swap，添加特有字段）。
type V3Swap struct {
	Swap
	Pool struct {
		ID string `json:"id"`
	} `json:"pool"`
	Amount0 string `json:"amount0"`
	Amount1 string `json:"amount1"`
}

// V2Swap Uniswap V2 Swap 结构（继承 Swap，添加特有字段）。
type V2Swap struct {
	Swap
	Pair struct {
		ID string `json:"id"`
	} `json:"pair"`
}
