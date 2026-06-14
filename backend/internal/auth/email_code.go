package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateEmailCode 生成 6 位数字验证码。
func GenerateEmailCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
