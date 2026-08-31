package smartmoney

import (
	"fmt"
	"time"

	"copyflow/internal/model"
	"copyflow/pkg/logger"

	"github.com/shopspring/decimal"
)

// AggregateTokenSignals 聚合代币信号（分析高分钱包最近N天的买入）。
func (s *Service) AggregateTokenSignals(days int) error {
	logger.Info("Starting token signal aggregation", "days", days)
	
	// 信号周期
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)
	
	// 获取 Top 钱包
	topWallets, err := s.GetTopWallets(20, s.minWalletScore)
	if err != nil {
		return fmt.Errorf("get top wallets: %w", err)
	}
	
	if len(topWallets) == 0 {
		logger.Info("No top wallets found, skipping signal aggregation")
		return nil
	}
	
	logger.Info("Analyzing trades from top wallets", "wallet_count", len(topWallets))
	
	// 获取这些钱包的地址列表
	walletAddrs := make([]string, len(topWallets))
	walletScoreMap := make(map[string]decimal.Decimal)
	for i, w := range topWallets {
		walletAddrs[i] = w.WalletAddress
		walletScoreMap[w.WalletAddress] = w.Score
	}
	
	// 获取这些钱包在指定时间段内的买入交易
	var buyTrades []model.WalletTrade
	err = s.store.DB().Where("wallet_address IN ? AND chain_id = ? AND is_buy = ? AND block_time >= ? AND block_time <= ?",
		walletAddrs, s.chainID, true, startDate, endDate).
		Order("block_time DESC").
		Find(&buyTrades).Error
	
	if err != nil {
		return fmt.Errorf("fetch buy trades: %w", err)
	}
	
	logger.Info("Found buy trades to analyze", "count", len(buyTrades))
	
	// 按 token 聚合
	tokenMap := make(map[string]*TokenAggregation)
	
	for _, trade := range buyTrades {
		tokenAddr := trade.TokenOut
		
		if _, exists := tokenMap[tokenAddr]; !exists {
			tokenMap[tokenAddr] = &TokenAggregation{
				TokenAddress:     tokenAddr,
				TokenSymbol:      trade.TokenOutSymbol,
				WalletBuys:       make(map[string]*WalletBuyInfo),
				TotalBuyVolume:   decimal.Zero,
				SmartWalletCount: 0,
			}
		}
		
		agg := tokenMap[tokenAddr]
		
		// 记录钱包买入信息
		if _, exists := agg.WalletBuys[trade.WalletAddress]; !exists {
			agg.WalletBuys[trade.WalletAddress] = &WalletBuyInfo{
				WalletAddress: trade.WalletAddress,
				WalletScore:   walletScoreMap[trade.WalletAddress],
				Trades:        []model.WalletTrade{},
				TotalVolume:   decimal.Zero,
			}
			agg.SmartWalletCount++
		}
		
		walletBuy := agg.WalletBuys[trade.WalletAddress]
		walletBuy.Trades = append(walletBuy.Trades, trade)
		walletBuy.TotalVolume = walletBuy.TotalVolume.Add(trade.AmountUSD)
		
		agg.TotalBuyVolume = agg.TotalBuyVolume.Add(trade.AmountUSD)
		
		// 更新首次和最后买入时间
		if agg.FirstBuyTime.IsZero() || trade.BlockTime.Before(agg.FirstBuyTime) {
			agg.FirstBuyTime = trade.BlockTime
		}
		if agg.LastBuyTime.IsZero() || trade.BlockTime.After(agg.LastBuyTime) {
			agg.LastBuyTime = trade.BlockTime
		}
	}
	
	// 计算每个代币的共识度评分并保存
	for tokenAddr, agg := range tokenMap {
		// 计算平均买入金额
		avgBuyAmount := decimal.Zero
		if agg.SmartWalletCount > 0 {
			avgBuyAmount = agg.TotalBuyVolume.Div(decimal.NewFromInt(int64(agg.SmartWalletCount)))
		}
		
		// 计算共识度评分
		consensusScore := s.calculateConsensusScore(agg, len(topWallets))
		
		// 创建或更新 TokenSignal
		signal := &model.TokenSignal{
			TokenAddress:     tokenAddr,
			TokenSymbol:      agg.TokenSymbol,
			ChainID:          s.chainID,
			SmartWalletCount: agg.SmartWalletCount,
			TotalBuyVolume:   agg.TotalBuyVolume,
			AvgBuyAmount:     avgBuyAmount,
			FirstBuyTime:     agg.FirstBuyTime,
			LastBuyTime:      agg.LastBuyTime,
			ConsensusScore:   consensusScore,
			SignalStartDate:  startDate,
			SignalEndDate:    endDate,
		}
		
		// Upsert
		var existing model.TokenSignal
		err := s.store.DB().Where("token_address = ? AND chain_id = ? AND signal_start_date = ?",
			tokenAddr, s.chainID, startDate).
			First(&existing).Error
		
		if err != nil {
			// 不存在，创建新记录
			if err := s.store.DB().Create(signal).Error; err != nil {
				logger.Error("Failed to create signal", "token", tokenAddr, "error", err)
				continue
			}
		} else {
			// 已存在，更新
			if err := s.store.DB().Model(&existing).Updates(signal).Error; err != nil {
				logger.Error("Failed to update signal", "token", tokenAddr, "error", err)
				continue
			}
			signal.ID = existing.ID
		}
		
		// 保存信号详情
		if err := s.saveSignalDetails(signal.ID, agg); err != nil {
			logger.Error("Failed to save signal details", "token", tokenAddr, "error", err)
		}
	}
	
	logger.Info("Token signal aggregation completed", "token_count", len(tokenMap))
	return nil
}

// TokenAggregation 代币聚合信息。
type TokenAggregation struct {
	TokenAddress     string
	TokenSymbol      *string
	WalletBuys       map[string]*WalletBuyInfo
	TotalBuyVolume   decimal.Decimal
	SmartWalletCount int
	FirstBuyTime     time.Time
	LastBuyTime      time.Time
}

// WalletBuyInfo 钱包买入信息。
type WalletBuyInfo struct {
	WalletAddress string
	WalletScore   decimal.Decimal
	Trades        []model.WalletTrade
	TotalVolume   decimal.Decimal
}

// calculateConsensusScore 计算共识度评分（0-100）。
func (s *Service) calculateConsensusScore(agg *TokenAggregation, totalTopWallets int) decimal.Decimal {
	score := decimal.Zero
	
	// 1. 买入钱包数量占比（0-40分）
	// 例如：10个钱包买入/20个Top钱包 = 50% → 20分
	walletRatio := float64(agg.SmartWalletCount) / float64(totalTopWallets)
	walletScore := decimal.NewFromFloat(walletRatio * 40)
	score = score.Add(walletScore)
	
	// 2. 总买入量（0-30分）
	// > 100,000 USD: 30分
	// > 50,000 USD: 20分
	// > 10,000 USD: 10分
	// > 5,000 USD: 5分
	volumeScore := decimal.Zero
	if agg.TotalBuyVolume.GreaterThan(decimal.NewFromInt(100000)) {
		volumeScore = decimal.NewFromInt(30)
	} else if agg.TotalBuyVolume.GreaterThan(decimal.NewFromInt(50000)) {
		volumeScore = decimal.NewFromInt(20)
	} else if agg.TotalBuyVolume.GreaterThan(decimal.NewFromInt(10000)) {
		volumeScore = decimal.NewFromInt(10)
	} else if agg.TotalBuyVolume.GreaterThan(decimal.NewFromInt(5000)) {
		volumeScore = decimal.NewFromInt(5)
	}
	score = score.Add(volumeScore)
	
	// 3. 高分钱包权重（0-20分）
	// 计算买入钱包的平均分数
	totalScore := decimal.Zero
	for _, walletBuy := range agg.WalletBuys {
		totalScore = totalScore.Add(walletBuy.WalletScore)
	}
	avgWalletScore := totalScore.Div(decimal.NewFromInt(int64(agg.SmartWalletCount)))
	
	// 平均分 > 90: 20分
	// 平均分 > 80: 15分
	// 平均分 > 70: 10分
	// 平均分 > 60: 5分
	walletQualityScore := decimal.Zero
	if avgWalletScore.GreaterThan(decimal.NewFromInt(90)) {
		walletQualityScore = decimal.NewFromInt(20)
	} else if avgWalletScore.GreaterThan(decimal.NewFromInt(80)) {
		walletQualityScore = decimal.NewFromInt(15)
	} else if avgWalletScore.GreaterThan(decimal.NewFromInt(70)) {
		walletQualityScore = decimal.NewFromInt(10)
	} else if avgWalletScore.GreaterThan(decimal.NewFromInt(60)) {
		walletQualityScore = decimal.NewFromInt(5)
	}
	score = score.Add(walletQualityScore)
	
	// 4. 时间集中度（0-10分）
	// 如果买入时间集中在短时间内，说明共识度更高
	if !agg.FirstBuyTime.IsZero() && !agg.LastBuyTime.IsZero() {
		timeSpan := agg.LastBuyTime.Sub(agg.FirstBuyTime).Hours()
		
		// 在24小时内: 10分
		// 在48小时内: 5分
		// 在72小时内: 2分
		timeScore := decimal.Zero
		if timeSpan <= 24 {
			timeScore = decimal.NewFromInt(10)
		} else if timeSpan <= 48 {
			timeScore = decimal.NewFromInt(5)
		} else if timeSpan <= 72 {
			timeScore = decimal.NewFromInt(2)
		}
		score = score.Add(timeScore)
	}
	
	// 确保得分在 0-100 之间
	if score.LessThan(decimal.Zero) {
		score = decimal.Zero
	}
	if score.GreaterThan(decimal.NewFromInt(100)) {
		score = decimal.NewFromInt(100)
	}
	
	return score
}

// saveSignalDetails 保存信号详情。
func (s *Service) saveSignalDetails(signalID uint64, agg *TokenAggregation) error {
	// 先删除旧的详情
	s.store.DB().Where("signal_id = ?", signalID).Delete(&model.TokenSignalDetail{})
	
	// 插入新的详情
	for _, walletBuy := range agg.WalletBuys {
		for _, trade := range walletBuy.Trades {
			detail := &model.TokenSignalDetail{
				SignalID:      signalID,
				WalletAddress: walletBuy.WalletAddress,
				WalletScore:   walletBuy.WalletScore,
				TradeID:       trade.ID,
				BuyAmountUSD:  trade.AmountUSD,
				BuyTime:       trade.BlockTime,
			}
			
			if err := s.store.DB().Create(detail).Error; err != nil {
				return fmt.Errorf("create signal detail: %w", err)
			}
		}
	}
	
	return nil
}

// GetTopSignals 获取共识度最高的代币信号。
func (s *Service) GetTopSignals(limit int, minConsensusScore float64) ([]model.TokenSignal, error) {
	var signals []model.TokenSignal
	
	query := s.store.DB().Where("chain_id = ?", s.chainID)
	
	if minConsensusScore > 0 {
		query = query.Where("consensus_score >= ?", minConsensusScore)
	}
	
	// 获取最新周期的信号
	var latestStartDate time.Time
	s.store.DB().Model(&model.TokenSignal{}).
		Where("chain_id = ?", s.chainID).
		Select("MAX(signal_start_date)").
		Scan(&latestStartDate)
	
	if !latestStartDate.IsZero() {
		query = query.Where("signal_start_date = ?", latestStartDate)
	}
	
	err := query.Order("consensus_score DESC, smart_wallet_count DESC").
		Limit(limit).
		Find(&signals).Error
	
	if err != nil {
		return nil, fmt.Errorf("fetch top signals: %w", err)
	}
	
	return signals, nil
}

// GetSignalDetails 获取代币信号的详细信息。
func (s *Service) GetSignalDetails(signalID uint64) ([]model.TokenSignalDetail, error) {
	var details []model.TokenSignalDetail
	
	err := s.store.DB().Where("signal_id = ?", signalID).
		Order("wallet_score DESC, buy_time DESC").
		Find(&details).Error
	
	if err != nil {
		return nil, fmt.Errorf("fetch signal details: %w", err)
	}
	
	return details, nil
}
