-- Campaign queries
-- sqlc query file for campaign operations

-- name: CreateCampaign :one
INSERT INTO campaigns (tenant_id, name, start_at, end_at, budget_id, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCampaignByID :one
SELECT * FROM campaigns
WHERE id = $1 AND tenant_id = $2;

-- name: ListCampaigns :many
SELECT * FROM campaigns
WHERE tenant_id = $1
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: ListActiveCampaigns :many
SELECT * FROM campaigns
WHERE tenant_id = $1
  AND status = 'active'
  AND (start_at IS NULL OR start_at <= NOW())
  AND (end_at IS NULL OR end_at >= NOW())
ORDER BY name;

-- name: UpdateCampaignStatus :exec
UPDATE campaigns
SET status = $3
WHERE id = $1 AND tenant_id = $2;

-- name: UpdateCampaign :exec
UPDATE campaigns
SET name = $3,
    start_at = $4,
    end_at = $5,
    budget_id = $6,
    status = $7
WHERE id = $1 AND tenant_id = $2;

-- name: GetCampaignsByBudget :many
SELECT * FROM campaigns
WHERE tenant_id = $1 AND budget_id = $2
ORDER BY name;

-- name: GetCampaignsByStatus :many
SELECT * FROM campaigns
WHERE tenant_id = $1 AND status = $2
ORDER BY name
LIMIT $3 OFFSET $4;

-- name: DeleteCampaign :exec
DELETE FROM campaigns
WHERE id = $1 AND tenant_id = $2;
