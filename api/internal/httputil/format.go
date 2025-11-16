package httputil

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// FormatUUID converts a pgtype.UUID to a formatted string
func FormatUUID(b [16]byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// FormatTimestamp converts pgtype.Timestamptz to ISO8601 string
func FormatTimestamp(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.Format(time.RFC3339)
}
