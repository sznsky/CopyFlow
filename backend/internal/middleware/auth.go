// Package middleware HTTP 中间件（鉴权、CORS 等）。
package middleware

import (
	"net/http"
	"strings"

	"copyflow/internal/auth"

	"github.com/gin-gonic/gin"
)

// Auth JWT 鉴权中间件，解析 Bearer Token。
func Auth(jwt *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		claims, err := jwt.Parse(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("walletAddress", claims.WalletAddress)
		c.Next()
	}
}
