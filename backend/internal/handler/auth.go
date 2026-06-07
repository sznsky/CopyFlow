// Package handler HTTP API 请求处理器。
package handler

import (
	"net/http"
	"strings"
	"time"

	"copyflow/internal/auth"
	"copyflow/internal/middleware"
	"copyflow/internal/store"

	"github.com/gin-gonic/gin"
)

// AuthHandler 钱包签名登录相关接口。
type AuthHandler struct {
	store  *store.Store
	jwt    *auth.JWTManager
	domain string
}

// NewAuthHandler 创建认证处理器。
func NewAuthHandler(st *store.Store, jwt *auth.JWTManager, domain string) *AuthHandler {
	return &AuthHandler{store: st, jwt: jwt, domain: domain}
}

type nonceRequest struct {
	Address string `json:"address" binding:"required"`
}

type nonceResponse struct {
	Nonce   string `json:"nonce"`
	Message string `json:"message"`
}

// Nonce 获取登录 nonce 与待签名消息。
func (h *AuthHandler) Nonce(c *gin.Context) {
	var req nonceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	nonce, err := auth.GenerateNonce()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate nonce"})
		return
	}
	addr := strings.ToLower(req.Address)
	user, err := h.store.UpsertUserNonce(addr, nonce)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	msg := auth.BuildSIWEMessage(h.domain, user.WalletAddress, nonce, time.Now())
	c.JSON(http.StatusOK, nonceResponse{Nonce: nonce, Message: msg})
}

type verifyRequest struct {
	Address   string `json:"address" binding:"required"`
	Message   string `json:"message" binding:"required"`
	Signature string `json:"signature" binding:"required"`
}

type verifyResponse struct {
	Token string `json:"token"`
}

// Verify 验证钱包签名并返回 JWT。
func (h *AuthHandler) Verify(c *gin.Context) {
	var req verifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	addr := strings.ToLower(req.Address)
	user, err := h.store.GetUserByAddress(addr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found, request nonce first"})
		return
	}
	if !auth.VerifyPersonalSign(req.Message, req.Signature, addr) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}
	if !strings.Contains(req.Message, user.Nonce) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "nonce mismatch"})
		return
	}
	newNonce, _ := auth.GenerateNonce()
	_, _ = h.store.UpsertUserNonce(addr, newNonce)

	token, err := h.jwt.Issue(user.ID, user.WalletAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, verifyResponse{Token: token})
}

// Me 返回当前登录用户信息。
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.UserID(c)
	user, err := h.store.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":             user.ID,
		"wallet_address": user.WalletAddress,
	})
}
