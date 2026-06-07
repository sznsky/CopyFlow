// Worker 入口：轮询扫描链上区块，监听领头地址交易并执行跟单。
package main

import (
	"context"
	"log"
	"time"

	"copyflow/internal/bootstrap"
	"copyflow/internal/config"
	"copyflow/internal/executor"
	"copyflow/internal/listener"
	"copyflow/internal/store"
)

// main 启动 Worker，定时扫描链上交易并执行跟单。
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if !cfg.Worker.Enabled {
		log.Println("worker disabled in config")
		return
	}

	st, err := store.New(cfg.Database.DSN)
	if err != nil {
		log.Fatal(err)
	}
	if err := st.AutoMigrate(); err != nil {
		log.Fatal(err)
	}

	chains, err := bootstrap.BuildChains(cfg)
	if err != nil {
		log.Fatal(err)
	}
	dexes, err := bootstrap.BuildDEXRegistry(cfg)
	if err != nil {
		log.Fatal(err)
	}

	scanner := listener.NewScanner(chains, dexes, st)
	execSvc := executor.NewService(chains, dexes, st, cfg.Auth.WalletEncryptKey)

	interval := time.Duration(cfg.Worker.PollIntervalSec) * time.Second
	log.Printf("worker started, poll interval=%s", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		runOnce(context.Background(), cfg, scanner, execSvc)
		<-ticker.C
	}
}

// runOnce 执行一轮：刷新领头地址 -> 扫描各链 -> 处理跟单。
func runOnce(ctx context.Context, cfg *config.Config, scanner *listener.Scanner, execSvc *executor.Service) {
	if err := scanner.RefreshLeaders(ctx); err != nil {
		log.Printf("[worker] refresh leaders: %v", err)
		return
	}

	for _, ch := range cfg.Chains {
		if !ch.Enabled {
			continue
		}
		metas, err := scanner.ScanChain(ctx, ch.ChainID, cfg.Worker.Confirmations)
		if err != nil {
			log.Printf("[worker] scan chain %d: %v", ch.ChainID, err)
			continue
		}
		for _, meta := range metas {
			if err := execSvc.ProcessLeaderTx(ctx, meta); err != nil {
				log.Printf("[worker] process tx %s: %v", meta.TxHash, err)
			}
		}
	}

	// 确认已广播的跟单交易最终状态
	if err := execSvc.ConfirmPendingTrades(ctx); err != nil {
		log.Printf("[worker] confirm trades: %v", err)
	}
}
