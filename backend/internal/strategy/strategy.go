// Package strategy 跟单策略引擎，判断是否跟单及计算金额。
package strategy

import (
	"fmt"
	"math/big"

	"copyflow/internal/dex"
	"copyflow/internal/model"

	"github.com/shopspring/decimal"
)

// Evaluator 策略评估器。
// 扩展点：代币白名单、蜜罐检测、每日限额。
type Evaluator struct{}

// NewEvaluator 创建策略评估器。
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Decision 策略评估结果。
type Decision struct {
	ShouldCopy bool     // 是否执行跟单
	AmountIn   *big.Int // 跟单投入量（wei）
	Reason     string   // 跳过原因（ShouldCopy=false 时）
}

// Evaluate 评估是否跟单并计算跟单金额。
func (e *Evaluator) Evaluate(cfg *model.CopyConfig, swap *dex.SwapInfo) (*Decision, error) {
	if !cfg.IsActive {
		return &Decision{ShouldCopy: false, Reason: "config inactive"}, nil
	}
	if cfg.DEXType != swap.DEXType {
		return &Decision{ShouldCopy: false, Reason: "dex mismatch"}, nil
	}
	if !swap.IsBuy {
		return &Decision{ShouldCopy: false, Reason: "not a buy swap"}, nil
	}

	amount, err := e.calcAmount(cfg, swap.AmountIn)
	if err != nil {
		return nil, err
	}
	if amount.Sign() <= 0 {
		return &Decision{ShouldCopy: false, Reason: "zero copy amount"}, nil
	}

	return &Decision{ShouldCopy: true, AmountIn: amount}, nil
}

// calcAmount 按固定金额或等比例计算跟单投入量。
func (e *Evaluator) calcAmount(cfg *model.CopyConfig, leaderAmount *big.Int) (*big.Int, error) {
	switch cfg.CopyMode {
	case model.CopyModeFixed:
		// copy_amount in BNB/ETH units (18 decimals)
		wei := cfg.CopyAmount.Mul(decimal.New(1, 18)).BigInt()
		return capAmount(wei, cfg), nil
	case model.CopyModeRatio:
		// copy_amount as ratio e.g. 0.1 = 10%
		ratio := cfg.CopyAmount
		if ratio.IsNegative() || ratio.GreaterThan(decimal.NewFromInt(1)) {
			return nil, fmt.Errorf("ratio must be between 0 and 1")
		}
		leader := decimal.NewFromBigInt(leaderAmount, 0)
		copyWei := leader.Mul(ratio).BigInt()
		return capAmount(copyWei, cfg), nil
	default:
		return nil, fmt.Errorf("unknown copy mode: %s", cfg.CopyMode)
	}
}

// capAmount 将金额限制在单笔上限内。
func capAmount(amount *big.Int, cfg *model.CopyConfig) *big.Int {
	if cfg.MaxPerTrade.IsPositive() {
		maxWei := cfg.MaxPerTrade.Mul(decimal.New(1, 18)).BigInt()
		if amount.Cmp(maxWei) > 0 {
			return maxWei
		}
	}
	return amount
}
