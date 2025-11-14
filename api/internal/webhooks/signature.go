package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateSignature generates an HMAC-SHA256 signature for webhook payloads
func GenerateSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies an HMAC-SHA256 signature for webhook payloads
func VerifySignature(secret string, payload []byte, signature string) bool {
	expectedSignature := GenerateSignature(secret, payload)
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}
