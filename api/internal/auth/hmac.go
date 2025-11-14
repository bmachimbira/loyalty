package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"time"
)

var (
	ErrMissingHMACHeaders = errors.New("missing required HMAC headers")
	ErrInvalidTimestamp   = errors.New("timestamp outside valid window")
	ErrInvalidSignature   = errors.New("invalid HMAC signature")
	ErrKeyNotFound        = errors.New("API key not found")
)

// HMACKey represents an API key with its secret
type HMACKey struct {
	Key    string `json:"key"`
	Secret string `json:"secret"`
}

// HMACKeys holds the map of API keys
type HMACKeys map[string]string // key -> secret

// LoadHMACKeys loads API keys from the HMAC_KEYS_JSON environment variable
func LoadHMACKeys() (HMACKeys, error) {
	keysJSON := os.Getenv("HMAC_KEYS_JSON")
	if keysJSON == "" {
		return HMACKeys{}, nil // No keys configured
	}

	var keys []HMACKey
	if err := json.Unmarshal([]byte(keysJSON), &keys); err != nil {
		return nil, err
	}

	hmacKeys := make(HMACKeys)
	for _, k := range keys {
		hmacKeys[k.Key] = k.Secret
	}

	return hmacKeys, nil
}

// ValidateHMAC verifies the HMAC signature from request headers
// Expected headers: X-Key, X-Timestamp, X-Signature
// Signature = HMAC-SHA256(secret, X-Timestamp + request_body)
func ValidateHMAC(apiKey, timestamp, signature, body string, keys HMACKeys) error {
	// Check if all required headers are present
	if apiKey == "" || timestamp == "" || signature == "" {
		return ErrMissingHMACHeaders
	}

	// Verify timestamp is within 5 minutes
	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return ErrInvalidTimestamp
	}

	now := time.Now()
	diff := now.Sub(ts)
	if diff < -5*time.Minute || diff > 5*time.Minute {
		return ErrInvalidTimestamp
	}

	// Look up the secret for this key
	secret, ok := keys[apiKey]
	if !ok {
		return ErrKeyNotFound
	}

	// Compute HMAC signature
	computed := ComputeHMAC(secret, timestamp, body)

	// Compare signatures securely
	if !hmac.Equal([]byte(signature), []byte(computed)) {
		return ErrInvalidSignature
	}

	return nil
}

// ComputeHMAC computes the HMAC-SHA256 signature
func ComputeHMAC(secret, timestamp, body string) string {
	message := timestamp + body
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
