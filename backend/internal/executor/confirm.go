package executor

import (
	"context"
	"log"

	"copyflow/internal/model"

	"github.com/ethereum/go-ethereum/common"
)

// ConfirmPendingTrades 轮询已提交交易的链上 receipt，更新最终状态。
func (s *Service) ConfirmPendingTrades(ctx context.Context) error {
	trades, err := s.store.ListSubmittedCopyTrades(50)
	if err != nil {
		return err
	}
	for _, t := range trades {
		if err := s.confirmOne(ctx, &t); err != nil {
			log.Printf("[executor] confirm trade %d: %v", t.ID, err)
		}
	}
	return nil
}

// confirmOne 确认单笔跟单交易的链上最终状态。
func (s *Service) confirmOne(ctx context.Context, t *model.CopyTrade) error {
	if t.TxHash == nil || *t.TxHash == "" {
		return nil
	}
	cfg, err := s.store.GetCopyConfigByID(t.ConfigID)
	if err != nil {
		return err
	}
	client, ok := s.chains.Get(cfg.ChainID)
	if !ok {
		return nil
	}

	receipt, err := client.TransactionReceipt(ctx, common.HexToHash(*t.TxHash))
	if err != nil {
		// 交易尚未上链，等待下一轮
		return nil
	}

	gasUsed := receipt.GasUsed
	t.GasUsed = &gasUsed
	if receipt.Status == 1 {
		t.Status = model.TradeStatusSuccess
		t.ErrorMsg = nil
	} else {
		t.Status = model.TradeStatusFailed
		msg := "链上执行失败 (reverted)"
		t.ErrorMsg = &msg
	}
	return s.store.UpdateCopyTrade(t)
}
