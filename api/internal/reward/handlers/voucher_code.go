package handlers

import (
	"context"
	"fmt"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/jackc/pgx/v5"
)

// VoucherCodeHandler handles pre-loaded voucher code pools
type VoucherCodeHandler struct {
	queries *db.Queries
}

// NewVoucherCodeHandler creates a new voucher code handler
func NewVoucherCodeHandler(queries *db.Queries) *VoucherCodeHandler {
	return &VoucherCodeHandler{
		queries: queries,
	}
}

// Process reserves and issues a voucher code from the pool
// Uses SKIP LOCKED to prevent race conditions in concurrent scenarios
func (h *VoucherCodeHandler) Process(ctx context.Context, issuance *db.Issuance, rewardCatalog *db.RewardCatalog) (*ProcessResult, error) {
	// Reserve a code from the voucher pool
	// This uses FOR UPDATE SKIP LOCKED to handle concurrent reservations
	voucherCode, err := h.queries.ReserveVoucherCode(ctx, db.ReserveVoucherCodeParams{
		TenantID:   issuance.TenantID,
		RewardID:   rewardCatalog.ID,
		IssuanceID: issuance.ID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no voucher codes available")
		}
		return nil, fmt.Errorf("failed to reserve voucher code: %w", err)
	}

	// Mark the code as issued
	err = h.queries.MarkVoucherCodeIssued(ctx, db.MarkVoucherCodeIssuedParams{
		ID:       voucherCode.ID,
		TenantID: issuance.TenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark voucher code as issued: %w", err)
	}

	return &ProcessResult{
		Code: voucherCode.Code,
		Metadata: map[string]interface{}{
			"voucher_code_id": voucherCode.ID,
		},
	}, nil
}
