-- WhatsApp session queries
-- sqlc query file for WhatsApp session management

-- name: UpsertWASession :one
INSERT INTO wa_sessions (tenant_id, customer_id, wa_id, phone_e164, state, last_msg_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (wa_id) DO UPDATE
SET state = EXCLUDED.state,
    last_msg_at = NOW(),
    customer_id = EXCLUDED.customer_id
RETURNING *;

-- name: GetWASessionByWAID :one
SELECT * FROM wa_sessions WHERE wa_id = $1;

-- name: GetWASessionByCustomer :one
SELECT * FROM wa_sessions
WHERE tenant_id = $1 AND customer_id = $2
ORDER BY last_msg_at DESC
LIMIT 1;

-- name: UpdateWASessionState :exec
UPDATE wa_sessions
SET state = $2, last_msg_at = NOW()
WHERE wa_id = $1;

-- name: UpdateWASessionCustomer :exec
UPDATE wa_sessions
SET customer_id = $2, last_msg_at = NOW()
WHERE wa_id = $1;

-- name: GetActiveWASessions :many
SELECT * FROM wa_sessions
WHERE tenant_id = $1
  AND last_msg_at > NOW() - INTERVAL '24 hours'
ORDER BY last_msg_at DESC;

-- name: DeleteOldWASessions :exec
DELETE FROM wa_sessions
WHERE last_msg_at < NOW() - INTERVAL '7 days';

-- name: GetWASessionsByTenant :many
SELECT * FROM wa_sessions
WHERE tenant_id = $1
ORDER BY last_msg_at DESC
LIMIT $2 OFFSET $3;
