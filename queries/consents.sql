-- Consent queries
-- sqlc query file for consent management

-- name: RecordConsent :one
INSERT INTO consents (tenant_id, customer_id, channel, purpose, granted, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCustomerConsents :many
SELECT * FROM consents
WHERE tenant_id = $1 AND customer_id = $2
ORDER BY occurred_at DESC;

-- name: GetLatestConsent :one
SELECT * FROM consents
WHERE tenant_id = $1 AND customer_id = $2 AND channel = $3 AND purpose = $4
ORDER BY occurred_at DESC
LIMIT 1;

-- name: GetConsentsByChannel :many
SELECT * FROM consents
WHERE tenant_id = $1 AND channel = $2
ORDER BY occurred_at DESC
LIMIT $3 OFFSET $4;

-- name: GetConsentsByPurpose :many
SELECT * FROM consents
WHERE tenant_id = $1 AND purpose = $2
ORDER BY occurred_at DESC
LIMIT $3 OFFSET $4;

-- name: CountGrantedConsents :one
SELECT COUNT(*) FROM consents
WHERE tenant_id = $1 AND channel = $2 AND purpose = $3 AND granted = true;

-- name: CountRevokedConsents :one
SELECT COUNT(*) FROM consents
WHERE tenant_id = $1 AND channel = $2 AND purpose = $3 AND granted = false;
