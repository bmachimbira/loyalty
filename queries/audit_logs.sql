-- Audit log queries
-- sqlc query file for audit log operations

-- name: InsertAuditLog :one
INSERT INTO audit_logs (tenant_id, actor_type, actor_id, action, resource_type, resource_id, details, ip_address)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE tenant_id = $1
  AND created_at >= $2
  AND created_at <= $3
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetAuditLogsByActor :many
SELECT * FROM audit_logs
WHERE tenant_id = $1
  AND actor_type = $2
  AND actor_id = $3
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetAuditLogsByAction :many
SELECT * FROM audit_logs
WHERE tenant_id = $1
  AND action = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetAuditLogsByResource :many
SELECT * FROM audit_logs
WHERE tenant_id = $1
  AND resource_type = $2
  AND resource_id = $3
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetAuditLogsByIPAddress :many
SELECT * FROM audit_logs
WHERE tenant_id = $1
  AND ip_address = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountAuditLogsByAction :one
SELECT COUNT(*) FROM audit_logs
WHERE tenant_id = $1
  AND action = $2
  AND created_at >= $3
  AND created_at <= $4;

-- name: GetRecentAuditLogs :many
SELECT * FROM audit_logs
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2;
