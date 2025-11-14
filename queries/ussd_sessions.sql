-- USSD session queries
-- sqlc query file for USSD session management

-- name: CreateUSSDSession :one
INSERT INTO ussd_sessions (tenant_id, customer_id, session_id, phone_e164, state, last_input_at)
VALUES ($1, $2, $3, $4, $5, NOW())
RETURNING *;

-- name: GetUSSDSessionByID :one
SELECT * FROM ussd_sessions WHERE session_id = $1;

-- name: GetUSSDSessionByCustomer :one
SELECT * FROM ussd_sessions
WHERE tenant_id = $1 AND customer_id = $2
ORDER BY last_input_at DESC
LIMIT 1;

-- name: UpdateUSSDSessionState :exec
UPDATE ussd_sessions
SET state = $2, last_input_at = NOW()
WHERE session_id = $1;

-- name: UpdateUSSDSessionCustomer :exec
UPDATE ussd_sessions
SET customer_id = $2, last_input_at = NOW()
WHERE session_id = $1;

-- name: GetActiveUSSDSessions :many
SELECT * FROM ussd_sessions
WHERE tenant_id = $1
  AND last_input_at > NOW() - INTERVAL '1 hour'
ORDER BY last_input_at DESC;

-- name: DeleteUSSDSession :exec
DELETE FROM ussd_sessions
WHERE session_id = $1;

-- name: DeleteOldUSSDSessions :exec
DELETE FROM ussd_sessions
WHERE last_input_at < NOW() - INTERVAL '24 hours';

-- name: GetUSSDSessionsByTenant :many
SELECT * FROM ussd_sessions
WHERE tenant_id = $1
ORDER BY last_input_at DESC
LIMIT $2 OFFSET $3;
