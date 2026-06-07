package middleware

import "github.com/gin-gonic/gin"

// UserID 从 gin 上下文获取当前登录用户 ID。
func UserID(c *gin.Context) uint64 {
	v, ok := c.Get("userID")
	if !ok {
		return 0
	}
	id, ok := v.(uint64)
	if !ok {
		return 0
	}
	return id
}
