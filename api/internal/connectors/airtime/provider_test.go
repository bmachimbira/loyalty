package airtime_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/connectors"
	"github.com/bmachimbira/loyalty/api/internal/connectors/airtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_IssueVoucher_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/issue", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.NotEmpty(t, r.Header.Get("X-API-Key"))
		assert.NotEmpty(t, r.Header.Get("X-Signature"))

		// Return success response
		response := map[string]string{
			"voucher_code":   "ABC123XYZ",
			"transaction_id": "TX123456",
			"status":         "success",
			"message":        "Voucher issued successfully",
			"external_ref":   "EXT-REF-123",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider
	provider := airtime.New(server.URL, "test-key", "test-secret", 30*time.Second)

	// Issue voucher
	ctx := context.Background()
	params := connectors.IssueParams{
		ProductID:   "DATA_200MB",
		CustomerID:  "customer-123",
		PhoneNumber: "+263771234567",
		Amount:      5.00,
		Currency:    "USD",
		Reference:   "ref-123",
	}

	resp, err := provider.IssueVoucher(ctx, params)

	// Assertions
	require.NoError(t, err)
	assert.Equal(t, "ABC123XYZ", resp.VoucherCode)
	assert.Equal(t, "TX123456", resp.TransactionID)
	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, "Voucher issued successfully", resp.Message)
	assert.Equal(t, "EXT-REF-123", resp.ExternalRef)
}

func TestProvider_IssueVoucher_ProviderError(t *testing.T) {
	// Mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid product ID"}`))
	}))
	defer server.Close()

	provider := airtime.New(server.URL, "test-key", "test-secret", 30*time.Second)

	ctx := context.Background()
	params := connectors.IssueParams{
		ProductID:   "INVALID",
		PhoneNumber: "+263771234567",
		Amount:      5.00,
		Currency:    "USD",
		Reference:   "ref-123",
	}

	_, err := provider.IssueVoucher(ctx, params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 400")
}

func TestProvider_IssueVoucher_Timeout(t *testing.T) {
	// Mock server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create provider with short timeout
	provider := airtime.New(server.URL, "test-key", "test-secret", 500*time.Millisecond)

	ctx := context.Background()
	params := connectors.IssueParams{
		ProductID:   "DATA_200MB",
		PhoneNumber: "+263771234567",
		Amount:      5.00,
		Currency:    "USD",
		Reference:   "ref-123",
	}

	_, err := provider.IssueVoucher(ctx, params)

	require.Error(t, err)
	// The error will be about the request failing (timeout in the HTTP client)
	assert.Contains(t, err.Error(), "failed after retries")
}

func TestProvider_IssueVoucher_RetryOn5xx(t *testing.T) {
	attemptCount := 0

	// Mock server that fails twice then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Success on third attempt
		response := map[string]string{
			"voucher_code":   "ABC123XYZ",
			"transaction_id": "TX123456",
			"status":         "success",
			"message":        "Voucher issued successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := airtime.New(server.URL, "test-key", "test-secret", 30*time.Second)

	ctx := context.Background()
	params := connectors.IssueParams{
		ProductID:   "DATA_200MB",
		PhoneNumber: "+263771234567",
		Amount:      5.00,
		Currency:    "USD",
		Reference:   "ref-123",
	}

	resp, err := provider.IssueVoucher(ctx, params)

	require.NoError(t, err)
	assert.Equal(t, "ABC123XYZ", resp.VoucherCode)
	assert.Equal(t, 3, attemptCount, "Should have retried twice before succeeding")
}

func TestProvider_CheckStatus_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/status/")

		response := map[string]string{
			"status":         "completed",
			"message":        "Voucher delivered",
			"transaction_id": "TX123456",
			"updated_at":     "2025-11-14T10:00:00Z",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := airtime.New(server.URL, "test-key", "test-secret", 30*time.Second)

	ctx := context.Background()
	resp, err := provider.CheckStatus(ctx, "TX123456")

	require.NoError(t, err)
	assert.Equal(t, "completed", resp.Status)
	assert.Equal(t, "Voucher delivered", resp.Message)
	assert.Equal(t, "TX123456", resp.TransactionID)
}

func TestProvider_CancelVoucher_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/cancel", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "cancelled"}`))
	}))
	defer server.Close()

	provider := airtime.New(server.URL, "test-key", "test-secret", 30*time.Second)

	ctx := context.Background()
	err := provider.CancelVoucher(ctx, "EXT-REF-123")

	require.NoError(t, err)
}

func TestProvider_Name(t *testing.T) {
	provider := airtime.New("http://example.com", "key", "secret", 30*time.Second)
	assert.Equal(t, "airtime_provider", provider.Name())
}
