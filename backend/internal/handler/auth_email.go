package handler

import (
	"net/http"
	"strings"
	"time"

	"copyflow/internal/auth"
	"copyflow/internal/model"
	"copyflow/internal/store"
	"copyflow/pkg/email"

	"github.com/gin-gonic/gin"
)

// AuthEmailHandler 邮箱注册登录。
type AuthEmailHandler struct {
	store  *store.Store
	jwt    *auth.JWTManager
	mailer *email.Sender
}

// NewAuthEmailHandler 创建邮箱认证处理器。
func NewAuthEmailHandler(st *store.Store, jwt *auth.JWTManager, mailer *email.Sender) *AuthEmailHandler {
	return &AuthEmailHandler{store: st, jwt: jwt, mailer: mailer}
}

type sendCodeRequest struct {
	Email   string `json:"email" binding:"required,email"`
	Purpose string `json:"purpose"` // register（默认）
}

// SendEmailCode 发送邮箱验证码。
func (h *AuthEmailHandler) SendEmailCode(c *gin.Context) {
	var req sendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入有效邮箱"})
		return
	}
	emailAddr := strings.ToLower(strings.TrimSpace(req.Email))
	purpose := req.Purpose
	if purpose == "" {
		purpose = "register"
	}

	if purpose == "register" {
		if _, err := h.store.GetUserByEmail(emailAddr); err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已注册"})
			return
		}
	}

	code, err := auth.GenerateEmailCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成验证码失败"})
		return
	}

	v := &model.EmailVerification{
		Email:     emailAddr,
		Code:      code,
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := h.store.CreateEmailVerification(v); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.mailer.SendVerificationCode(emailAddr, code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送邮件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "验证码已发送"})
}

type emailRegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required,len=6"`
	Password string `json:"password" binding:"required,min=6"`
}

// EmailRegister 邮箱验证码注册。
func (h *AuthEmailHandler) EmailRegister(c *gin.Context) {
	var req emailRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写邮箱、6位验证码和密码（至少6位）"})
		return
	}
	emailAddr := strings.ToLower(strings.TrimSpace(req.Email))

	v, err := h.store.FindValidEmailCode(emailAddr, req.Code, "register")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码无效或已过期"})
		return
	}

	if _, err := h.store.GetUserByEmail(emailAddr); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已注册"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	user, err := h.store.CreateEmailUser(emailAddr, hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.MarkEmailCodeUsed(v.ID)

	token, err := h.jwt.Issue(user.ID, ptrStr(user.WalletAddress), ptrStr(user.Email))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签发 token 失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

type emailLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// EmailLogin 邮箱密码登录。
func (h *AuthEmailHandler) EmailLogin(c *gin.Context) {
	var req emailLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	emailAddr := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := h.store.GetUserByEmail(emailAddr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}

	token, err := h.jwt.Issue(user.ID, ptrStr(user.WalletAddress), ptrStr(user.Email))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签发 token 失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}
