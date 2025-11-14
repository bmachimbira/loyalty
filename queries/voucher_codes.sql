-- Voucher code queries
-- sqlc query file for voucher code pool management

-- name: InsertVoucherCode :one
INSERT INTO voucher_codes (tenant_id, reward_id, code, status)
VALUES ($1, $2, $3, 'available')
RETURNING *;

-- name: ReserveVoucherCode :one
UPDATE voucher_codes
SET status = 'reserved',
    issuance_id = $3
WHERE id = (
  SELECT voucher_codes.id FROM voucher_codes
  WHERE voucher_codes.tenant_id = $1 AND voucher_codes.reward_id = $2 AND voucher_codes.status = 'available'
  ORDER BY voucher_codes.created_at
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: MarkVoucherCodeIssued :exec
UPDATE voucher_codes
SET status = 'issued',
    issued_at = NOW()
WHERE id = $1 AND tenant_id = $2;

-- name: MarkVoucherCodeInvalid :exec
UPDATE voucher_codes
SET status = 'invalid'
WHERE id = $1 AND tenant_id = $2;

-- name: GetVoucherCodeByID :one
SELECT * FROM voucher_codes
WHERE id = $1 AND tenant_id = $2;

-- name: GetVoucherCodeByCode :one
SELECT * FROM voucher_codes
WHERE tenant_id = $1 AND reward_id = $2 AND code = $3;

-- name: ListVoucherCodesByReward :many
SELECT * FROM voucher_codes
WHERE tenant_id = $1 AND reward_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountAvailableVoucherCodes :one
SELECT COUNT(*) FROM voucher_codes
WHERE tenant_id = $1 AND reward_id = $2 AND status = 'available';

-- name: CountVoucherCodesByStatus :one
SELECT COUNT(*) FROM voucher_codes
WHERE tenant_id = $1 AND reward_id = $2 AND status = $3;

-- name: GetVoucherCodesByStatus :many
SELECT * FROM voucher_codes
WHERE tenant_id = $1 AND reward_id = $2 AND status = $3
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: DeleteVoucherCode :exec
DELETE FROM voucher_codes
WHERE id = $1 AND tenant_id = $2;

-- name: GetVoucherCodesByIssuance :many
SELECT * FROM voucher_codes
WHERE tenant_id = $1 AND issuance_id = $2;
