-- Ledger queries
-- sqlc query file for ledger operations
-- Note: InsertLedgerEntry and GetLedgerEntries are in budgets.sql

-- name: GetLedgerEntriesByDateRangeOnly :many
SELECT * FROM ledger_entries
WHERE tenant_id = $1 AND budget_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetLedgerEntriesByDateRange :many
SELECT * FROM ledger_entries
WHERE tenant_id = $1
  AND budget_id = $2
  AND created_at >= $3
  AND created_at <= $4
ORDER BY created_at DESC;

-- name: GetLedgerEntriesByType :many
SELECT * FROM ledger_entries
WHERE tenant_id = $1
  AND budget_id = $2
  AND entry_type = $3
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetLedgerEntryByRef :many
SELECT * FROM ledger_entries
WHERE tenant_id = $1
  AND ref_type = $2
  AND ref_id = $3
ORDER BY created_at DESC;

-- name: GetTotalLedgerAmount :one
SELECT COALESCE(SUM(amount), 0) as total
FROM ledger_entries
WHERE tenant_id = $1 AND budget_id = $2;

-- name: GetLedgerSummaryByType :many
SELECT
  entry_type,
  currency,
  COUNT(*) as entry_count,
  COALESCE(SUM(amount), 0) as total_amount
FROM ledger_entries
WHERE tenant_id = $1
  AND budget_id = $2
  AND created_at >= $3
  AND created_at <= $4
GROUP BY entry_type, currency
ORDER BY entry_type;
