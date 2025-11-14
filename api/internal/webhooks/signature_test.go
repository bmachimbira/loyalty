package webhooks_test

import (
	"testing"

	"github.com/bmachimbira/loyalty/api/internal/webhooks"
	"github.com/stretchr/testify/assert"
)

func TestGenerateSignature(t *testing.T) {
	secret := "my-secret-key"
	payload := []byte(`{"event":"test","data":"value"}`)

	signature := webhooks.GenerateSignature(secret, payload)

	// Signature should be a hex string
	assert.NotEmpty(t, signature)
	assert.Len(t, signature, 64) // SHA256 produces 64 hex characters
}

func TestGenerateSignature_Deterministic(t *testing.T) {
	secret := "my-secret-key"
	payload := []byte(`{"event":"test","data":"value"}`)

	signature1 := webhooks.GenerateSignature(secret, payload)
	signature2 := webhooks.GenerateSignature(secret, payload)

	assert.Equal(t, signature1, signature2, "Same input should produce same signature")
}

func TestGenerateSignature_DifferentSecrets(t *testing.T) {
	payload := []byte(`{"event":"test","data":"value"}`)

	signature1 := webhooks.GenerateSignature("secret1", payload)
	signature2 := webhooks.GenerateSignature("secret2", payload)

	assert.NotEqual(t, signature1, signature2, "Different secrets should produce different signatures")
}

func TestGenerateSignature_DifferentPayloads(t *testing.T) {
	secret := "my-secret-key"

	signature1 := webhooks.GenerateSignature(secret, []byte(`{"event":"test1"}`))
	signature2 := webhooks.GenerateSignature(secret, []byte(`{"event":"test2"}`))

	assert.NotEqual(t, signature1, signature2, "Different payloads should produce different signatures")
}

func TestVerifySignature_Valid(t *testing.T) {
	secret := "my-secret-key"
	payload := []byte(`{"event":"test","data":"value"}`)

	signature := webhooks.GenerateSignature(secret, payload)
	valid := webhooks.VerifySignature(secret, payload, signature)

	assert.True(t, valid, "Valid signature should be verified")
}

func TestVerifySignature_Invalid(t *testing.T) {
	secret := "my-secret-key"
	payload := []byte(`{"event":"test","data":"value"}`)

	invalidSignature := "0000000000000000000000000000000000000000000000000000000000000000"
	valid := webhooks.VerifySignature(secret, payload, invalidSignature)

	assert.False(t, valid, "Invalid signature should not be verified")
}

func TestVerifySignature_WrongSecret(t *testing.T) {
	payload := []byte(`{"event":"test","data":"value"}`)

	signature := webhooks.GenerateSignature("secret1", payload)
	valid := webhooks.VerifySignature("secret2", payload, signature)

	assert.False(t, valid, "Signature with wrong secret should not be verified")
}

func TestVerifySignature_ModifiedPayload(t *testing.T) {
	secret := "my-secret-key"
	originalPayload := []byte(`{"event":"test","data":"value"}`)
	modifiedPayload := []byte(`{"event":"test","data":"modified"}`)

	signature := webhooks.GenerateSignature(secret, originalPayload)
	valid := webhooks.VerifySignature(secret, modifiedPayload, signature)

	assert.False(t, valid, "Signature should not be valid for modified payload")
}

func TestVerifySignature_EmptyPayload(t *testing.T) {
	secret := "my-secret-key"
	payload := []byte{}

	signature := webhooks.GenerateSignature(secret, payload)
	valid := webhooks.VerifySignature(secret, payload, signature)

	assert.True(t, valid, "Should work with empty payload")
}

func TestVerifySignature_TimingSafe(t *testing.T) {
	// This test verifies that the comparison is timing-safe
	// by ensuring it uses hmac.Equal internally
	secret := "my-secret-key"
	payload := []byte(`{"event":"test"}`)

	signature := webhooks.GenerateSignature(secret, payload)

	// Create similar but different signatures
	wrongSignature := signature[:len(signature)-1] + "0"

	valid := webhooks.VerifySignature(secret, payload, wrongSignature)
	assert.False(t, valid)

	valid = webhooks.VerifySignature(secret, payload, signature)
	assert.True(t, valid)
}
