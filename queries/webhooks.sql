-- name: GetWebhookByID :one
SELECT * FROM webhooks WHERE id = $1 AND tenant_id = current_setting('app.tenant_id', true)::uuid;

-- name: GetWebhooksByEvent :many
SELECT * FROM webhooks
WHERE tenant_id = current_setting('app.tenant_id', true)::uuid
  AND active = true
  AND $1::text = ANY(events);

-- name: ListWebhooks :many
SELECT * FROM webhooks
WHERE tenant_id = current_setting('app.tenant_id', true)::uuid
ORDER BY created_at DESC;

-- name: CreateWebhook :one
INSERT INTO webhooks (tenant_id, name, url, events, secret, active)
VALUES (current_setting('app.tenant_id', true)::uuid, $1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateWebhook :one
UPDATE webhooks
SET name = $2, url = $3, events = $4, active = $5
WHERE id = $1 AND tenant_id = current_setting('app.tenant_id', true)::uuid
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM webhooks
WHERE id = $1 AND tenant_id = current_setting('app.tenant_id', true)::uuid;

-- name: InsertWebhookDelivery :one
INSERT INTO webhook_deliveries (tenant_id, webhook_id, event_type, attempt, status, response_code, response_body, error_message)
VALUES (current_setting('app.tenant_id', true)::uuid, $1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetWebhookDeliveries :many
SELECT * FROM webhook_deliveries
WHERE webhook_id = $1 AND tenant_id = current_setting('app.tenant_id', true)::uuid
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetFailedDeliveries :many
SELECT * FROM webhook_deliveries
WHERE tenant_id = current_setting('app.tenant_id', true)::uuid
  AND status = 'failed'
  AND created_at > $1
ORDER BY created_at DESC;
