-- name: CreateCustomer :one
INSERT INTO customers (tenant_id, phone_e164, external_ref, status)
VALUES ($1, $2, $3, 'active')
RETURNING *;

-- name: GetCustomerByID :one
SELECT * FROM customers
WHERE id = $1 AND tenant_id = $2;

-- name: GetCustomerByPhone :one
SELECT * FROM customers
WHERE tenant_id = $1 AND phone_e164 = $2;

-- name: GetCustomerByExternalRef :one
SELECT * FROM customers
WHERE tenant_id = $1 AND external_ref = $2;

-- name: ListCustomers :many
SELECT * FROM customers
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateCustomerStatus :exec
UPDATE customers
SET status = $3
WHERE id = $1 AND tenant_id = $2;
