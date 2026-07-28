package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const listing1688CookieKeyEnv = "LISTING_1688_COOKIE_KEY"

// Listing1688LoadCookieKey 读取 Base64 编码的 32 字节密钥（C12）
func Listing1688LoadCookieKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(listing1688CookieKeyEnv))
	if raw == "" {
		return nil, fmt.Errorf("%s 未配置", listing1688CookieKeyEnv)
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("%s 须为 Base64: %w", listing1688CookieKeyEnv, err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("%s 解码后须为 32 字节，当前 %d", listing1688CookieKeyEnv, len(key))
	}
	return key, nil
}

// Listing1688EncryptCookie AES-GCM 加密 Cookie；密文 = base64(nonce|ciphertext)
func Listing1688EncryptCookie(plain string) (string, error) {
	key, err := Listing1688LoadCookieKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
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
	out := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Listing1688DecryptCookie 解密 Cookie 明文
func Listing1688DecryptCookie(enc string) (string, error) {
	key, err := Listing1688LoadCookieKey()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(enc))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("密文过短")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// Listing1688DevCookieKeyBase64 仅用于本地单测生成合法密钥（勿写入生产配置提交）
func Listing1688DevCookieKeyBase64() string {
	key := make([]byte, 32)
	copy(key, []byte("listing1688-dev-cookie-key-32b!!"))
	return base64.StdEncoding.EncodeToString(key)
}
