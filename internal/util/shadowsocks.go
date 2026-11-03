package util

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// ShadowsocksKeySize returns 0 for legacy AEAD (arbitrary password) and
// the byte length required for the three AEAD-2022 methods.
func ShadowsocksKeySize(method string) (int, error) {
	switch method {
	case "aes-128-gcm", "aes-256-gcm", "chacha20-ietf-poly1305", "xchacha20-ietf-poly1305":
		return 0, nil
	case "2022-blake3-aes-128-gcm":
		return 16, nil
	case "2022-blake3-aes-256-gcm":
		return 32, nil
	case "2022-blake3-chacha20-poly1305":
		return 0, fmt.Errorf("当前固定版本 sing-box 1.13.12 的多用户入站无法加载 SS2022 ChaCha20，请使用 SS2022 AES-GCM")
	default:
		return 0, fmt.Errorf("不支持 Shadowsocks 加密方式: %s", method)
	}
}

func ValidateShadowsocksPassword(method, password string) error {
	length, err := ShadowsocksKeySize(method)
	if err != nil {
		return err
	}
	if password == "" {
		return fmt.Errorf("Shadowsocks 密钥不能为空")
	}
	if length == 0 {
		return nil
	}
	key, err := base64.StdEncoding.Strict().DecodeString(password)
	if err != nil || len(key) != length {
		return fmt.Errorf("%s 需要 %d 字节的 Base64 密钥", method, length)
	}
	return nil
}

func GenerateShadowsocksPassword(method string) (string, error) {
	length, err := ShadowsocksKeySize(method)
	if err != nil {
		return "", err
	}
	if length == 0 {
		length = 24
	}
	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// The same per-user credentials must be used by the core and every exporter.
// The inbound secret is the HMAC key; the user's token is stable across
// subscriptions, and does not modify passwords used by other protocols.
func ShadowsocksUserPassword(method, serverSecret, userToken string) string {
	length, _ := ShadowsocksKeySize(method)
	h := hmac.New(sha256.New, []byte(serverSecret))
	h.Write([]byte("slinx-ef/shadowsocks-user/v1/"))
	h.Write([]byte(userToken))
	digest := h.Sum(nil)
	if length == 0 {
		return base64.RawURLEncoding.EncodeToString(digest[:24])
	}
	return base64.StdEncoding.EncodeToString(digest[:length])
}

func ShadowsocksClientPassword(method, serverSecret, userToken string) string {
	userKey := ShadowsocksUserPassword(method, serverSecret, userToken)
	if strings.HasPrefix(method, "2022-") {
		return serverSecret + ":" + userKey
	}
	return userKey
}
