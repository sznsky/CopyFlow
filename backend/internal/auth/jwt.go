package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 载荷，包含用户 ID 与钱包地址。
type Claims struct {
	UserID        uint64 `json:"user_id"`
	WalletAddress string `json:"wallet_address"`
	jwt.RegisteredClaims
}

// JWTManager 负责签发与解析 JWT。
type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTManager 创建 JWT 管理器。
func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), ttl: ttl}
}

// Issue 签发 JWT Token。
func (m *JWTManager) Issue(userID uint64, walletAddress string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:        userID,
		WalletAddress: walletAddress,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse 解析并校验 JWT Token。
func (m *JWTManager) Parse(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
