-- Webhook deliveries table
-- Tracks webhook delivery attempts for auditing and debugging

CREATE TABLE webhook_deliveries (
  id             bigserial PRIMARY KEY,
  tenant_id      uuid NOT NULL REFERENCES tenants(id),
  webhook_id     uuid NOT NULL REFERENCES webhooks(id),
  event_type     text NOT NULL,
  attempt        int NOT NULL DEFAULT 1,
  status         text NOT NULL CHECK (status IN ('pending','success','failed')),
  response_code  int,
  response_body  text,
  error_message  text,
  created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id, created_at DESC);
CREATE INDEX idx_webhook_deliveries_status ON webhook_deliveries(tenant_id, status, created_at DESC);

-- Enable RLS
ALTER TABLE webhook_deliveries ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_webhook_deliveries
  ON webhook_deliveries
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

ALTER TABLE webhook_deliveries FORCE ROW LEVEL SECURITY;
