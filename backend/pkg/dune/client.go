// Package dune 提供与 Dune Analytics API 的集成。
package dune

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL = "https://api.dune.com/api/v1"
)

// Client Dune Analytics API 客户端。
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient 创建 Dune API 客户端。
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteQueryRequest 执行查询请求体。
type ExecuteQueryRequest struct {
	QueryParameters map[string]interface{} `json:"query_parameters,omitempty"`
}

// ExecuteQueryResponse 执行查询响应。
type ExecuteQueryResponse struct {
	ExecutionID string `json:"execution_id"`
	State       string `json:"state"`
}

// GetExecutionStatusResponse 查询执行状态响应。
type GetExecutionStatusResponse struct {
	ExecutionID      string    `json:"execution_id"`
	QueryID          int       `json:"query_id"`
	State            string    `json:"state"`
	SubmittedAt      time.Time `json:"submitted_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	ExecutionStarted *time.Time `json:"execution_started"`
	ExecutionEnded   *time.Time `json:"execution_ended"`
	Result           *QueryResult `json:"result"`
}

// QueryResult 查询结果。
type QueryResult struct {
	Rows     []map[string]interface{} `json:"rows"`
	Metadata QueryMetadata            `json:"metadata"`
}

// QueryMetadata 查询元数据。
type QueryMetadata struct {
	ColumnNames []string `json:"column_names"`
	ColumnTypes []string `json:"column_types"`
	RowCount    int      `json:"row_count"`
	ResultSetBytes int   `json:"result_set_bytes"`
	TotalRowCount int    `json:"total_row_count"`
}

// ExecuteQuery 执行 Dune 查询。
func (c *Client) ExecuteQuery(queryID int, params map[string]interface{}) (*ExecuteQueryResponse, error) {
	url := fmt.Sprintf("%s/query/%d/execute", baseURL, queryID)
	
	reqBody := ExecuteQueryRequest{
		QueryParameters: params,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dune-API-Key", c.apiKey)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	var result ExecuteQueryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	
	return &result, nil
}

// GetExecutionStatus 获取查询执行状态。
func (c *Client) GetExecutionStatus(executionID string) (*GetExecutionStatusResponse, error) {
	url := fmt.Sprintf("%s/execution/%s/status", baseURL, executionID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	
	req.Header.Set("X-Dune-API-Key", c.apiKey)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	var result GetExecutionStatusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	
	return &result, nil
}

// GetExecutionResults 获取查询执行结果（轮询直到完成）。
func (c *Client) GetExecutionResults(executionID string, maxWaitTime time.Duration) (*QueryResult, error) {
	startTime := time.Now()
	pollInterval := 2 * time.Second
	
	for {
		if time.Since(startTime) > maxWaitTime {
			return nil, fmt.Errorf("timeout waiting for query execution")
		}
		
		status, err := c.GetExecutionStatus(executionID)
		if err != nil {
			return nil, err
		}
		
		switch status.State {
		case "QUERY_STATE_COMPLETED":
			if status.Result == nil {
				return nil, fmt.Errorf("query completed but no result")
			}
			return status.Result, nil
		case "QUERY_STATE_FAILED":
			return nil, fmt.Errorf("query execution failed")
		case "QUERY_STATE_CANCELLED":
			return nil, fmt.Errorf("query execution cancelled")
		case "QUERY_STATE_EXECUTING", "QUERY_STATE_PENDING":
			time.Sleep(pollInterval)
			continue
		default:
			return nil, fmt.Errorf("unknown query state: %s", status.State)
		}
	}
}

// ExecuteAndWait 执行查询并等待结果。
func (c *Client) ExecuteAndWait(queryID int, params map[string]interface{}, maxWaitTime time.Duration) (*QueryResult, error) {
	execResp, err := c.ExecuteQuery(queryID, params)
	if err != nil {
		return nil, fmt.Errorf("execute query: %w", err)
	}
	
	return c.GetExecutionResults(execResp.ExecutionID, maxWaitTime)
}
