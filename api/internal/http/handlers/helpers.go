package handlers

import (
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/jackc/pgx/v5/pgtype"
)

// formatUUID converts pgtype.UUID to string
func formatUUID(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}
	b := uuid.Bytes
	return httputil.FormatUUID(b)
}

// formatTimestamp converts pgtype.Timestamptz to ISO8601 string
func formatTimestamp(ts pgtype.Timestamptz) string {
	return httputil.FormatTimestamp(ts)
}
