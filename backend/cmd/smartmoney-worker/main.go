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
		cfg.SmartMoney.Pairs,
		cfg.SmartMoney.EvalDays,
		cfg.SmartMoney.SeedWallets,
		cfg.SmartMoney.MinWalletScore,
	)

	mode := "pair_filter"
	if service.IsSeedMode() {
		mode = "seed_wallets"
	}
	logger.Info("SmartMoney Worker started",
		"chain_id", cfg.SmartMoney.ChainID,
		"sync_interval_hours", cfg.SmartMoney.SyncIntervalHours,
		"min_amount_usd", cfg.SmartMoney.MinAmountUSD,
		"retention_days", cfg.SmartMoney.RetentionDays,
		"eval_days", cfg.SmartMoney.EvalDays,
		"mode", mode,
		"seed_wallets_count", len(cfg.SmartMoney.SeedWallets),
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

// syncOnce 根据模式调用对应的同步方法。
func syncOnce(service *smartmoney.Service, startTime, endTime time.Time, minAmountUSD float64, syncType string) {
	var err error
	if service.IsSeedMode() {
		err = service.SyncSeedWallets(startTime, endTime, minAmountUSD, syncType)
	} else {
		err = service.SyncTradesFromTheGraph(startTime, endTime, minAmountUSD, syncType)
	}
	if err != nil {
		logger.Error("Sync failed", "type", syncType, "error", err)
	}
}

// runHistoricalSync 拉取历史数据（后台任务，只运行一次），窗口由 eval_days 决定。
func runHistoricalSync(service *smartmoney.Service, cfg *config.Config) {
	evalDays := cfg.SmartMoney.EvalDays
	if evalDays <= 0 {
		evalDays = 30
	}
	logger.Info("========== Starting historical sync ==========", "days", evalDays)

	endTime := time.Now()                         // 当前时间
	startTime := endTime.AddDate(0, 0, -evalDays) // 往前 eval_days 天

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

		syncOnce(service, currentStart, currentEnd, cfg.SmartMoney.MinAmountUSD, "historical")

		currentStart = currentEnd
		time.Sleep(5 * time.Second) // 避免请求过快
	}

	logger.Info("========== Historical sync completed ==========")

	// 历史数据同步完成后，执行一次完整的 PNL 计算和评分
	logger.Info("Running post-historical analysis...")
	runAnalysis(service, cfg)
}

// runIncrementalCycle 执行增量同步（拉取最近 1 小时的数据）。
func runIncrementalCycle(service *smartmoney.Service, cfg *config.Config) {
	logger.Info("========== Starting incremental cycle ==========")

	// 最近 1 小时：从 (now - 1h) 到 now
	now := time.Now()
	intervalHours := cfg.SmartMoney.SyncIntervalHours
	if intervalHours <= 0 {
		intervalHours = 1
	}
	endTime := now
	startTime := now.Add(-time.Duration(intervalHours) * time.Hour)

	// 1. 同步最近一个周期的交易数据
	logger.Info("Step 1: Syncing recent trades",
		"from", startTime.Format("2006-01-02 15:04:05"),
		"to", endTime.Format("2006-01-02 15:04:05"),
	)
	syncOnce(service, startTime, endTime, cfg.SmartMoney.MinAmountUSD, "incremental")
	logger.Info("Step 1: ✓ Trades synced")

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
