// SmartMoney Worker: 定期同步 The Graph 数据、计算评分、聚合信号。
package main

import (
	"time"

	"copyflow/internal/config"
	"copyflow/internal/smartmoney"
	"copyflow/internal/store"
	"copyflow/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// 初始化 logger
	if err := logger.Init(cfg.Server.Mode); err != nil {
		panic(err)
	}
	defer logger.Sync()

	if !cfg.SmartMoney.Enabled {
		logger.Info("SmartMoney Worker disabled in config, exiting")
		return
	}

	if !cfg.TheGraph.Enabled {
		logger.Info("The Graph disabled, exiting")
		return
	}

	st, err := store.New(cfg.Database.DSN)
	if err != nil {
		logger.Fatal("Failed to connect database", "error", err)
	}
	if err := st.AutoMigrate(); err != nil {
		logger.Fatal("Failed to migrate database", "error", err)
	}

	service := smartmoney.NewService(
		st,
		cfg.TheGraph.UniswapV2Endpoint,
		cfg.TheGraph.UniswapV3Endpoint,
		cfg.TheGraph.APIKey,
		cfg.SmartMoney.ChainID,
		cfg.SmartMoney.BatchSize,
	)

	logger.Info("SmartMoney Worker started",
		"chain_id", cfg.SmartMoney.ChainID,
		"sync_interval_hours", cfg.SmartMoney.SyncIntervalHours,
		"min_amount_usd", cfg.SmartMoney.MinAmountUSD,
		"retention_days", cfg.SmartMoney.RetentionDays,
	)

	// 首次执行：同时启动历史数据拉取和增量同步
	go runHistoricalSync(service, cfg)
	runIncrementalCycle(service, cfg)

	// 定期执行增量同步
	ticker := time.NewTicker(time.Duration(cfg.SmartMoney.SyncIntervalHours) * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		runIncrementalCycle(service, cfg)
	}
}

// runHistoricalSync 拉取半年历史数据（后台任务，只运行一次）。
func runHistoricalSync(service *smartmoney.Service, cfg *config.Config) {
	logger.Info("========== Starting historical sync (6 months) ==========")

	endTime := time.Now().AddDate(0, 0, -1) // 昨天结束
	startTime := endTime.AddDate(0, 0, -180) // 往前 180 天

	// 分批拉取，每次 30 天
	batchDays := 30
	currentStart := startTime

	for currentStart.Before(endTime) {
		currentEnd := currentStart.AddDate(0, 0, batchDays)
		if currentEnd.After(endTime) {
			currentEnd = endTime
		}

		logger.Info("Historical batch",
			"start_date", currentStart.Format("2006-01-02"),
			"end_date", currentEnd.Format("2006-01-02"),
		)

		if err := service.SyncTradesFromTheGraph(currentStart, currentEnd, cfg.SmartMoney.MinAmountUSD, "historical"); err != nil {
			logger.Error("Historical sync failed", "error", err)
		}

		currentStart = currentEnd
		time.Sleep(5 * time.Second) // 避免请求过快
	}

	logger.Info("========== Historical sync completed ==========")

	// 历史数据同步完成后，执行一次完整的 PNL 计算和评分
	logger.Info("Running post-historical analysis...")
	runAnalysis(service, cfg)
}

// runIncrementalCycle 执行增量同步（拉取昨天的数据）。
func runIncrementalCycle(service *smartmoney.Service, cfg *config.Config) {
	logger.Info("========== Starting incremental cycle ==========")

	// 昨天 00:00:00 到 23:59:59
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	startTime := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
	endTime := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 0, time.UTC)

	// 1. 同步昨天的交易数据
	logger.Info("Step 1: Syncing yesterday's trades",
		"date", yesterday.Format("2006-01-02"),
	)
	if err := service.SyncTradesFromTheGraph(startTime, endTime, cfg.SmartMoney.MinAmountUSD, "incremental"); err != nil {
		logger.Error("Failed to sync trades", "error", err)
	} else {
		logger.Info("Step 1: ✓ Trades synced")
	}

	time.Sleep(2 * time.Second)

	// 2. 运行分析
	runAnalysis(service, cfg)

	// 3. 清理旧数据
	logger.Info("Step 4: Cleaning up old data")
	if err := service.CleanupOldTrades(cfg.SmartMoney.RetentionDays); err != nil {
		logger.Error("Failed to cleanup", "error", err)
	} else {
		logger.Info("Step 4: ✓ Cleanup completed")
	}

	logger.Info("========== Incremental cycle completed ==========",
		"next_cycle_in_hours", cfg.SmartMoney.SyncIntervalHours,
	)
}

// runAnalysis 运行 PNL 计算、评分、信号聚合。
func runAnalysis(service *smartmoney.Service, cfg *config.Config) {
	// 1. 计算交易盈亏
	logger.Info("Step 2: Calculating PNL for trades")
	if err := service.CalculatePNLForTrades(); err != nil {
		logger.Error("Failed to calculate PNL", "error", err)
	} else {
		logger.Info("Step 2: ✓ PNL calculated")
	}

	// 2. 计算钱包评分
	logger.Info("Step 3: Calculating wallet scores")
	if err := service.CalculateWalletScores(); err != nil {
		logger.Error("Failed to calculate scores", "error", err)
	} else {
		logger.Info("Step 3: ✓ Wallet scores calculated")
	}

	// 3. 聚合代币信号
	logger.Info("Step 4: Aggregating token signals")
	if err := service.AggregateTokenSignals(cfg.SmartMoney.SignalDays); err != nil {
		logger.Error("Failed to aggregate signals", "error", err)
	} else {
		logger.Info("Step 4: ✓ Token signals aggregated")
	}
}
