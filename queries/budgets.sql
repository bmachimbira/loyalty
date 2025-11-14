-- name: CreateBudget :one
INSERT INTO budgets (tenant_id, name, currency, soft_cap, hard_cap, balance, period)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetBudgetByID :one
SELECT * FROM budgets
WHERE id = $1 AND tenant_id = $2;

-- name: ListBudgets :many
SELECT * FROM budgets
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: UpdateBudgetBalance :exec
UPDATE budgets
SET balance = balance + $3
WHERE id = $1 AND tenant_id = $2;

-- name: InsertLedgerEntry :one
INSERT INTO ledger_entries (tenant_id, budget_id, entry_type, currency, amount, ref_type, ref_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetLedgerEntries :many
SELECT * FROM ledger_entries
WHERE tenant_id = $1 AND budget_id = $2
  AND created_at >= $3 AND created_at <= $4
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;
