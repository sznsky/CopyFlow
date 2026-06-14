// Package smartmoney 提供聪明钱分析相关功能。
package smartmoney

import (
	"fmt"
	"log"
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
func (s *Service) SyncTradesFromDune(queryID int, minAmountUSD float64) error {
	log.Printf("[SmartMoney] Starting Dune sync for query %d, minAmount=%f USD", queryID, minAmountUSD)
	
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
		"days_back":      180, // 6个月
	}
	
	result, err := s.duneClient.ExecuteAndWait(queryID, params, 5*time.Minute)
	if err != nil {
		syncLog.Status = "failed"
		syncLog.EndTime = time.Now()
		errMsg := err.Error()
		syncLog.ErrorMessage = &errMsg
		s.store.DB.Create(syncLog)
		return fmt.Errorf("execute dune query: %w", err)
	}
	
	syncLog.RecordsFetched = result.Metadata.RowCount
	
	// 解析并插入交易数据
	inserted := 0
	updated := 0
	
	for _, row := range result.Rows {
		trade, err := s.parseTradeRow(row)
		if err != nil {
			log.Printf("[SmartMoney] Failed to parse row: %v", err)
			continue
		}
		
		// 检查是否已存在
		var existing model.WalletTrade
		err = s.store.DB.Where("tx_hash = ? AND wallet_address = ? AND token_out = ?",
			trade.TxHash, trade.WalletAddress, trade.TokenOut).First(&existing).Error
		
		if err != nil {
			// 不存在，插入新记录
			if err := s.store.DB.Create(trade).Error; err != nil {
				log.Printf("[SmartMoney] Failed to insert trade: %v", err)
				continue
			}
			inserted++
		} else {
			// 已存在，更新
			if err := s.store.DB.Model(&existing).Updates(trade).Error; err != nil {
				log.Printf("[SmartMoney] Failed to update trade: %v", err)
				continue
			}
			updated++
		}
	}
	
	syncLog.RecordsInserted = inserted
	syncLog.RecordsUpdated = updated
	syncLog.Status = "success"
	syncLog.EndTime = time.Now()
	
	if err := s.store.DB.Create(syncLog).Error; err != nil {
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
	
	// 必填字段
	walletAddr, ok := row["wallet_address"].(string)
	if !ok {
		return nil, fmt.Errorf("missing wallet_address")
	}
	trade.WalletAddress = walletAddr
	
	txHash, ok := row["tx_hash"].(string)
	if !ok {
		return nil, fmt.Errorf("missing tx_hash")
	}
	trade.TxHash = txHash
	
	blockNum, ok := row["block_number"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing block_number")
	}
	trade.BlockNumber = uint64(blockNum)
	
	blockTime, ok := row["block_time"].(string)
	if !ok {
		return nil, fmt.Errorf("missing block_time")
	}
	parsedTime, err := time.Parse(time.RFC3339, blockTime)
	if err != nil {
		return nil, fmt.Errorf("parse block_time: %w", err)
	}
	trade.BlockTime = parsedTime
	
	dexName, ok := row["dex_name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing dex_name")
	}
	trade.DEXName = dexName
	
	poolAddr, ok := row["pool_address"].(string)
	if !ok {
		return nil, fmt.Errorf("missing pool_address")
	}
	trade.PoolAddress = poolAddr
	
	tokenIn, ok := row["token_in"].(string)
	if !ok {
		return nil, fmt.Errorf("missing token_in")
	}
	trade.TokenIn = tokenIn
	
	tokenOut, ok := row["token_out"].(string)
	if !ok {
		return nil, fmt.Errorf("missing token_out")
	}
	trade.TokenOut = tokenOut
	
	amountInStr, ok := row["amount_in"].(string)
	if !ok {
		return nil, fmt.Errorf("missing amount_in")
	}
	amountIn, err := decimal.NewFromString(amountInStr)
	if err != nil {
		return nil, fmt.Errorf("parse amount_in: %w", err)
	}
	trade.AmountIn = amountIn
	
	amountOutStr, ok := row["amount_out"].(string)
	if !ok {
		return nil, fmt.Errorf("missing amount_out")
	}
	amountOut, err := decimal.NewFromString(amountOutStr)
	if err != nil {
		return nil, fmt.Errorf("parse amount_out: %w", err)
	}
	trade.AmountOut = amountOut
	
	amountUSDFloat, ok := row["amount_usd"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing amount_usd")
	}
	trade.AmountUSD = decimal.NewFromFloat(amountUSDFloat)
	
	isBuy, ok := row["is_buy"].(bool)
	if !ok {
		return nil, fmt.Errorf("missing is_buy")
	}
	trade.IsBuy = isBuy
	
	// 可选字段
	if tokenInSymbol, ok := row["token_in_symbol"].(string); ok {
		trade.TokenInSymbol = &tokenInSymbol
	}
	
	if tokenOutSymbol, ok := row["token_out_symbol"].(string); ok {
		trade.TokenOutSymbol = &tokenOutSymbol
	}
	
	if pnlUSDFloat, ok := row["pnl_usd"].(float64); ok {
		pnlUSD := decimal.NewFromFloat(pnlUSDFloat)
		trade.PNLUSD = &pnlUSD
	}
	
	if pnlPercentFloat, ok := row["pnl_percent"].(float64); ok {
		pnlPercent := decimal.NewFromFloat(pnlPercentFloat)
		trade.PNLPercent = &pnlPercent
	}
	
	if holdingHours, ok := row["holding_duration_hours"].(float64); ok {
		hours := int(holdingHours)
		trade.HoldingDurationHours = &hours
	}
	
	return trade, nil
}
