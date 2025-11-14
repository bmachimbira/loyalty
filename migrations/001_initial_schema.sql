-- Initial schema for Zimbabwe Loyalty Platform
-- Version: 1.0
-- Date: 2025-11-14

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";

-- =============================================================================
-- TENANCY & IDENTITY
-- =============================================================================

CREATE TABLE tenants (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name          text NOT NULL,
  country_code  text NOT NULL DEFAULT 'ZW',
  default_ccy   text NOT NULL CHECK (default_ccy IN ('ZWG','USD')),
  theme         jsonb NOT NULL DEFAULT '{}',
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE staff_users (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenants(id),
  email        citext NOT NULL,
  full_name    text NOT NULL,
  role         text NOT NULL CHECK (role IN ('owner','admin','staff','viewer')),
  pwd_hash     text NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, email)
);

CREATE INDEX idx_staff_users_tenant ON staff_users(tenant_id);

CREATE TABLE customers (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  phone_e164    text,
  external_ref  text,
  status        text NOT NULL DEFAULT 'active',
  created_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, phone_e164),
  UNIQUE (tenant_id, external_ref)
);

CREATE INDEX idx_customers_tenant ON customers(tenant_id);
CREATE INDEX idx_customers_phone ON customers(tenant_id, phone_e164) WHERE phone_e164 IS NOT NULL;

CREATE TABLE consents (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenants(id),
  customer_id  uuid NOT NULL REFERENCES customers(id),
  channel      text NOT NULL CHECK (channel IN ('whatsapp','sms','email','web')),
  purpose      text NOT NULL,
  granted      boolean NOT NULL,
  occurred_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_consents_customer ON consents(customer_id);

-- =============================================================================
-- BUDGETS & LEDGER
-- =============================================================================

CREATE TABLE budgets (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenants(id),
  name         text NOT NULL,
  currency     text NOT NULL CHECK (currency IN ('ZWG','USD')),
  soft_cap     numeric(18,2) NOT NULL DEFAULT 0,
  hard_cap     numeric(18,2) NOT NULL DEFAULT 0,
  balance      numeric(18,2) NOT NULL DEFAULT 0,
  period       text NOT NULL DEFAULT 'rolling',
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_budgets_tenant ON budgets(tenant_id);

CREATE TABLE ledger_entries (
  id           bigserial PRIMARY KEY,
  tenant_id    uuid NOT NULL REFERENCES tenants(id),
  budget_id    uuid NOT NULL REFERENCES budgets(id),
  entry_type   text NOT NULL CHECK (entry_type IN
                 ('fund','reserve','release','charge','expire','reverse')),
  currency     text NOT NULL CHECK (currency IN ('ZWG','USD')),
  amount       numeric(18,2) NOT NULL,
  ref_type     text,
  ref_id       uuid,
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_ledger_entries_tenant_budget ON ledger_entries(tenant_id, budget_id, created_at DESC);

-- =============================================================================
-- CATALOG & RULES
-- =============================================================================

CREATE TABLE reward_catalog (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  name          text NOT NULL,
  type          text NOT NULL CHECK (type IN
    ('discount','voucher_code','points_credit','external_voucher','physical_item','webhook_custom')),
  face_value    numeric(18,2),
  currency      text CHECK (currency IN ('ZWG','USD')),
  inventory     text NOT NULL CHECK (inventory IN ('none','pool','jit_external')),
  supplier_id   uuid,
  metadata      jsonb NOT NULL DEFAULT '{}',
  active        boolean NOT NULL DEFAULT true,
  UNIQUE (tenant_id, name)
);

CREATE INDEX idx_reward_catalog_tenant ON reward_catalog(tenant_id);

CREATE TABLE campaigns (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  name          text NOT NULL,
  start_at      timestamptz,
  end_at        timestamptz,
  budget_id     uuid REFERENCES budgets(id),
  status        text NOT NULL DEFAULT 'active'
);

CREATE INDEX idx_campaigns_tenant ON campaigns(tenant_id);

CREATE TABLE rules (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  campaign_id   uuid REFERENCES campaigns(id),
  name          text NOT NULL,
  event_type    text NOT NULL,
  conditions    jsonb NOT NULL,
  reward_id     uuid NOT NULL REFERENCES reward_catalog(id),
  per_user_cap  int NOT NULL DEFAULT 1,
  global_cap    int,
  cool_down_sec int NOT NULL DEFAULT 0,
  active        boolean NOT NULL DEFAULT true
);

CREATE INDEX idx_rules_tenant_event ON rules(tenant_id, event_type) WHERE active = true;

-- =============================================================================
-- EVENTS & ISSUANCES
-- =============================================================================

CREATE TABLE events (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id        uuid NOT NULL REFERENCES tenants(id),
  customer_id      uuid REFERENCES customers(id),
  event_type       text NOT NULL,
  properties       jsonb NOT NULL DEFAULT '{}',
  occurred_at      timestamptz NOT NULL,
  source           text NOT NULL,
  location_id      uuid,
  idempotency_key  text NOT NULL,
  created_at       timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, idempotency_key)
);

CREATE INDEX idx_events_tenant_type_occurred ON events(tenant_id, event_type, occurred_at DESC);

CREATE TABLE issuances (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      uuid NOT NULL REFERENCES tenants(id),
  customer_id    uuid NOT NULL REFERENCES customers(id),
  campaign_id    uuid REFERENCES campaigns(id),
  reward_id      uuid NOT NULL REFERENCES reward_catalog(id),
  status         text NOT NULL CHECK (status IN ('reserved','issued','redeemed','expired','cancelled','failed')),
  code           text,
  external_ref   text,
  currency       text CHECK (currency IN ('ZWG','USD')),
  cost_amount    numeric(18,2),
  face_amount    numeric(18,2),
  issued_at      timestamptz,
  expires_at     timestamptz,
  redeemed_at    timestamptz,
  event_id       uuid REFERENCES events(id)
);

CREATE INDEX idx_issuances_tenant_customer_status ON issuances(tenant_id, customer_id, status);
CREATE INDEX idx_issuances_tenant_reward ON issuances(tenant_id, reward_id);

-- =============================================================================
-- CHANNELS
-- =============================================================================

CREATE TABLE wa_sessions (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  customer_id   uuid REFERENCES customers(id),
  wa_id         text NOT NULL,
  phone_e164    text NOT NULL,
  state         jsonb NOT NULL DEFAULT '{}',
  last_msg_at   timestamptz NOT NULL DEFAULT now(),
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_wa_sessions_wa_id ON wa_sessions(wa_id);

CREATE TABLE ussd_sessions (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  customer_id   uuid REFERENCES customers(id),
  session_id    text NOT NULL,
  phone_e164    text NOT NULL,
  state         jsonb NOT NULL DEFAULT '{}',
  last_input_at timestamptz NOT NULL DEFAULT now(),
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_ussd_sessions_session_id ON ussd_sessions(session_id);

-- =============================================================================
-- WEBHOOKS & AUDIT
-- =============================================================================

CREATE TABLE webhooks (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  name          text NOT NULL,
  url           text NOT NULL,
  events        text[] NOT NULL,
  secret        text NOT NULL,
  active        boolean NOT NULL DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE audit_logs (
  id            bigserial PRIMARY KEY,
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  actor_type    text NOT NULL,
  actor_id      uuid,
  action        text NOT NULL,
  resource_type text,
  resource_id   uuid,
  details       jsonb NOT NULL DEFAULT '{}',
  ip_address    inet,
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_tenant_created ON audit_logs(tenant_id, created_at DESC);

-- =============================================================================
-- ROW LEVEL SECURITY (RLS)
-- =============================================================================

-- Enable RLS on tenant-scoped tables
ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE consents ENABLE ROW LEVEL SECURITY;
ALTER TABLE budgets ENABLE ROW LEVEL SECURITY;
ALTER TABLE ledger_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE reward_catalog ENABLE ROW LEVEL SECURITY;
ALTER TABLE campaigns ENABLE ROW LEVEL SECURITY;
ALTER TABLE rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE events ENABLE ROW LEVEL SECURITY;
ALTER TABLE issuances ENABLE ROW LEVEL SECURITY;
ALTER TABLE wa_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE ussd_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhooks ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- Create RLS policies
CREATE POLICY tenant_isolation_customers
  ON customers
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_consents
  ON consents
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_budgets
  ON budgets
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_ledger_entries
  ON ledger_entries
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_reward_catalog
  ON reward_catalog
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_campaigns
  ON campaigns
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_rules
  ON rules
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_events
  ON events
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_issuances
  ON issuances
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_wa_sessions
  ON wa_sessions
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_ussd_sessions
  ON ussd_sessions
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_webhooks
  ON webhooks
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_audit_logs
  ON audit_logs
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- Force RLS (prevent superuser bypass for app connections)
ALTER TABLE customers FORCE ROW LEVEL SECURITY;
ALTER TABLE consents FORCE ROW LEVEL SECURITY;
ALTER TABLE budgets FORCE ROW LEVEL SECURITY;
ALTER TABLE ledger_entries FORCE ROW LEVEL SECURITY;
ALTER TABLE reward_catalog FORCE ROW LEVEL SECURITY;
ALTER TABLE campaigns FORCE ROW LEVEL SECURITY;
ALTER TABLE rules FORCE ROW LEVEL SECURITY;
ALTER TABLE events FORCE ROW LEVEL SECURITY;
ALTER TABLE issuances FORCE ROW LEVEL SECURITY;
ALTER TABLE wa_sessions FORCE ROW LEVEL SECURITY;
ALTER TABLE ussd_sessions FORCE ROW LEVEL SECURITY;
ALTER TABLE webhooks FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;
