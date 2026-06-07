// Package auth 提供 SIWE 钱包签名登录与 JWT 鉴权。
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// BuildSIWEMessage 构造待签名的 SIWE 登录消息。
func BuildSIWEMessage(domain, address, nonce string, issuedAt time.Time) string {
	return fmt.Sprintf(`%s wants you to sign in with your Ethereum account:
%s

Sign in to CopyFlow

URI: %s
Version: 1
Chain ID: 1
Nonce: %s
Issued At: %s`,
		domain,
		common.HexToAddress(address).Hex(),
		domain,
		nonce,
		issuedAt.UTC().Format(time.RFC3339),
	)
}

// GenerateNonce 生成随机 nonce，防重放攻击。
func GenerateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// VerifySIWESignature 验证 EIP-191 personal_sign 签名。
func VerifySIWESignature(message, signature, expectedAddress string) (bool, error) {
	sig := strings.TrimPrefix(signature, "0x")
	sigBytes, err := hex.DecodeString(sig)
	if err != nil {
		return false, err
	}
	if len(sigBytes) != 65 {
		return false, fmt.Errorf("invalid signature length")
	}
	// Adjust V value for recovery
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}
	hash := crypto.Keccak256Hash([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)))
	pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		return false, err
	}
	recovered := crypto.PubkeyToAddress(*pubKey)
	expected := common.HexToAddress(expectedAddress)
	return strings.EqualFold(recovered.Hex(), expected.Hex()), nil
}

// VerifyPersonalSign 验证 personal_sign 签名（Handler 便捷封装）。
func VerifyPersonalSign(message, signature, address string) bool {
	ok, err := VerifySIWESignature(message, signature, address)
	if err != nil {
		return false
	}
	return ok
}

// HashMessage 计算签名消息的 Keccak 哈希（调试用）。
func HashMessage(message string) common.Hash {
	return crypto.Keccak256Hash([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)))
}
