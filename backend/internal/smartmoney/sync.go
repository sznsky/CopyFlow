// Package smartmoney 提供聪明钱分析相关功能。
package smartmoney

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"copyflow/internal/model"
	"copyflow/internal/store"
	"copyflow/pkg/dune"

	"github.com/shopspring/decimal"
)

// Service 聪明钱服务。
type Service struct {
	store      *store.Store
	duneClient *dune.Client
	chainID    int
}

// NewService 创建聪明钱服务实例。
func NewService(st *store.Store, duneAPIKey string, chainID int) *Service {
	return &Service{
		store:      st,
		duneClient: dune.NewClient(duneAPIKey),
		chainID:    chainID,
	}
}

// Store 返回 store 实例。
func (s *Service) Store() *store.Store {
	return s.store
}

// SyncTradesFromDune 从 Dune Analytics 同步交易数据。
// queryID: Dune 查询 ID
// minAmountUSD: 最小交易金额（USD）
// daysBack: 查询历史天数
func (s *Service) SyncTradesFromDune(queryID int, minAmountUSD float64, daysBack int) error {
	log.Printf("[SmartMoney] Starting Dune sync for query %d, minAmount=%f USD, daysBack=%d", queryID, minAmountUSD, daysBack)
	
	syncLog := &model.DuneSyncLog{
		QueryID:   fmt.Sprintf("%d", queryID),
		ChainID:   s.chainID,
		SyncType:  "trades",
		StartTime: time.Now(),
		Status:    "running",
	}
	
	// 执行 Dune 查询
	params := map[string]interface{}{
		"min_amount_usd": minAmountUSD,
		"days_back":      daysBack,
	}
	
	result, err := s.duneClient.ExecuteAndWait(queryID, params, 5*time.Minute)
	if err != nil {
		syncLog.Status = "failed"
		syncLog.EndTime = time.Now()
		errMsg := err.Error()
		syncLog.ErrorMessage = &errMsg
		s.store.DB().Create(syncLog)
		return fmt.Errorf("execute dune query: %w", err)
	}
	
	syncLog.RecordsFetched = result.Metadata.RowCount
	
	if result.Metadata.RowCount > 0 && len(result.Rows) == 0 {
		log.Printf("[SmartMoney] Dune returned row_count=%d but parsed 0 rows (check API response structure)",
			result.Metadata.RowCount)
	}
	
	// 解析并插入交易数据
	inserted := 0
	updated := 0
	parseFailed := 0
	insertFailed := 0
	parseErrors := make(map[string]int)
	
	for _, row := range result.Rows {
		trade, err := s.parseTradeRow(row)
		if err != nil {
			parseFailed++
			parseErrors[err.Error()]++
			continue
		}
		
		// 检查是否已存在
		var existing model.WalletTrade
		err = s.store.DB().Where("tx_hash = ? AND wallet_address = ? AND token_out = ?",
			trade.TxHash, trade.WalletAddress, trade.TokenOut).First(&existing).Error
		
		if err != nil {
			// 不存在，插入新记录
			if err := s.store.DB().Create(trade).Error; err != nil {
				insertFailed++
				if insertFailed <= 3 {
					log.Printf("[SmartMoney] Insert failed (tx=%s): %v", trade.TxHash, err)
				}
				continue
			}
			inserted++
		} else {
			// 已存在，更新（保留已计算的盈亏字段）
			if err := s.store.DB().Model(&existing).
				Omit("PnlUSD", "PnlPercent", "HoldingDurationHours").
				Updates(trade).Error; err != nil {
				insertFailed++
				if insertFailed <= 3 {
					log.Printf("[SmartMoney] Update failed (tx=%s): %v", trade.TxHash, err)
				}
				continue
			}
			updated++
		}
	}
	
	if parseFailed > 0 {
		log.Printf("[SmartMoney] %d rows failed to parse:", parseFailed)
		for msg, count := range parseErrors {
			log.Printf("[SmartMoney]   [%d] %s", count, msg)
		}
	}
	if insertFailed > 0 {
		log.Printf("[SmartMoney] %d rows failed to insert/update", insertFailed)
	}
	
	syncLog.RecordsInserted = inserted
	syncLog.RecordsUpdated = updated
	syncLog.Status = "success"
	syncLog.EndTime = time.Now()
	
	if err := s.store.DB().Create(syncLog).Error; err != nil {
		log.Printf("[SmartMoney] Failed to save sync log: %v", err)
	}
	
	log.Printf("[SmartMoney] Sync completed: %d fetched, %d inserted, %d updated",
		syncLog.RecordsFetched, inserted, updated)
	
	return nil
}

// parseTradeRow 解析 Dune 返回的交易行数据。
func (s *Service) parseTradeRow(row map[string]interface{}) (*model.WalletTrade, error) {
	trade := &model.WalletTrade{
		ChainID: s.chainID,
	}
	
	walletAddr, err := duneStringField(row, "wallet_address")
	if err != nil {
		return nil, err
	}
	trade.WalletAddress = strings.ToLower(walletAddr)
	
	txHash, err := duneStringField(row, "tx_hash")
	if err != nil {
		return nil, err
	}
	trade.TxHash = txHash
	
	blockNum, err := duneUint64Field(row, "block_number")
	if err != nil {
		return nil, err
	}
	trade.BlockNumber = blockNum
	
	blockTimeStr, err := duneStringField(row, "block_time")
	if err != nil {
		return nil, err
	}
	parsedTime, err := parseDuneBlockTime(blockTimeStr)
	if err != nil {
		return nil, fmt.Errorf("parse block_time: %w", err)
	}
	trade.BlockTime = parsedTime
	
	dexName, err := duneStringField(row, "dex_name")
	if err != nil {
		return nil, err
	}
	trade.DEXName, trade.DEXVersion = parseDEXNameVersion(dexName)
	if ver, err := duneOptionalStringField(row, "dex_version"); err == nil && ver != "" {
		trade.DEXVersion = ver
	}
	
	poolAddr, err := duneStringField(row, "pool_address")
	if err != nil {
		return nil, err
	}
	trade.PoolAddress = strings.ToLower(poolAddr)
	
	tokenIn, err := duneStringField(row, "token_in")
	if err != nil {
		return nil, err
	}
	trade.TokenIn = strings.ToLower(tokenIn)
	
	tokenOut, err := duneStringField(row, "token_out")
	if err != nil {
		return nil, err
	}
	trade.TokenOut = strings.ToLower(tokenOut)
	
	amountInStr, err := duneStringField(row, "amount_in")
	if err != nil {
		return nil, err
	}
	amountIn, err := decimal.NewFromString(amountInStr)
	if err != nil {
		return nil, fmt.Errorf("parse amount_in: %w", err)
	}
	trade.AmountIn = amountIn
	
	amountOutStr, err := duneStringField(row, "amount_out")
	if err != nil {
		return nil, err
	}
	amountOut, err := decimal.NewFromString(amountOutStr)
	if err != nil {
		return nil, fmt.Errorf("parse amount_out: %w", err)
	}
	trade.AmountOut = amountOut
	
	amountUSD, err := duneFloatField(row, "amount_usd")
	if err != nil {
		return nil, err
	}
	trade.AmountUSD = decimal.NewFromFloat(amountUSD)
	
	isBuy, ok := row["is_buy"].(bool)
	if !ok {
		return nil, fmt.Errorf("missing is_buy")
	}
	trade.IsBuy = isBuy
	
	// 可选字段
	if tokenInSymbol, err := duneOptionalStringField(row, "token_in_symbol"); err == nil {
		trade.TokenInSymbol = &tokenInSymbol
	}
	
	if tokenOutSymbol, err := duneOptionalStringField(row, "token_out_symbol"); err == nil {
		trade.TokenOutSymbol = &tokenOutSymbol
	}
	
	return trade, nil
}

// parseDEXNameVersion 从 dex_name 拆分协议名与版本（如 uniswap_v2 -> uniswap, v2）。
func parseDEXNameVersion(dexName string) (name, version string) {
	dexName = strings.TrimSpace(strings.ToLower(dexName))
	if idx := strings.LastIndex(dexName, "_"); idx > 0 {
		suffix := dexName[idx+1:]
		if strings.HasPrefix(suffix, "v") && len(suffix) <= 3 {
			return dexName[:idx], suffix
		}
	}
	return dexName, ""
}

func duneStringField(row map[string]interface{}, key string) (string, error) {
	v, ok := row[key]
	if !ok || v == nil {
		return "", fmt.Errorf("missing %s", key)
	}
	switch s := v.(type) {
	case string:
		return s, nil
	default:
		return fmt.Sprint(v), nil
	}
}

func duneOptionalStringField(row map[string]interface{}, key string) (string, error) {
	v, ok := row[key]
	if !ok || v == nil {
		return "", fmt.Errorf("missing %s", key)
	}
	switch s := v.(type) {
	case string:
		return s, nil
	default:
		return fmt.Sprint(v), nil
	}
}

func duneUint64Field(row map[string]interface{}, key string) (uint64, error) {
	v, ok := row[key]
	if !ok || v == nil {
		return 0, fmt.Errorf("missing %s", key)
	}
	switch n := v.(type) {
	case float64:
		return uint64(n), nil
	case int64:
		return uint64(n), nil
	case int:
		return uint64(n), nil
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", key, err)
		}
		return uint64(i), nil
	default:
		i, err := strconv.ParseUint(fmt.Sprint(v), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", key, err)
		}
		return i, nil
	}
}

func duneFloatField(row map[string]interface{}, key string) (float64, error) {
	v, ok := row[key]
	if !ok || v == nil {
		return 0, fmt.Errorf("missing %s", key)
	}
	switch n := v.(type) {
	case float64:
		return n, nil
	case int64:
		return float64(n), nil
	case int:
		return float64(n), nil
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", key, err)
		}
		return f, nil
	default:
		f, err := strconv.ParseFloat(fmt.Sprint(v), 64)
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", key, err)
		}
		return f, nil
	}
}

func duneOptionalFloatField(row map[string]interface{}, key string) (float64, error) {
	return duneFloatField(row, key)
}

func parseDuneBlockTime(value string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05.000 UTC",
		"2006-01-02 15:04:05 UTC",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported format: %s", value)
}
