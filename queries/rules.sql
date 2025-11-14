-- name: CreateRule :one
INSERT INTO rules (tenant_id, campaign_id, name, event_type, conditions, reward_id, per_user_cap, global_cap, cool_down_sec, active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetRuleByID :one
SELECT * FROM rules
WHERE id = $1 AND tenant_id = $2;

-- name: ListActiveRules :many
SELECT * FROM rules
WHERE tenant_id = $1 AND active = true
ORDER BY created_at DESC;

-- name: UpdateRuleStatus :exec
UPDATE rules
SET active = $3
WHERE id = $1 AND tenant_id = $2;
