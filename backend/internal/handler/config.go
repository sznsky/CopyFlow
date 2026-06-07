package handler

import (
	"net/http"
	"strings"

	"copyflow/internal/middleware"
	"copyflow/internal/model"
	"copyflow/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// ConfigHandler 跟单配置 CRUD。
type ConfigHandler struct {
	store *store.Store
}

// NewConfigHandler 创建跟单配置处理器。
func NewConfigHandler(st *store.Store) *ConfigHandler {
	return &ConfigHandler{store: st}
}

type createConfigRequest struct {
	ChainID       int     `json:"chain_id" binding:"required"`
	DEXType       string  `json:"dex_type" binding:"required"`
	LeaderAddress string  `json:"leader_address" binding:"required"`
	CopyMode      string  `json:"copy_mode"`
	CopyAmount    float64 `json:"copy_amount" binding:"required"`
	MaxPerTrade   float64 `json:"max_per_trade"`
	SlippageBps   int     `json:"slippage_bps"`
	IsActive      *bool   `json:"is_active"`
}

// List 列出当前用户的跟单配置。
func (h *ConfigHandler) List(c *gin.Context) {
	userID := middleware.UserID(c)
	list, err := h.store.ListCopyConfigs(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Create 新增跟单配置。
func (h *ConfigHandler) Create(c *gin.Context) {
	userID := middleware.UserID(c)
	var req createConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	copyMode := req.CopyMode
	if copyMode == "" {
		copyMode = model.CopyModeRatio
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	cfg := &model.CopyConfig{
		UserID:        userID,
		ChainID:       req.ChainID,
		DEXType:       req.DEXType,
		LeaderAddress: strings.ToLower(req.LeaderAddress),
		CopyMode:      copyMode,
		CopyAmount:    decimal.NewFromFloat(req.CopyAmount),
		MaxPerTrade:   decimal.NewFromFloat(req.MaxPerTrade),
		SlippageBps:   req.SlippageBps,
		IsActive:      active,
	}
	if cfg.SlippageBps == 0 {
		cfg.SlippageBps = 300
	}
	if err := h.store.CreateCopyConfig(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cfg)
}

// Update 更新跟单配置。
func (h *ConfigHandler) Update(c *gin.Context) {
	userID := middleware.UserID(c)
	id := parseUint64(c.Param("id"))
	existing, err := h.store.GetCopyConfig(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}
	var req createConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChainID != 0 {
		existing.ChainID = req.ChainID
	}
	if req.DEXType != "" {
		existing.DEXType = req.DEXType
	}
	if req.LeaderAddress != "" {
		existing.LeaderAddress = strings.ToLower(req.LeaderAddress)
	}
	if req.CopyMode != "" {
		existing.CopyMode = req.CopyMode
	}
	if req.CopyAmount != 0 {
		existing.CopyAmount = decimal.NewFromFloat(req.CopyAmount)
	}
	if req.MaxPerTrade != 0 {
		existing.MaxPerTrade = decimal.NewFromFloat(req.MaxPerTrade)
	}
	if req.SlippageBps != 0 {
		existing.SlippageBps = req.SlippageBps
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if err := h.store.UpdateCopyConfig(userID, existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

// Delete 删除跟单配置。
func (h *ConfigHandler) Delete(c *gin.Context) {
	userID := middleware.UserID(c)
	id := parseUint64(c.Param("id"))
	if err := h.store.DeleteCopyConfig(userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
