-- name: ReserveIssuance :one
INSERT INTO issuances (tenant_id, customer_id, campaign_id, reward_id, status, currency, face_amount, cost_amount, issued_at)
VALUES ($1, $2, $3, $4, 'reserved', $5, $6, $7, now())
RETURNING *;

-- name: UpdateIssuanceStatus :exec
UPDATE issuances
SET status = $4,
    redeemed_at = CASE WHEN $4 = 'redeemed' THEN now() ELSE redeemed_at END
WHERE id = $1 AND tenant_id = $2 AND status = $3;

-- name: GetIssuanceByID :one
SELECT * FROM issuances
WHERE id = $1 AND tenant_id = $2;

-- name: ListIssuancesByCustomer :many
SELECT * FROM issuances
WHERE tenant_id = $1 AND customer_id = $2
ORDER BY issued_at DESC
LIMIT $3 OFFSET $4;

-- name: ListActiveIssuances :many
SELECT * FROM issuances
WHERE tenant_id = $1 AND customer_id = $2 AND status IN ('issued', 'reserved')
ORDER BY issued_at DESC;

-- name: UpdateIssuanceDetails :exec
UPDATE issuances
SET code = $3,
    external_ref = $4,
    expires_at = $5
WHERE id = $1 AND tenant_id = $2;

-- name: GetExpiredIssuances :many
SELECT id, tenant_id, campaign_id, cost_amount
FROM issuances
WHERE status = 'issued'
  AND expires_at IS NOT NULL
  AND expires_at < NOW()
FOR UPDATE SKIP LOCKED;
