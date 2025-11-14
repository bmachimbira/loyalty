-- Tenant queries
-- sqlc query file for tenant operations

-- name: CreateTenant :one
INSERT INTO tenants (name, country_code, default_ccy, theme)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetTenantByID :one
SELECT * FROM tenants WHERE id = $1;

-- name: ListTenants :many
SELECT * FROM tenants
ORDER BY created_at DESC;

-- name: UpdateTenantTheme :exec
UPDATE tenants
SET theme = $2
WHERE id = $1;

-- name: UpdateTenantName :exec
UPDATE tenants
SET name = $2
WHERE id = $1;

-- name: UpdateTenantDefaultCurrency :exec
UPDATE tenants
SET default_ccy = $2
WHERE id = $1;
