package smartmoney

import (
	"errors"
	"fmt"
	"math"
	"time"

	"copyflow/internal/model"
	"copyflow/pkg/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// CalculateWalletScores 计算所有钱包的评分。
func (s *Service) CalculateWalletScores() error {
	logger.Info("Starting wallet score calculation")

	// 评估周期：使用可配置的 evalDays（默认 30 天）
	endDate := time.Now()
	evalDays := s.evalDays
	if evalDays <= 0 {
		evalDays = 30
	}
	startDate := endDate.AddDate(0, 0, -evalDays)
	
	// 获取在评估周期内有交易的所有钱包
	var walletAddresses []string
	err := s.store.DB().Model(&model.WalletTrade{}).
		Where("chain_id = ? AND block_time >= ? AND block_time <= ?",
			s.chainID, startDate, endDate).
		Distinct("wallet_address").
		Pluck("wallet_address", &walletAddresses).Error
	
	if err != nil {
		return fmt.Errorf("fetch wallet addresses: %w", err)
	}
	
	logger.Info("Found wallets to evaluate", "count", len(walletAddresses))
	
	for _, walletAddr := range walletAddresses {
		if err := s.calculateSingleWalletScore(walletAddr, startDate, endDate); err != nil {
			logger.Error("Failed to calculate score",
				"wallet", walletAddr,
				"error", err,
			)
			continue
		}
	}
	
	// 更新排名
	if err := s.updateWalletRankings(); err != nil {
		return fmt.Errorf("update rankings: %w", err)
	}
	
	logger.Info("Wallet score calculation completed")
	return nil
}

// calculateSingleWalletScore 计算单个钱包的评分。
func (s *Service) calculateSingleWalletScore(walletAddr string, startDate, endDate time.Time) error {
	// 获取钱包的所有交易
	var trades []model.WalletTrade
	err := s.store.DB().Where("wallet_address = ? AND chain_id = ? AND block_time >= ? AND block_time <= ?",
		walletAddr, s.chainID, startDate, endDate).
		Order("block_time ASC").
		Find(&trades).Error
	
	if err != nil {
		return fmt.Errorf("fetch trades: %w", err)
	}
	
	if len(trades) == 0 {
		return nil
	}
	
	// 计算评分维度
	totalPNL := decimal.Zero
	totalVolume := decimal.Zero
	winningTrades := 0
	losingTrades := 0
	mainstreamVolume := decimal.Zero
	
	var pnlHistory []decimal.Decimal
	cumPNL := decimal.Zero
	
	for _, trade := range trades {
		totalVolume = totalVolume.Add(trade.AmountUSD)
		
		// 计算盈亏
		if trade.PnlUSD != nil {
			pnl := *trade.PnlUSD
			totalPNL = totalPNL.Add(pnl)
			cumPNL = cumPNL.Add(pnl)
			pnlHistory = append(pnlHistory, cumPNL)
			
			if pnl.GreaterThan(decimal.Zero) {
				winningTrades++
			} else if pnl.LessThan(decimal.Zero) {
				losingTrades++
			}
		}
		
		// 主流币判断（ETH、BTC、USDT、USDC、DAI、WBTC、WETH）
		if s.isMainstreamToken(trade.TokenOut) {
			mainstreamVolume = mainstreamVolume.Add(trade.AmountUSD)
		}
	}
	
	totalTrades := len(trades)
	completedTrades := winningTrades + losingTrades
	
	// 1. 胜率
	winRate := decimal.Zero
	if completedTrades > 0 {
		winRate = decimal.NewFromInt(int64(winningTrades)).
			Div(decimal.NewFromInt(int64(completedTrades))).
			Mul(decimal.NewFromInt(100))
	}
	
	// 2. 盈亏比
	profitLossRatio := decimal.Zero
	if completedTrades > 0 {
		avgWin := decimal.Zero
		avgLoss := decimal.Zero
		
		for _, trade := range trades {
			if trade.PnlUSD != nil {
				if trade.PnlUSD.GreaterThan(decimal.Zero) {
					avgWin = avgWin.Add(*trade.PnlUSD)
				} else if trade.PnlUSD.LessThan(decimal.Zero) {
					avgLoss = avgLoss.Add(trade.PnlUSD.Abs())
				}
			}
		}
		
		if winningTrades > 0 {
			avgWin = avgWin.Div(decimal.NewFromInt(int64(winningTrades)))
		}
		if losingTrades > 0 {
			avgLoss = avgLoss.Div(decimal.NewFromInt(int64(losingTrades)))
		}
		
		if avgLoss.GreaterThan(decimal.Zero) {
			profitLossRatio = avgWin.Div(avgLoss)
		}
	}
	
	// 3. 最大回撤
	maxDrawdown := decimal.Zero
	if len(pnlHistory) > 0 {
		peak := pnlHistory[0]
		for _, cum := range pnlHistory {
			if cum.GreaterThan(peak) {
				peak = cum
			}
			drawdown := peak.Sub(cum)
			if drawdown.GreaterThan(maxDrawdown) {
				maxDrawdown = drawdown
			}
		}
		
		// 转换为百分比
		if peak.GreaterThan(decimal.Zero) {
			maxDrawdown = maxDrawdown.Div(peak).Mul(decimal.NewFromInt(100))
		}
	}
	
	// 4. 主流币占比
	mainstreamRatio := decimal.Zero
	if totalVolume.GreaterThan(decimal.Zero) {
		mainstreamRatio = mainstreamVolume.Div(totalVolume).Mul(decimal.NewFromInt(100))
	}
	
	// 5. 交易频率（次/天）
	days := endDate.Sub(startDate).Hours() / 24
	tradeFrequency := decimal.NewFromFloat(float64(totalTrades) / days)
	
	// 计算综合评分（0-100）
	score := s.calculateCompositeScore(
		totalPNL,
		winRate,
		profitLossRatio,
		maxDrawdown,
		mainstreamRatio,
		tradeFrequency,
		totalVolume,
	)
	
	// 保存或更新钱包评分
	wallet := &model.SmartWallet{
		WalletAddress:       walletAddr,
		ChainID:             s.chainID,
		Score:               score,
		TotalPNL:            totalPNL,
		WinRate:             winRate,
		ProfitLossRatio:     profitLossRatio,
		MaxDrawdown:         maxDrawdown,
		MainstreamRatio:     mainstreamRatio,
		TradeFrequency:      tradeFrequency,
		TotalTrades:         totalTrades,
		WinningTrades:       winningTrades,
		TotalVolume:         totalVolume,
		EvaluationStartDate: startDate,
		EvaluationEndDate:   endDate,
	}

	// 先查再决定 Create 还是 Save，避免 FirstOrCreate+Assign 在更新时丢字段的 bug
	var existing model.SmartWallet
	result := s.store.DB().Where("wallet_address = ? AND chain_id = ?", walletAddr, s.chainID).First(&existing)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		if err := s.store.DB().Create(wallet).Error; err != nil {
			return fmt.Errorf("create wallet score: %w", err)
		}
	} else if result.Error != nil {
		return fmt.Errorf("query wallet: %w", result.Error)
	} else {
		wallet.ID = existing.ID
		wallet.CreatedAt = existing.CreatedAt // 保留原始创建时间，避免零值写入 MySQL
		if err := s.store.DB().Save(wallet).Error; err != nil {
			return fmt.Errorf("update wallet score: %w", err)
		}
	}

	return nil
}

// calculateCompositeScore 计算综合评分（0-100）。
func (s *Service) calculateCompositeScore(
	totalPNL, winRate, profitLossRatio, maxDrawdown, mainstreamRatio, tradeFrequency, totalVolume decimal.Decimal,
) decimal.Decimal {
	score := decimal.Zero
	
	// 1. 累计盈亏得分（0-30分）
	// 盈亏 > 10000 USD: 30分
	// 盈亏 > 5000 USD: 20分
	// 盈亏 > 1000 USD: 10分
	// 盈亏 > 0: 5分
	pnlScore := decimal.Zero
	if totalPNL.GreaterThan(decimal.NewFromInt(10000)) {
		pnlScore = decimal.NewFromInt(30)
	} else if totalPNL.GreaterThan(decimal.NewFromInt(5000)) {
		pnlScore = decimal.NewFromInt(20)
	} else if totalPNL.GreaterThan(decimal.NewFromInt(1000)) {
		pnlScore = decimal.NewFromInt(10)
	} else if totalPNL.GreaterThan(decimal.Zero) {
		pnlScore = decimal.NewFromInt(5)
	}
	score = score.Add(pnlScore)
	
	// 2. 胜率得分（0-20分）
	// 胜率 >= 70%: 20分
	// 胜率 >= 60%: 15分
	// 胜率 >= 50%: 10分
	// 胜率 >= 40%: 5分
	winRateScore := decimal.Zero
	if winRate.GreaterThanOrEqual(decimal.NewFromInt(70)) {
		winRateScore = decimal.NewFromInt(20)
	} else if winRate.GreaterThanOrEqual(decimal.NewFromInt(60)) {
		winRateScore = decimal.NewFromInt(15)
	} else if winRate.GreaterThanOrEqual(decimal.NewFromInt(50)) {
		winRateScore = decimal.NewFromInt(10)
	} else if winRate.GreaterThanOrEqual(decimal.NewFromInt(40)) {
		winRateScore = decimal.NewFromInt(5)
	}
	score = score.Add(winRateScore)
	
	// 3. 盈亏比得分（0-15分）
	// 盈亏比 >= 3: 15分
	// 盈亏比 >= 2: 10分
	// 盈亏比 >= 1.5: 5分
	plRatioScore := decimal.Zero
	if profitLossRatio.GreaterThanOrEqual(decimal.NewFromInt(3)) {
		plRatioScore = decimal.NewFromInt(15)
	} else if profitLossRatio.GreaterThanOrEqual(decimal.NewFromInt(2)) {
		plRatioScore = decimal.NewFromInt(10)
	} else if profitLossRatio.GreaterThanOrEqual(decimal.NewFromFloat(1.5)) {
		plRatioScore = decimal.NewFromInt(5)
	}
	score = score.Add(plRatioScore)
	
	// 4. 最大回撤得分（0-15分，回撤越小得分越高）
	// 回撤 < 10%: 15分
	// 回撤 < 20%: 10分
	// 回撤 < 30%: 5分
	drawdownScore := decimal.Zero
	if maxDrawdown.LessThan(decimal.NewFromInt(10)) {
		drawdownScore = decimal.NewFromInt(15)
	} else if maxDrawdown.LessThan(decimal.NewFromInt(20)) {
		drawdownScore = decimal.NewFromInt(10)
	} else if maxDrawdown.LessThan(decimal.NewFromInt(30)) {
		drawdownScore = decimal.NewFromInt(5)
	}
	score = score.Add(drawdownScore)
	
	// 5. 主流币占比得分（0-10分）
	// 占比 >= 50%: 10分
	// 占比 >= 30%: 5分
	mainstreamScore := decimal.Zero
	if mainstreamRatio.GreaterThanOrEqual(decimal.NewFromInt(50)) {
		mainstreamScore = decimal.NewFromInt(10)
	} else if mainstreamRatio.GreaterThanOrEqual(decimal.NewFromInt(30)) {
		mainstreamScore = decimal.NewFromInt(5)
	}
	score = score.Add(mainstreamScore)
	
	// 6. 交易频率得分（0-10分）
	// 频率 > 5次/天: 10分
	// 频率 > 2次/天: 5分
	// 频率 > 0.5次/天: 2分
	frequencyScore := decimal.Zero
	if tradeFrequency.GreaterThan(decimal.NewFromInt(5)) {
		frequencyScore = decimal.NewFromInt(10)
	} else if tradeFrequency.GreaterThan(decimal.NewFromInt(2)) {
		frequencyScore = decimal.NewFromInt(5)
	} else if tradeFrequency.GreaterThan(decimal.NewFromFloat(0.5)) {
		frequencyScore = decimal.NewFromInt(2)
	}
	score = score.Add(frequencyScore)
	
	// 确保得分在 0-100 之间
	if score.LessThan(decimal.Zero) {
		score = decimal.Zero
	}
	if score.GreaterThan(decimal.NewFromInt(100)) {
		score = decimal.NewFromInt(100)
	}
	
	return score
}

// isMainstreamToken 判断是否为主流币。
func (s *Service) isMainstreamToken(tokenAddr string) bool {
	// Ethereum 主网主流币地址（小写）
	mainstream := map[string]bool{
		"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, // WETH
		"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": true, // WBTC
		"0xdac17f958d2ee523a2206206994597c13d831ec7": true, // USDT
		"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": true, // USDC
		"0x6b175474e89094c44da98b954eedeac495271d0f": true, // DAI
		"0xae7ab96520de3a18e5e111b5eaab095312d7fe84": true, // stETH
	}
	
	return mainstream[tokenAddr]
}

// updateWalletRankings 更新钱包排名。
func (s *Service) updateWalletRankings() error {
	// 获取所有钱包并按得分排序
	var wallets []model.SmartWallet
	err := s.store.DB().Where("chain_id = ?", s.chainID).
		Order("score DESC, total_pnl DESC").
		Find(&wallets).Error
	
	if err != nil {
		return fmt.Errorf("fetch wallets: %w", err)
	}
	
	// 更新排名和 Top 20 标记
	for i, wallet := range wallets {
		rank := i + 1
		isTop := rank <= 20
		
		err := s.store.DB().Model(&wallet).Updates(map[string]interface{}{
			"rank_position": rank,
			"is_top_wallet": isTop,
		}).Error
		
		if err != nil {
			logger.Error("Failed to update ranking",
				"wallet", wallet.WalletAddress,
				"error", err,
			)
		}
	}
	
	logger.Info("Updated wallet rankings", "count", len(wallets))
	return nil
}

// GetTopWallets 获取 Top N 钱包。
func (s *Service) GetTopWallets(limit int, minScore float64) ([]model.SmartWallet, error) {
	var wallets []model.SmartWallet

	query := s.store.DB().Where("chain_id = ?", s.chainID)

	if minScore > 0 {
		query = query.Where("score >= ?", minScore)
	}

	err := query.Order("score DESC, total_pnl DESC").
		Limit(limit).
		Find(&wallets).Error

	if err != nil {
		return nil, fmt.Errorf("fetch top wallets: %w", err)
	}

	return wallets, nil
}

// RecentActivityItem 首页聪明钱动态条目。
type RecentActivityItem struct {
	WalletAddress string          `json:"wallet_address"`
	WalletScore   decimal.Decimal `json:"wallet_score"`
	RankPosition  int             `json:"rank_position"`
	TxHash        string          `json:"tx_hash"`
	IsBuy         bool            `json:"is_buy"`
	TokenIn       string          `json:"token_in"`
	TokenOut      string          `json:"token_out"`
	TokenInSymbol string          `json:"token_in_symbol"`
	TokenOutSymbol string         `json:"token_out_symbol"`
	AmountUSD     decimal.Decimal `json:"amount_usd"`
	BlockTime     string          `json:"block_time"`
}

// GetRecentActivity 获取高分钱包最近的交易动态，按 block_time 倒序。
// 直接 JOIN wallet_trades + smart_wallets，避免先取钱包再逐个查交易。
func (s *Service) GetRecentActivity(minScore float64, limit int) ([]RecentActivityItem, error) {
	type row struct {
		WalletAddress  string
		WalletScore    decimal.Decimal
		RankPosition   int
		TxHash         string
		IsBuy          bool
		TokenIn        string
		TokenOut       string
		TokenInSymbol  string
		TokenOutSymbol string
		AmountUSD      decimal.Decimal
		BlockTime      string
	}

	var rows []row
	err := s.store.DB().Raw(`
		SELECT
			t.wallet_address,
			w.score          AS wallet_score,
			w.rank_position,
			t.tx_hash,
			t.is_buy,
			t.token_in,
			t.token_out,
			t.token_in_symbol,
			t.token_out_symbol,
			t.amount_usd,
			t.block_time
		FROM wallet_trades t
		INNER JOIN smart_wallets w
			ON w.wallet_address = t.wallet_address
			AND w.chain_id = t.chain_id
		WHERE t.chain_id = ?
		  AND w.score >= ?
		ORDER BY t.block_time DESC
		LIMIT ?
	`, s.chainID, minScore, limit).Scan(&rows).Error

	if err != nil {
		return nil, fmt.Errorf("fetch recent activity: %w", err)
	}

	items := make([]RecentActivityItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, RecentActivityItem{
			WalletAddress:  r.WalletAddress,
			WalletScore:    r.WalletScore,
			RankPosition:   r.RankPosition,
			TxHash:         r.TxHash,
			IsBuy:          r.IsBuy,
			TokenIn:        r.TokenIn,
			TokenOut:       r.TokenOut,
			TokenInSymbol:  r.TokenInSymbol,
			TokenOutSymbol: r.TokenOutSymbol,
			AmountUSD:      r.AmountUSD,
			BlockTime:      r.BlockTime,
		})
	}
	return items, nil
}

// DashboardStats 首页统计指标。
type DashboardStats struct {
	MonitoredWallets int64   `json:"monitored_wallets"` // 监控钱包总数（评分 >= minScore）
	TodaySignals     int64   `json:"today_signals"`     // 最新信号周期信号数
	TopScore         float64 `json:"top_score"`         // 最高评分
	AvgWinRate       float64 `json:"avg_win_rate"`      // 平均胜率（%）
}

// GetDashboardStats 聚合首页统计指标。
func (s *Service) GetDashboardStats(minScore float64) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// 1. 监控钱包总数
	if err := s.store.DB().Model(&model.SmartWallet{}).
		Where("chain_id = ? AND score >= ?", s.chainID, minScore).
		Count(&stats.MonitoredWallets).Error; err != nil {
		return nil, fmt.Errorf("count monitored wallets: %w", err)
	}

	// 2. 最高评分 + 平均胜率
	var agg struct {
		TopScore   decimal.Decimal
		AvgWinRate decimal.Decimal
	}
	if err := s.store.DB().Model(&model.SmartWallet{}).
		Where("chain_id = ? AND score >= ?", s.chainID, minScore).
		Select("COALESCE(MAX(score), 0) AS top_score, COALESCE(AVG(win_rate), 0) AS avg_win_rate").
		Scan(&agg).Error; err != nil {
		return nil, fmt.Errorf("aggregate wallet stats: %w", err)
	}
	stats.TopScore, _ = agg.TopScore.Float64()
	stats.AvgWinRate, _ = agg.AvgWinRate.Float64()

	// 3. 最新信号周期的信号总数
	var latestStartDate time.Time
	if err := s.store.DB().Model(&model.TokenSignal{}).
		Where("chain_id = ?", s.chainID).
		Select("MAX(signal_start_date)").
		Scan(&latestStartDate).Error; err != nil {
		return nil, fmt.Errorf("fetch latest signal period: %w", err)
	}
	if !latestStartDate.IsZero() {
		if err := s.store.DB().Model(&model.TokenSignal{}).
			Where("chain_id = ? AND signal_start_date = ?", s.chainID, latestStartDate).
			Count(&stats.TodaySignals).Error; err != nil {
			return nil, fmt.Errorf("count today signals: %w", err)
		}
	}

	return stats, nil
}

// CalculatePNLForTrades 为交易计算盈亏（匹配买入和卖出）。
func (s *Service) CalculatePNLForTrades() error {
	logger.Info("Starting PNL calculation for trades")
	
	// 获取所有卖出交易（未计算盈亏的）
	var sellTrades []model.WalletTrade
	err := s.store.DB().Where("chain_id = ? AND is_buy = ? AND pnl_usd IS NULL",
		s.chainID, false).
		Order("block_time ASC").
		Find(&sellTrades).Error
	
	if err != nil {
		return fmt.Errorf("fetch sell trades: %w", err)
	}
	
	logger.Info("Found sell trades to calculate PNL", "count", len(sellTrades))
	
	matched := 0
	skipped := 0
	
	for _, sellTrade := range sellTrades {
		// 查找对应的买入交易（FIFO）
		var buyTrade model.WalletTrade
		err := s.store.DB().Where(
			"wallet_address = ? AND chain_id = ? AND token_out = ? AND is_buy = ? AND block_time < ?",
			sellTrade.WalletAddress, s.chainID, sellTrade.TokenIn, true, sellTrade.BlockTime).
			Order("block_time ASC").
			First(&buyTrade).Error
		
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			skipped++
		}
		continue
	}
	
	// 盈亏（USD）
	pnlUSD := sellTrade.AmountUSD.Sub(buyTrade.AmountUSD)
	
	// 盈亏百分比
	pnlPercent := decimal.Zero
	if buyTrade.AmountUSD.GreaterThan(decimal.Zero) {
		pnlPercent = pnlUSD.Div(buyTrade.AmountUSD).Mul(decimal.NewFromInt(100))
	}
	
	// 持仓时长（小时）
	holdingHours := int(math.Round(sellTrade.BlockTime.Sub(buyTrade.BlockTime).Hours()))
	
	// 更新卖出交易的盈亏信息
	err = s.store.DB().Model(&sellTrade).Updates(map[string]interface{}{
		"pnl_usd":                pnlUSD,
		"pnl_percent":            pnlPercent,
		"holding_duration_hours": holdingHours,
	}).Error
	
	if err != nil {
		logger.Error("Failed to update PNL",
			"tx_hash", sellTrade.TxHash,
			"error", err,
		)
		continue
	}
	matched++
}

logger.Info("PNL calculation completed",
	"matched", matched,
	"skipped", skipped,
)
return nil
}
