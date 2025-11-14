-- name: CreateReward :one
INSERT INTO reward_catalog (tenant_id, name, type, face_value, currency, inventory, supplier_id, metadata, active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetRewardByID :one
SELECT * FROM reward_catalog
WHERE id = $1 AND tenant_id = $2;

-- name: ListRewards :many
SELECT * FROM reward_catalog
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: ListActiveRewards :many
SELECT * FROM reward_catalog
WHERE tenant_id = $1 AND active = true
ORDER BY name;

-- name: UpdateRewardStatus :exec
UPDATE reward_catalog
SET active = $3
WHERE id = $1 AND tenant_id = $2;
