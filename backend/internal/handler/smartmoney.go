// Package handler 提供聪明钱相关的 HTTP 处理器。
package handler

import (
	"net/http"
	"strconv"
	"strings"

	"copyflow/internal/model"
	"copyflow/internal/smartmoney"
	"copyflow/internal/store"

	"github.com/gin-gonic/gin"
)

// SmartMoneyHandler 聪明钱 API 处理器。
type SmartMoneyHandler struct {
	service *smartmoney.Service
}

// NewSmartMoneyHandler 创建聪明钱处理器实例。
func NewSmartMoneyHandler(st *store.Store, duneAPIKey string, chainID int) *SmartMoneyHandler {
	return &SmartMoneyHandler{
		service: smartmoney.NewService(st, duneAPIKey, chainID),
	}
}

// GetTopWalletsRequest 获取 Top 钱包请求。
type GetTopWalletsRequest struct {
	Limit    int     `form:"limit" binding:"omitempty,min=1,max=100"`
	MinScore float64 `form:"min_score" binding:"omitempty,min=0,max=100"`
}

// GetTopWallets 获取高分钱包列表。
// GET /api/smart-wallets
func (h *SmartMoneyHandler) GetTopWallets(c *gin.Context) {
	var req GetTopWalletsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 默认值
	if req.Limit == 0 {
		req.Limit = 20
	}
	if req.MinScore == 0 {
		req.MinScore = 60
	}
	
	wallets, err := h.service.GetTopWallets(req.Limit, req.MinScore)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"wallets": wallets,
		"count":   len(wallets),
	})
}

// GetTopSignalsRequest 获取代币信号请求。
type GetTopSignalsRequest struct {
	Limit              int     `form:"limit" binding:"omitempty,min=1,max=100"`
	MinConsensusScore  float64 `form:"min_consensus_score" binding:"omitempty,min=0,max=100"`
}

// GetTopSignals 获取当前代币信号。
// GET /api/token-signals
func (h *SmartMoneyHandler) GetTopSignals(c *gin.Context) {
	var req GetTopSignalsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 默认值
	if req.Limit == 0 {
		req.Limit = 20
	}
	if req.MinConsensusScore == 0 {
		req.MinConsensusScore = 50
	}
	
	signals, err := h.service.GetTopSignals(req.Limit, req.MinConsensusScore)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"signals": signals,
		"count":   len(signals),
	})
}

// GetSignalDetails 获取代币信号详情。
// GET /api/token-signals/:id/details
func (h *SmartMoneyHandler) GetSignalDetails(c *gin.Context) {
	signalIDStr := c.Param("id")
	signalID, err := strconv.ParseUint(signalIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signal ID"})
		return
	}
	
	details, err := h.service.GetSignalDetails(signalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"details": details,
		"count":   len(details),
	})
}

// GetWalletHistory 获取钱包的交易历史。
// GET /api/wallet-history/:address
func (h *SmartMoneyHandler) GetWalletHistory(c *gin.Context) {
	walletAddr := c.Param("address")
	
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	
	var trades []model.WalletTrade
	err = h.service.Store().DB().
		Where("wallet_address = ?", strings.ToLower(walletAddr)).
		Order("block_time DESC").
		Limit(limit).
		Find(&trades).Error
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"wallet_address": walletAddr,
		"trades":         trades,
		"count":          len(trades),
	})
}

// TriggerSync 手动触发数据同步（管理员接口）。
// POST /api/admin/sync
func (h *SmartMoneyHandler) TriggerSync(c *gin.Context) {
	type SyncRequest struct {
		QueryID      int     `json:"query_id" binding:"required"`
		MinAmountUSD float64 `json:"min_amount_usd" binding:"required,min=0"`
		DaysBack     int     `json:"days_back" binding:"omitempty,min=1"`
	}
	
	var req SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 默认值：180天（6个月）
	if req.DaysBack == 0 {
		req.DaysBack = 180
	}
	
	// 异步执行同步
	go func() {
		if err := h.service.SyncTradesFromDune(req.QueryID, req.MinAmountUSD, req.DaysBack); err != nil {
			// 日志已在 service 中记录
		}
	}()
	
	c.JSON(http.StatusOK, gin.H{"message": "sync started"})
}

// TriggerScoring 手动触发评分计算（管理员接口）。
// POST /api/admin/calculate-scores
func (h *SmartMoneyHandler) TriggerScoring(c *gin.Context) {
	// 异步执行评分
	go func() {
		// 先计算盈亏
		if err := h.service.CalculatePNLForTrades(); err != nil {
			// 日志已在 service 中记录
		}
		
		// 再计算评分
		if err := h.service.CalculateWalletScores(); err != nil {
			// 日志已在 service 中记录
		}
	}()
	
	c.JSON(http.StatusOK, gin.H{"message": "scoring started"})
}

// TriggerSignalAggregation 手动触发信号聚合（管理员接口）。
// POST /api/admin/aggregate-signals
func (h *SmartMoneyHandler) TriggerSignalAggregation(c *gin.Context) {
	type AggregateRequest struct {
		Days int `json:"days" binding:"required,min=1,max=30"`
	}
	
	var req AggregateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 异步执行聚合
	go func() {
		if err := h.service.AggregateTokenSignals(req.Days); err != nil {
			// 日志已在 service 中记录
		}
	}()
	
	c.JSON(http.StatusOK, gin.H{"message": "signal aggregation started"})
}
