// Package smartmoney 提供聪明钱分析相关功能。
package smartmoney

import (
	"context"
	"fmt"
	"strings"
	"time"

	"copyflow/internal/model"
	"copyflow/internal/store"
	"copyflow/pkg/logger"
	"copyflow/pkg/thegraph"

	"github.com/shopspring/decimal"
)

// Service 聪明钱服务。
type Service struct {
	store          *store.Store
	v2Client       *thegraph.Client
	v3Client       *thegraph.Client
	chainID        int
	batchSize      int
	pairs          []string // 监听的交易对地址列表（空表示不过滤）
	evalDays       int      // 评分评估窗口天数
	seedWallets    []string // 手动维护的种子钱包地址（非空时优先使用）
	minWalletScore float64  // 进入 Top 榜单的最低分
}

// NewService 创建聪明钱服务实例。
func NewService(st *store.Store, v2Endpoint, v3Endpoint, apiKey string, chainID, batchSize int, pairs []string, evalDays int, seedWallets []string, minWalletScore float64) *Service {
	return &Service{
		store:          st,
		v2Client:       thegraph.NewClient(v2Endpoint, apiKey),
		v3Client:       thegraph.NewClient(v3Endpoint, apiKey),
		chainID:        chainID,
		batchSize:      batchSize,
		pairs:          pairs,
		evalDays:       evalDays,
		seedWallets:    seedWallets,
		minWalletScore: minWalletScore,
	}
}

// IsSeedMode 是否启用种子钱包模式。
func (s *Service) IsSeedMode() bool {
	return len(s.seedWallets) > 0
}

// Store 返回 store 实例。
func (s *Service) Store() *store.Store {
	return s.store
}

// SyncTradesFromTheGraph 从 The Graph 同步交易数据。
func (s *Service) SyncTradesFromTheGraph(startTime, endTime time.Time, minAmountUSD float64, syncType string) error {
	logger.Info("Starting The Graph sync",
		"sync_type", syncType,
		"start_date", startTime.Format("2006-01-02"),
		"end_date", endTime.Format("2006-01-02"),
		"min_amount_usd", minAmountUSD,
	)

	start := time.Now()
	ctx := context.Background()
	totalInserted := 0
	totalUpdated := 0
	totalSkipped := 0

	if len(s.pairs) > 0 {
		logger.Info("Pair filter active", "pairs_count", len(s.pairs))
	}

	// 同步 Uniswap V2
	logger.Info("Syncing Uniswap V2...")
	v2Inserted, v2Updated, v2Skipped, err := s.syncV2Swaps(ctx, startTime, endTime, minAmountUSD)
	if err != nil {
		logger.Error("V2 sync error", "error", err)
	} else {
		totalInserted += v2Inserted
		totalUpdated += v2Updated
		totalSkipped += v2Skipped
		logger.Info("V2 completed",
			"inserted", v2Inserted,
			"updated", v2Updated,
			"skipped", v2Skipped,
		)
	}

	// 同步 Uniswap V3
	logger.Info("Syncing Uniswap V3...")
	v3Inserted, v3Updated, v3Skipped, err := s.syncV3Swaps(ctx, startTime, endTime, minAmountUSD)
	if err != nil {
		logger.Error("V3 sync error", "error", err)
	} else {
		totalInserted += v3Inserted
		totalUpdated += v3Updated
		totalSkipped += v3Skipped
		logger.Info("V3 completed",
			"inserted", v3Inserted,
			"updated", v3Updated,
			"skipped", v3Skipped,
		)
	}

	// 记录同步日志
	syncLog := &model.SyncLog{
		Source:          "thegraph",
		SyncType:        syncType,
		ChainID:         s.chainID,
		StartTime:       startTime,
		EndTime:         endTime,
		RecordsInserted: totalInserted,
		RecordsUpdated:  totalUpdated,
		RecordsSkipped:  totalSkipped,
		Status:          "success",
		CompletedAt:     time.Now(),
	}
	if err := s.store.DB().Create(syncLog).Error; err != nil {
		logger.Error("Failed to save sync log", "error", err)
	}

	logger.Info("Sync completed",
		"total_inserted", totalInserted,
		"total_updated", totalUpdated,
		"total_skipped", totalSkipped,
		"duration_sec", time.Since(start).Seconds(),
	)

	return nil
}

// syncV2Swaps 同步 Uniswap V2 交易。
func (s *Service) syncV2Swaps(ctx context.Context, startTime, endTime time.Time, minAmountUSD float64) (int, int, int, error) {
	results, err := s.v2Client.FetchUniswapV2Swaps(ctx, startTime, endTime, minAmountUSD, s.batchSize, s.pairs, s.seedWallets)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("fetch V2 swaps: %w", err)
	}

	inserted := 0
	updated := 0
	skipped := 0

	for _, batch := range results {
		for _, swap := range batch.Swaps {
			// 将 swap 转为 map[string]interface{} 以便复用 ParseV2Swap
			swapMap := map[string]interface{}{
				"transaction": map[string]interface{}{
					"id":          swap.Transaction.ID,
					"timestamp":   swap.Transaction.Timestamp,
					"blockNumber": swap.Transaction.BlockNumber,
				},
				"pair": map[string]interface{}{
					"id": swap.Pair.ID,
					"token0": map[string]interface{}{
						"id":     swap.Pair.Token0.ID,
						"symbol": swap.Pair.Token0.Symbol,
					},
					"token1": map[string]interface{}{
						"id":     swap.Pair.Token1.ID,
						"symbol": swap.Pair.Token1.Symbol,
					},
				},
				"to":         swap.To,
				"amount0In":  swap.Amount0In,
				"amount1In":  swap.Amount1In,
				"amount0Out": swap.Amount0Out,
				"amount1Out": swap.Amount1Out,
				"amountUSD":  swap.AmountUSD,
			}

			parsed, err := thegraph.ParseV2Swap(swapMap)
			if err != nil {
				skipped++
				continue
			}

			if s.shouldSkipSwap(parsed) {
				skipped++
				continue
			}

			trade, err := s.parsedSwapToTrade(parsed)
			if err != nil {
				skipped++
				continue
			}

			if err := s.upsertTrade(trade); err != nil {
				logger.Error("Upsert V2 trade failed",
					"error", err,
					"tx_hash", trade.TxHash,
					"wallet", trade.WalletAddress,
				)
				skipped++
				continue
			}

			// 简单判断是插入还是更新（实际由 upsert 处理）
			inserted++
		}
	}

	return inserted, updated, skipped, nil
}

// syncV3Swaps 同步 Uniswap V3 交易。
func (s *Service) syncV3Swaps(ctx context.Context, startTime, endTime time.Time, minAmountUSD float64) (int, int, int, error) {
	results, err := s.v3Client.FetchUniswapV3Swaps(ctx, startTime, endTime, minAmountUSD, s.batchSize, s.pairs, s.seedWallets)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("fetch V3 swaps: %w", err)
	}

	inserted := 0
	updated := 0
	skipped := 0

	for _, batch := range results {
		for _, swap := range batch.Swaps {
			swapMap := map[string]interface{}{
				"transaction": map[string]interface{}{
					"id":          swap.Transaction.ID,
					"timestamp":   swap.Transaction.Timestamp,
					"blockNumber": swap.Transaction.BlockNumber,
				},
				"pool": map[string]interface{}{
					"id": swap.Pool.ID,
					"token0": map[string]interface{}{
						"id":     swap.Pool.Token0.ID,
						"symbol": swap.Pool.Token0.Symbol,
					},
					"token1": map[string]interface{}{
						"id":     swap.Pool.Token1.ID,
						"symbol": swap.Pool.Token1.Symbol,
					},
				},
				"origin":    swap.Origin,
				"amount0":   swap.Amount0,
				"amount1":   swap.Amount1,
				"amountUSD": swap.AmountUSD,
			}

			parsed, err := thegraph.ParseV3Swap(swapMap)
			if err != nil {
				skipped++
				continue
			}

			if s.shouldSkipSwap(parsed) {
				skipped++
				continue
			}

			trade, err := s.parsedSwapToTrade(parsed)
			if err != nil {
				skipped++
				continue
			}

			if err := s.upsertTrade(trade); err != nil {
				logger.Error("Upsert V3 trade failed",
					"error", err,
					"tx_hash", trade.TxHash,
					"wallet", trade.WalletAddress,
				)
				skipped++
				continue
			}

			inserted++
		}
	}

	return inserted, updated, skipped, nil
}

// shouldSkipSwap 判断是否应该跳过该交易。
func (s *Service) shouldSkipSwap(parsed *thegraph.ParsedSwap) bool {
	// 1. 过滤稳定币互换
	if thegraph.IsStablecoinSwap(parsed.TokenIn, parsed.TokenOut) {
		return true
	}

	// 2. 必须能判断买入/卖出方向
	_, ok := thegraph.ClassifyDirection(parsed.TokenIn, parsed.TokenOut)
	if !ok {
		return true
	}

	// 3. 钱包地址不能为空
	if parsed.WalletAddress == "" || parsed.WalletAddress == "0x0000000000000000000000000000000000000000" {
		return true
	}

	return false
}

// parsedSwapToTrade 将解析后的 Swap 转换为 WalletTrade 模型。
func (s *Service) parsedSwapToTrade(parsed *thegraph.ParsedSwap) (*model.WalletTrade, error) {
	trade := &model.WalletTrade{
		ChainID:       s.chainID,
		TxHash:        strings.ToLower(parsed.TxHash),
		BlockNumber:   parsed.BlockNumber,
		BlockTime:     parsed.Timestamp,
		DEXName:       parsed.DEXName,
		DEXVersion:    parsed.DEXVersion,
		PoolAddress:   strings.ToLower(parsed.PoolAddress),
		WalletAddress: strings.ToLower(parsed.WalletAddress),
		TokenIn:       strings.ToLower(parsed.TokenIn),
		TokenOut:      strings.ToLower(parsed.TokenOut),
	}

	// Token symbols
	if parsed.TokenInSymbol != "" {
		trade.TokenInSymbol = &parsed.TokenInSymbol
	}
	if parsed.TokenOutSymbol != "" {
		trade.TokenOutSymbol = &parsed.TokenOutSymbol
	}

	// 金额转换
	amountIn, err := decimal.NewFromString(parsed.AmountIn)
	if err != nil {
		return nil, fmt.Errorf("parse amount_in: %w", err)
	}
	trade.AmountIn = amountIn

	amountOut, err := decimal.NewFromString(parsed.AmountOut)
	if err != nil {
		return nil, fmt.Errorf("parse amount_out: %w", err)
	}
	trade.AmountOut = amountOut

	amountUSD, err := decimal.NewFromString(parsed.AmountUSD)
	if err != nil {
		return nil, fmt.Errorf("parse amount_usd: %w", err)
	}
	trade.AmountUSD = amountUSD

	// 判断买入/卖出
	isBuy, _ := thegraph.ClassifyDirection(parsed.TokenIn, parsed.TokenOut)
	trade.IsBuy = isBuy

	return trade, nil
}

// upsertTrade 插入或更新交易（保留已计算的 PNL）。
func (s *Service) upsertTrade(trade *model.WalletTrade) error {
	var existing model.WalletTrade
	err := s.store.DB().Where("tx_hash = ? AND wallet_address = ? AND token_out = ?",
		trade.TxHash, trade.WalletAddress, trade.TokenOut).First(&existing).Error

	if err != nil {
		// 不存在，插入新记录
		return s.store.DB().Create(trade).Error
	}

	// 已存在，更新（保留已计算的盈亏字段）
	return s.store.DB().Model(&existing).
		Omit("PnlUSD", "PnlPercent", "HoldingDurationHours").
		Updates(trade).Error
}

// SyncSeedWallets 种子钱包模式：拉取 seed_wallets 中所有地址的历史交易。
// 直接复用 SyncTradesFromTheGraph，由 syncV2/V3Swaps 内部自动选择钱包过滤查询。
func (s *Service) SyncSeedWallets(startTime, endTime time.Time, minAmountUSD float64, syncType string) error {
	if !s.IsSeedMode() {
		logger.Info("SyncSeedWallets skipped: no seed wallets configured")
		return nil
	}
	logger.Info("Starting seed wallet sync",
		"wallet_count", len(s.seedWallets),
		"sync_type", syncType,
		"start_date", startTime.Format("2006-01-02"),
		"end_date", endTime.Format("2006-01-02"),
	)
	return s.SyncTradesFromTheGraph(startTime, endTime, minAmountUSD, syncType)
}

// CleanupOldTrades 清理超过保留期的交易数据。
func (s *Service) CleanupOldTrades(retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	
	logger.Info("Cleaning up old trades",
		"cutoff_date", cutoffTime.Format("2006-01-02"),
		"retention_days", retentionDays,
	)
	
	result := s.store.DB().Where("chain_id = ? AND block_time < ?", s.chainID, cutoffTime).
		Delete(&model.WalletTrade{})
	
	if result.Error != nil {
		return fmt.Errorf("cleanup old trades: %w", result.Error)
	}
	
	logger.Info("Cleanup completed", "deleted_rows", result.RowsAffected)
	return nil
}
