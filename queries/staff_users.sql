-- Staff user queries
-- sqlc query file for staff user operations

-- name: CreateStaffUser :one
INSERT INTO staff_users (tenant_id, email, full_name, role, pwd_hash)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetStaffUserByEmail :one
SELECT * FROM staff_users
WHERE tenant_id = $1 AND email = $2;

-- name: GetStaffUserByID :one
SELECT * FROM staff_users
WHERE id = $1 AND tenant_id = $2;

-- name: ListStaffUsers :many
SELECT * FROM staff_users
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: UpdateStaffUserRole :exec
UPDATE staff_users
SET role = $3
WHERE id = $1 AND tenant_id = $2;

-- name: UpdateStaffUserPassword :exec
UPDATE staff_users
SET pwd_hash = $3
WHERE id = $1 AND tenant_id = $2;

-- name: UpdateStaffUserFullName :exec
UPDATE staff_users
SET full_name = $3
WHERE id = $1 AND tenant_id = $2;

-- name: DeleteStaffUser :exec
DELETE FROM staff_users
WHERE id = $1 AND tenant_id = $2;
