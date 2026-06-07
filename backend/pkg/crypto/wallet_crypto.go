// Package crypto 跟单钱包私钥加解密。
// 扩展点：生产环境可替换为 KMS/HSM。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// EncryptPrivateKey 使用 AES-256-GCM 加密 hex 私钥。
func EncryptPrivateKey(plaintext, key string) (string, error) {
	keyBytes := []byte(key)
	if len(keyBytes) != 32 {
		return "", errors.New("encryption key must be 32 bytes")
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptPrivateKey 解密跟单钱包私钥。
func DecryptPrivateKey(ciphertext, key string) (string, error) {
	keyBytes := []byte(key)
	if len(keyBytes) != 32 {
		return "", errors.New("encryption key must be 32 bytes")
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt failed: %w", err)
	}
	return string(plain), nil
}
