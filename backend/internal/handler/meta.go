package handler

import (
	"net/http"

	"copyflow/internal/config"

	"github.com/gin-gonic/gin"
)

// MetaHandler 元信息接口（链、DEX 列表等）。
type MetaHandler struct {
	cfg *config.Config
}

// NewMetaHandler 创建元信息处理器。
func NewMetaHandler(cfg *config.Config) *MetaHandler {
	return &MetaHandler{cfg: cfg}
}

// Chains 返回已启用的链与 DEX，供前端下拉选择。
func (h *MetaHandler) Chains(c *gin.Context) {
	type dexInfo struct {
		Type    string `json:"type"`
		Enabled bool   `json:"enabled"`
	}
	type chainInfo struct {
		ChainID      int       `json:"chain_id"`
		Name         string    `json:"name"`
		NativeSymbol string    `json:"native_symbol"`
		Enabled      bool      `json:"enabled"`
		DEXes        []dexInfo `json:"dexes"`
	}
	var out []chainInfo
	for _, ch := range h.cfg.Chains {
		ci := chainInfo{
			ChainID:      ch.ChainID,
			Name:         ch.Name,
			NativeSymbol: ch.NativeSymbol,
			Enabled:      ch.Enabled,
		}
		for _, d := range ch.DEXes {
			ci.DEXes = append(ci.DEXes, dexInfo{Type: d.Type, Enabled: d.Enabled})
		}
		out = append(out, ci)
	}
	c.JSON(http.StatusOK, out)
}

// Health 健康检查接口。
func (h *MetaHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
