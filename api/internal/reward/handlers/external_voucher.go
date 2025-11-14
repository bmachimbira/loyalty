package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/rewardtypes"
)

// Connector defines the interface for external voucher providers
type Connector interface {
	// IssueVoucher calls the external provider to issue a voucher
	IssueVoucher(ctx context.Context, params IssueParams) (*IssueResponse, error)
}

// IssueParams contains parameters for issuing a voucher
type IssueParams struct {
	TenantID   string
	CustomerID string
	ProductID  string
	Amount     float64
	Currency   string
	Reference  string // Issuance ID for idempotency
}

// IssueResponse contains the response from issuing a voucher
type IssueResponse struct {
	VoucherCode   string
	TransactionID string
	ExpiresAt     *time.Time
	PIN           string // Optional PIN for voucher
	Instructions  string // Usage instructions
}

// ExternalVoucherHandler handles external voucher provider integration
type ExternalVoucherHandler struct {
	connectors map[string]Connector
}

// NewExternalVoucherHandler creates a new external voucher handler
func NewExternalVoucherHandler() *ExternalVoucherHandler {
	return &ExternalVoucherHandler{
		connectors: make(map[string]Connector),
	}
}

// RegisterConnector registers an external provider connector
func (h *ExternalVoucherHandler) RegisterConnector(providerID string, connector Connector) {
	h.connectors[providerID] = connector
}

// Process issues a voucher through an external provider
func (h *ExternalVoucherHandler) Process(ctx context.Context, issuance *db.Issuance, rewardCatalog *db.RewardCatalog) (*ProcessResult, error) {
	// Parse metadata to get supplier info
	var meta rewardtypes.ExternalVoucherMetadata
	if err := json.Unmarshal(rewardCatalog.Metadata, &meta); err != nil {
		return nil, fmt.Errorf("invalid external voucher metadata: %w", err)
	}

	// Get the connector for this supplier
	connector, ok := h.connectors[meta.SupplierID]
	if !ok {
		return nil, fmt.Errorf("supplier not configured: %s", meta.SupplierID)
	}

	// Get face value as float64
	faceValue, err := issuance.FaceAmount.Float64Value()
	if err != nil {
		return nil, fmt.Errorf("invalid face amount: %w", err)
	}

	// Get currency
	currency := "USD"
	if issuance.Currency.Valid {
		currency = issuance.Currency.String
	}

	// Prepare issue parameters
	params := IssueParams{
		TenantID:   issuance.TenantID.String(),
		CustomerID: issuance.CustomerID.String(),
		ProductID:  meta.ProductID,
		Amount:     faceValue.Float64,
		Currency:   currency,
		Reference:  issuance.ID.String(), // For idempotency
	}

	// Call external provider with retry logic
	var resp *IssueResponse
	var lastErr error

	// Retry up to 3 times with exponential backoff
	for attempt := 1; attempt <= 3; attempt++ {
		resp, lastErr = connector.IssueVoucher(ctx, params)
		if lastErr == nil {
			break
		}

		if attempt < 3 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to issue voucher after 3 attempts: %w", lastErr)
	}

	// Prepare result metadata
	resultMeta := map[string]interface{}{
		"supplier_id":    meta.SupplierID,
		"product_id":     meta.ProductID,
		"transaction_id": resp.TransactionID,
	}

	if resp.PIN != "" {
		resultMeta["pin"] = resp.PIN
	}

	if resp.Instructions != "" {
		resultMeta["instructions"] = resp.Instructions
	}

	return &ProcessResult{
		Code:        resp.VoucherCode,
		ExternalRef: resp.TransactionID,
		ExpiresAt:   resp.ExpiresAt,
		Metadata:    resultMeta,
	}, nil
}
