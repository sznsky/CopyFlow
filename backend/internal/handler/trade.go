package handler

import (
	"net/http"
	"strconv"

	"copyflow/internal/middleware"
	"copyflow/internal/store"

	"github.com/gin-gonic/gin"
)

// TradeHandler 领头/跟单交易记录查询。
type TradeHandler struct {
	store *store.Store
}

// NewTradeHandler 创建交易记录处理器。
func NewTradeHandler(st *store.Store) *TradeHandler {
	return &TradeHandler{store: st}
}

// ListCopyTrades 查询当前用户的跟单记录。
func (h *TradeHandler) ListCopyTrades(c *gin.Context) {
	userID := middleware.UserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	list, err := h.store.ListCopyTrades(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// ListLeaderTrades 查询监听到的领头交易（全局）。
func (h *TradeHandler) ListLeaderTrades(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	list, err := h.store.ListLeaderTrades(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// parseUint64 将 URL 路径参数转为 uint64。
func parseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}
