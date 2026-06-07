// Package executor 编排跟单执行：策略判断 -> 解密钱包 -> 发送 Swap 交易。
package executor

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"

	"copyflow/internal/chain"
	"copyflow/internal/dex"
	"copyflow/internal/listener"
	"copyflow/internal/model"
	"copyflow/internal/store"
	"copyflow/internal/strategy"
	walletcrypto "copyflow/pkg/crypto"

)

// Service 跟单执行服务。
// 扩展点：消息队列、失败重试、nonce 管理、Gas 策略。
type Service struct {
	chains      *chain.Registry
	dexes       *dex.Registry
	store       *store.Store
	strategy    *strategy.Evaluator
	encryptKey  string
}

// NewService 创建跟单执行服务。
func NewService(chains *chain.Registry, dexes *dex.Registry, st *store.Store, encryptKey string) *Service {
	return &Service{
		chains:     chains,
		dexes:      dexes,
		store:      st,
		strategy:   strategy.NewEvaluator(),
		encryptKey: encryptKey,
	}
}

// ProcessLeaderTx 处理一笔领头交易：落库 -> 匹配配置 -> 执行跟单。
func (s *Service) ProcessLeaderTx(ctx context.Context, meta listener.LeaderTxMeta) error {
	leaderTrade := dex.LeaderTradeFromSwap(meta.ChainID, meta.LeaderAddress, meta.TxHash, meta.BlockNumber, meta.Swap)
	leader, created, err := s.store.CreateLeaderTradeIfNotExists(leaderTrade)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}

	configs, err := s.store.ListActiveConfigsByLeader(meta.ChainID, meta.LeaderAddress)
	if err != nil {
		return err
	}

	for _, cfg := range configs {
		if err := s.executeForConfig(ctx, &cfg, leader, meta.Swap); err != nil {
			log.Printf("[executor] config %d error: %v", cfg.ID, err)
		}
	}
	return nil
}

// executeForConfig 为单个配置执行跟单：策略判断 -> 签名 -> 广播。
func (s *Service) executeForConfig(ctx context.Context, cfg *model.CopyConfig, leader *model.LeaderTrade, swap *dex.SwapInfo) error {
	decision, err := s.strategy.Evaluate(cfg, swap)
	if err != nil {
		return err
	}

	copyTrade := &model.CopyTrade{
		UserID:        cfg.UserID,
		ConfigID:      cfg.ID,
		LeaderTradeID: leader.ID,
		Status:        model.TradeStatusPending,
		AmountIn:      "0",
		TokenOut:      swap.TokenOut.Hex(),
	}

	if !decision.ShouldCopy {
		reason := decision.Reason
		copyTrade.Status = model.TradeStatusSkipped
		copyTrade.ErrorMsg = &reason
		return s.store.CreateCopyTrade(copyTrade)
	}

	copyTrade.AmountIn = decision.AmountIn.String()

	wallet, err := s.store.GetCopyWallet(cfg.UserID, cfg.ChainID)
	if err != nil {
		msg := "copy wallet not found"
		copyTrade.Status = model.TradeStatusFailed
		copyTrade.ErrorMsg = &msg
		_ = s.store.CreateCopyTrade(copyTrade)
		return fmt.Errorf(msg)
	}

	pk, err := walletcrypto.DecryptPrivateKey(wallet.EncryptedPrivateKey, s.encryptKey)
	if err != nil {
		msg := "decrypt wallet failed"
		copyTrade.Status = model.TradeStatusFailed
		copyTrade.ErrorMsg = &msg
		_ = s.store.CreateCopyTrade(copyTrade)
		return err
	}

	client, ok := s.chains.Get(cfg.ChainID)
	if !ok {
		msg := "chain not configured"
		copyTrade.Status = model.TradeStatusFailed
		copyTrade.ErrorMsg = &msg
		_ = s.store.CreateCopyTrade(copyTrade)
		return fmt.Errorf(msg)
	}

	exec, ok := s.dexes.Executor(cfg.DEXType)
	if !ok {
		msg := "dex executor not found"
		copyTrade.Status = model.TradeStatusFailed
		copyTrade.ErrorMsg = &msg
		_ = s.store.CreateCopyTrade(copyTrade)
		return fmt.Errorf(msg)
	}

	if err := s.store.CreateCopyTrade(copyTrade); err != nil {
		return err
	}

	leaderAmountOut, _ := new(big.Int).SetString(leader.AmountOut, 10)
	hash, err := exec.ExecuteBuy(ctx, client, dex.ExecuteParams{
		PrivateKeyHex:   pk,
		TokenOut:        swap.TokenOut,
		AmountIn:        decision.AmountIn,
		SlippageBps:     cfg.SlippageBps,
		LeaderAmountOut: leaderAmountOut,
		Path:            swap.Path,
	})
	if err != nil {
		msg := err.Error()
		copyTrade.Status = model.TradeStatusFailed
		copyTrade.ErrorMsg = &msg
		return s.store.UpdateCopyTrade(copyTrade)
	}

	hashStr := strings.ToLower(hash.Hex())
	copyTrade.TxHash = &hashStr
	copyTrade.Status = model.TradeStatusSubmitted
	return s.store.UpdateCopyTrade(copyTrade)
}

