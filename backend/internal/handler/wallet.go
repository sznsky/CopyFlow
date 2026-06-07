package handler

import (
	"encoding/hex"
	"net/http"

	"copyflow/internal/middleware"
	"copyflow/internal/model"
	"copyflow/internal/store"
	walletcrypto "copyflow/pkg/crypto"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
)

// WalletHandler 跟单钱包管理。
type WalletHandler struct {
	store      *store.Store
	encryptKey string
}

// NewWalletHandler 创建跟单钱包处理器。
func NewWalletHandler(st *store.Store, encryptKey string) *WalletHandler {
	return &WalletHandler{store: st, encryptKey: encryptKey}
}

type createWalletRequest struct {
	ChainID int `json:"chain_id" binding:"required"`
}

// List 列出用户的跟单钱包。
func (h *WalletHandler) List(c *gin.Context) {
	userID := middleware.UserID(c)
	list, err := h.store.ListCopyWallets(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Create 为用户在指定链上生成跟单钱包。
// 扩展点：导入已有钱包、每链多钱包。
func (h *WalletHandler) Create(c *gin.Context) {
	userID := middleware.UserID(c)
	var req createWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	exists, err := h.store.HasCopyWallet(userID, req.ChainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "该链已存在跟单钱包"})
		return
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}
	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	pkHex := hex.EncodeToString(crypto.FromECDSA(privateKey))
	encrypted, err := walletcrypto.EncryptPrivateKey(pkHex, h.encryptKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	w := &model.CopyWallet{
		UserID:              userID,
		ChainID:             req.ChainID,
		Address:             address,
		EncryptedPrivateKey: encrypted,
		IsActive:            true,
	}
	if err := h.store.CreateCopyWallet(w); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":       w.ID,
		"chain_id": w.ChainID,
		"address":  w.Address,
		"message":  "Deposit BNB/ETH to this address for copy trading",
	})
}
