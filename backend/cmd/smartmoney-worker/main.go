// SmartMoney Worker: 定期同步 Dune 数据、计算评分、聚合信号。
package main

import (
	"log"
	"time"

	"copyflow/internal/config"
	"copyflow/internal/smartmoney"
	"copyflow/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if !cfg.SmartMoney.Enabled {
		log.Println("[SmartMoney Worker] Disabled in config, exiting")
		return
	}

	if !cfg.Dune.Enabled {
		log.Println("[SmartMoney Worker] Dune API disabled, exiting")
		return
	}

	st, err := store.New(cfg.Database.DSN)
	if err != nil {
		log.Fatal(err)
	}
	if err := st.AutoMigrate(); err != nil {
		log.Fatal(err)
	}

	service := smartmoney.NewService(st, cfg.Dune.APIKey, cfg.SmartMoney.ChainID)

	log.Printf("[SmartMoney Worker] Started for chain %d", cfg.SmartMoney.ChainID)
	log.Printf("[SmartMoney Worker] Sync interval: %d hours", cfg.SmartMoney.SyncIntervalHours)

	// 首次执行
	runFullCycle(service, cfg)

	// 定期执行
	ticker := time.NewTicker(time.Duration(cfg.SmartMoney.SyncIntervalHours) * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		runFullCycle(service, cfg)
	}
}

// runFullCycle 执行完整的数据同步→评分→信号聚合周期。
func runFullCycle(service *smartmoney.Service, cfg *config.Config) {
	log.Println("[SmartMoney Worker] ========== Starting full cycle ==========")

	// 1. 从 Dune 同步交易数据
	log.Println("[SmartMoney Worker] Step 1: Syncing trades from Dune...")
	if err := service.SyncTradesFromDune(cfg.Dune.QueryID, cfg.SmartMoney.MinAmountUSD, cfg.SmartMoney.DaysBack); err != nil {
		log.Printf("[SmartMoney Worker] ERROR: Failed to sync trades: %v", err)
	} else {
		log.Println("[SmartMoney Worker] Step 1: ✓ Trades synced")
	}

	// 等待一段时间，避免对 Dune API 请求过快
	time.Sleep(5 * time.Second)

	// 2. 计算交易盈亏
	log.Println("[SmartMoney Worker] Step 2: Calculating PNL for trades...")
	if err := service.CalculatePNLForTrades(); err != nil {
		log.Printf("[SmartMoney Worker] ERROR: Failed to calculate PNL: %v", err)
	} else {
		log.Println("[SmartMoney Worker] Step 2: ✓ PNL calculated")
	}

	// 3. 计算钱包评分
	log.Println("[SmartMoney Worker] Step 3: Calculating wallet scores...")
	if err := service.CalculateWalletScores(); err != nil {
		log.Printf("[SmartMoney Worker] ERROR: Failed to calculate scores: %v", err)
	} else {
		log.Println("[SmartMoney Worker] Step 3: ✓ Wallet scores calculated")
	}

	// 4. 聚合代币信号
	log.Println("[SmartMoney Worker] Step 4: Aggregating token signals...")
	if err := service.AggregateTokenSignals(cfg.SmartMoney.SignalDays); err != nil {
		log.Printf("[SmartMoney Worker] ERROR: Failed to aggregate signals: %v", err)
	} else {
		log.Println("[SmartMoney Worker] Step 4: ✓ Token signals aggregated")
	}

	log.Println("[SmartMoney Worker] ========== Full cycle completed ==========")
	log.Printf("[SmartMoney Worker] Next cycle in %d hours\n", cfg.SmartMoney.SyncIntervalHours)
}
