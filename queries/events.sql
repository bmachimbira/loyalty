-- name: InsertEvent :one
INSERT INTO events (tenant_id, customer_id, event_type, properties, occurred_at, source, idempotency_key)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetEventByIdemKey :one
SELECT * FROM events WHERE tenant_id = $1 AND idempotency_key = $2;

-- name: GetActiveRulesForEvent :many
SELECT * FROM rules
WHERE tenant_id = $1 AND event_type = $2 AND active = true;

-- name: ListEventsByCustomer :many
SELECT * FROM events
WHERE tenant_id = $1 AND customer_id = $2
ORDER BY occurred_at DESC
LIMIT $3 OFFSET $4;
