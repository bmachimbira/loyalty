package airtime

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/connectors"
)

// Provider implements the Connector interface for airtime/data providers
type Provider struct {
	client  *http.Client
	baseURL string
	apiKey  string
	secret  string
	name    string
}

// New creates a new airtime provider
func New(baseURL, apiKey, secret string, timeout time.Duration) *Provider {
	return &Provider{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
		secret:  secret,
		name:    "airtime_provider",
	}
}

// NewDataProvider creates a new data provider (similar to airtime)
func NewDataProvider(baseURL, apiKey, secret string, timeout time.Duration) *Provider {
	return &Provider{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
		secret:  secret,
		name:    "data_provider",
	}
}

// Name returns the provider's identifier
func (p *Provider) Name() string {
	return p.name
}

// IssueVoucher issues a voucher through the provider's API
func (p *Provider) IssueVoucher(ctx context.Context, params connectors.IssueParams) (*connectors.IssueResponse, error) {
	// Build request payload
	payload := map[string]interface{}{
		"product_id":   params.ProductID,
		"phone_number": params.PhoneNumber,
		"amount":       params.Amount,
		"currency":     params.Currency,
		"reference":    params.Reference,
		"customer_id":  params.CustomerID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/v1/issue", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.apiKey)
	req.Header.Set("X-Signature", p.sign(body))

	// Send request with retry logic
	retryConfig := connectors.DefaultRetryConfig()
	var resp *http.Response

	err = connectors.RetryWithBackoff(ctx, retryConfig, func() error {
		var reqErr error
		resp, reqErr = p.client.Do(req)
		if reqErr != nil {
			return reqErr
		}

		// Check for retriable HTTP status codes
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			return connectors.NewHTTPError(resp.StatusCode, "server error or rate limit")
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("request failed after retries: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return nil, connectors.NewHTTPError(resp.StatusCode, string(respBody))
	}

	// Parse success response
	var result struct {
		VoucherCode   string `json:"voucher_code"`
		TransactionID string `json:"transaction_id"`
		Status        string `json:"status"`
		Message       string `json:"message"`
		ExternalRef   string `json:"external_ref"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &connectors.IssueResponse{
		VoucherCode:   result.VoucherCode,
		TransactionID: result.TransactionID,
		Status:        result.Status,
		Message:       result.Message,
		ExternalRef:   result.ExternalRef,
	}, nil
}

// CheckStatus checks the status of a transaction
func (p *Provider) CheckStatus(ctx context.Context, transactionID string) (*connectors.StatusResponse, error) {
	// Create request
	url := fmt.Sprintf("%s/api/v1/status/%s", p.baseURL, transactionID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("X-API-Key", p.apiKey)
	req.Header.Set("X-Signature", p.sign([]byte(transactionID)))

	// Send request with retry
	retryConfig := connectors.DefaultRetryConfig()
	var resp *http.Response

	err = connectors.RetryWithBackoff(ctx, retryConfig, func() error {
		var reqErr error
		resp, reqErr = p.client.Do(req)
		if reqErr != nil {
			return reqErr
		}

		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			return connectors.NewHTTPError(resp.StatusCode, "server error or rate limit")
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("status check failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, connectors.NewHTTPError(resp.StatusCode, string(respBody))
	}

	// Parse response
	var result struct {
		Status        string `json:"status"`
		Message       string `json:"message"`
		TransactionID string `json:"transaction_id"`
		UpdatedAt     string `json:"updated_at"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &connectors.StatusResponse{
		Status:        result.Status,
		Message:       result.Message,
		TransactionID: result.TransactionID,
		UpdatedAt:     result.UpdatedAt,
	}, nil
}

// CancelVoucher cancels a voucher (if supported by provider)
func (p *Provider) CancelVoucher(ctx context.Context, externalRef string) error {
	// Create request
	payload := map[string]interface{}{
		"external_ref": externalRef,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/v1/cancel", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.apiKey)
	req.Header.Set("X-Signature", p.sign(body))

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("cancel request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return connectors.NewHTTPError(resp.StatusCode, string(respBody))
	}

	return nil
}

// sign generates an HMAC signature for the request
func (p *Provider) sign(body []byte) string {
	mac := hmac.New(sha256.New, []byte(p.secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
