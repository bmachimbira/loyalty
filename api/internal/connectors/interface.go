package connectors

import "context"

// Connector defines the interface for external service integrations
type Connector interface {
	// Name returns the connector's unique identifier
	Name() string

	// IssueVoucher issues a voucher through the external provider
	IssueVoucher(ctx context.Context, params IssueParams) (*IssueResponse, error)

	// CheckStatus checks the status of a previously issued voucher
	CheckStatus(ctx context.Context, transactionID string) (*StatusResponse, error)

	// CancelVoucher cancels a voucher if supported by the provider
	// Returns an error if cancellation is not supported or fails
	CancelVoucher(ctx context.Context, externalRef string) error
}

// IssueParams contains standard parameters for issuing a voucher
type IssueParams struct {
	ProductID   string  // Provider-specific product identifier
	CustomerID  string  // Internal customer ID
	PhoneNumber string  // E.164 formatted phone number
	Amount      float64 // Voucher amount (if applicable)
	Currency    string  // Currency code (ZWG, USD)
	Reference   string  // Internal reference for idempotency
}

// IssueResponse contains the response from issuing a voucher
type IssueResponse struct {
	VoucherCode   string // Voucher code (if applicable)
	TransactionID string // Provider's transaction ID
	Status        string // Status: success, pending, failed
	Message       string // Human-readable message
	ExternalRef   string // Additional reference from provider
}

// StatusResponse contains the status of a voucher
type StatusResponse struct {
	Status        string // Current status
	Message       string // Status message
	TransactionID string // Provider's transaction ID
	UpdatedAt     string // Last update timestamp
}
