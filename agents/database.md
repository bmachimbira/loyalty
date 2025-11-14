# Database Agent

## Mission
Manage PostgreSQL database schema, migrations, sqlc queries, and database optimization.

## Prerequisites
- PostgreSQL 16+ installed
- sqlc CLI installed
- Understanding of RLS (Row-Level Security)

## Tasks

### 1. Schema Management

#### Additional Migrations
**File**: `migrations/002_seed_data.sql`

Create seed data for development:
- [ ] Demo tenant
- [ ] Staff users (owner, admin, staff, viewer)
- [ ] Sample customers
- [ ] Sample reward catalog items
- [ ] Sample campaigns and rules
- [ ] Initial budgets

**File**: `migrations/003_indexes_optimization.sql`

Add performance indexes:
```sql
-- Events lookup by customer
CREATE INDEX idx_events_customer_occurred ON events(customer_id, occurred_at DESC);

-- Issuances by status and expiry
CREATE INDEX idx_issuances_status_expires ON issuances(status, expires_at)
WHERE status IN ('issued', 'reserved');

-- Rules by campaign
CREATE INDEX idx_rules_campaign ON rules(campaign_id) WHERE active = true;

-- Composite index for rule matching
CREATE INDEX idx_rules_tenant_event_active ON rules(tenant_id, event_type, active);
```

#### Voucher Code Pool Table
**File**: `migrations/004_voucher_pool.sql`

```sql
CREATE TABLE voucher_codes (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  reward_id     uuid NOT NULL REFERENCES reward_catalog(id),
  code          text NOT NULL,
  status        text NOT NULL CHECK (status IN ('available','reserved','issued','invalid')),
  issuance_id   uuid REFERENCES issuances(id),
  created_at    timestamptz NOT NULL DEFAULT now(),
  issued_at     timestamptz
);

CREATE INDEX idx_voucher_codes_available ON voucher_codes(reward_id, status)
WHERE status = 'available';

ALTER TABLE voucher_codes ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_voucher_codes ON voucher_codes
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
```

### 2. sqlc Queries

#### Tenant Queries
**File**: `queries/tenants.sql`

```sql
-- name: CreateTenant :one
INSERT INTO tenants (name, country_code, default_ccy, theme)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetTenantByID :one
SELECT * FROM tenants WHERE id = $1;

-- name: UpdateTenantTheme :exec
UPDATE tenants SET theme = $2 WHERE id = $1;
```

#### Staff Users Queries
**File**: `queries/staff.sql`

```sql
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
```

#### Consents Queries
**File**: `queries/consents.sql`

```sql
-- name: RecordConsent :one
INSERT INTO consents (tenant_id, customer_id, channel, purpose, granted, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCustomerConsents :many
SELECT * FROM consents
WHERE tenant_id = $1 AND customer_id = $2
ORDER BY occurred_at DESC;

-- name: GetLatestConsent :one
SELECT * FROM consents
WHERE tenant_id = $1 AND customer_id = $2 AND channel = $3 AND purpose = $4
ORDER BY occurred_at DESC
LIMIT 1;
```

#### Campaign Queries
**File**: `queries/campaigns.sql`

```sql
-- name: CreateCampaign :one
INSERT INTO campaigns (tenant_id, name, start_at, end_at, budget_id, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCampaignByID :one
SELECT * FROM campaigns
WHERE id = $1 AND tenant_id = $2;

-- name: ListActiveCampaigns :many
SELECT * FROM campaigns
WHERE tenant_id = $1
  AND status = 'active'
  AND (start_at IS NULL OR start_at <= NOW())
  AND (end_at IS NULL OR end_at >= NOW())
ORDER BY created_at DESC;

-- name: UpdateCampaignStatus :exec
UPDATE campaigns
SET status = $3
WHERE id = $1 AND tenant_id = $2;
```

#### WhatsApp Session Queries
**File**: `queries/whatsapp.sql`

```sql
-- name: UpsertWASession :one
INSERT INTO wa_sessions (tenant_id, customer_id, wa_id, phone_e164, state, last_msg_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (wa_id) DO UPDATE
SET state = EXCLUDED.state,
    last_msg_at = NOW()
RETURNING *;

-- name: GetWASessionByWAID :one
SELECT * FROM wa_sessions WHERE wa_id = $1;

-- name: UpdateWASessionState :exec
UPDATE wa_sessions
SET state = $2, last_msg_at = NOW()
WHERE wa_id = $1;
```

#### Audit Log Queries
**File**: `queries/audit.sql`

```sql
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
```

#### Voucher Code Queries
**File**: `queries/voucher_codes.sql`

```sql
-- name: BulkInsertVoucherCodes :copyfrom
INSERT INTO voucher_codes (tenant_id, reward_id, code, status)
VALUES ($1, $2, $3, 'available');

-- name: ReserveVoucherCode :one
UPDATE voucher_codes
SET status = 'reserved',
    issuance_id = $3
WHERE id = (
  SELECT id FROM voucher_codes
  WHERE tenant_id = $1 AND reward_id = $2 AND status = 'available'
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: MarkVoucherCodeIssued :exec
UPDATE voucher_codes
SET status = 'issued',
    issued_at = NOW()
WHERE id = $1 AND tenant_id = $2;
```

### 3. Advanced Queries

#### Analytics Queries
**File**: `queries/analytics.sql`

```sql
-- name: GetDashboardStats :one
SELECT
  (SELECT COUNT(*) FROM customers WHERE tenant_id = $1 AND status = 'active') as active_customers,
  (SELECT COUNT(*) FROM events WHERE tenant_id = $1 AND occurred_at >= $2) as events_today,
  (SELECT COUNT(*) FROM issuances WHERE tenant_id = $1 AND issued_at >= $2) as rewards_issued_today,
  (SELECT COUNT(*) FROM issuances WHERE tenant_id = $1 AND status = 'redeemed' AND redeemed_at >= $2) as rewards_redeemed_today;

-- name: GetRedemptionRate :one
SELECT
  COUNT(*) FILTER (WHERE status = 'issued') as issued,
  COUNT(*) FILTER (WHERE status = 'redeemed') as redeemed,
  CASE
    WHEN COUNT(*) FILTER (WHERE status = 'issued') = 0 THEN 0
    ELSE (COUNT(*) FILTER (WHERE status = 'redeemed')::float / COUNT(*) FILTER (WHERE status = 'issued')::float) * 100
  END as redemption_rate_percent
FROM issuances
WHERE tenant_id = $1
  AND issued_at >= $2
  AND issued_at <= $3;

-- name: GetTopRewards :many
SELECT
  r.id,
  r.name,
  r.type,
  COUNT(i.id) as issuance_count,
  SUM(i.cost_amount) as total_cost
FROM reward_catalog r
LEFT JOIN issuances i ON i.reward_id = r.id
WHERE r.tenant_id = $1
  AND i.issued_at >= $2
  AND i.issued_at <= $3
GROUP BY r.id, r.name, r.type
ORDER BY issuance_count DESC
LIMIT $4;
```

#### Rule Usage Queries
**File**: `queries/rule_usage.sql`

```sql
-- name: GetRuleIssuanceCount :one
SELECT COUNT(*) FROM issuances
WHERE tenant_id = $1
  AND campaign_id = (SELECT campaign_id FROM rules WHERE id = $2)
  AND issued_at >= $3;

-- name: GetCustomerRuleIssuanceCount :one
SELECT COUNT(*) FROM issuances i
JOIN rules r ON i.campaign_id = r.campaign_id
WHERE i.tenant_id = $1
  AND i.customer_id = $2
  AND r.id = $3;
```

### 4. Database Functions

#### Check Budget Availability
**File**: `migrations/005_functions.sql`

```sql
-- Function to check if budget has capacity
CREATE OR REPLACE FUNCTION check_budget_capacity(
  p_budget_id uuid,
  p_amount numeric
) RETURNS boolean AS $$
DECLARE
  v_balance numeric;
  v_hard_cap numeric;
BEGIN
  SELECT balance, hard_cap
  INTO v_balance, v_hard_cap
  FROM budgets
  WHERE id = p_budget_id;

  RETURN (v_balance + p_amount) <= v_hard_cap;
END;
$$ LANGUAGE plpgsql;

-- Function to reserve budget
CREATE OR REPLACE FUNCTION reserve_budget(
  p_tenant_id uuid,
  p_budget_id uuid,
  p_amount numeric,
  p_currency text,
  p_ref_id uuid
) RETURNS boolean AS $$
BEGIN
  -- Check capacity
  IF NOT check_budget_capacity(p_budget_id, p_amount) THEN
    RETURN false;
  END IF;

  -- Update balance
  UPDATE budgets
  SET balance = balance + p_amount
  WHERE id = p_budget_id AND tenant_id = p_tenant_id;

  -- Insert ledger entry
  INSERT INTO ledger_entries (tenant_id, budget_id, entry_type, currency, amount, ref_type, ref_id)
  VALUES (p_tenant_id, p_budget_id, 'reserve', p_currency, p_amount, 'issuance', p_ref_id);

  RETURN true;
END;
$$ LANGUAGE plpgsql;
```

### 5. Code Generation

Run sqlc to generate Go code:

```bash
make sqlc
# or
sqlc generate
```

Verify generated files in `api/internal/db/`

### 6. Testing

#### RLS Policy Tests
**File**: `migrations/999_test_rls.sql`

```sql
-- Test RLS isolation
DO $$
DECLARE
  tenant1_id uuid := gen_random_uuid();
  tenant2_id uuid := gen_random_uuid();
BEGIN
  -- Create two tenants
  INSERT INTO tenants (id, name, default_ccy) VALUES
    (tenant1_id, 'Tenant 1', 'USD'),
    (tenant2_id, 'Tenant 2', 'USD');

  -- Set context to tenant1
  PERFORM set_config('app.tenant_id', tenant1_id::text, false);

  -- Insert customer for tenant1
  INSERT INTO customers (tenant_id, phone_e164) VALUES (tenant1_id, '+26377111111');

  -- Try to query all customers (should only see tenant1's)
  IF (SELECT COUNT(*) FROM customers) != 1 THEN
    RAISE EXCEPTION 'RLS test failed: saw customers from other tenants';
  END IF;

  -- Switch to tenant2
  PERFORM set_config('app.tenant_id', tenant2_id::text, false);

  -- Should see no customers
  IF (SELECT COUNT(*) FROM customers) != 0 THEN
    RAISE EXCEPTION 'RLS test failed: tenant2 saw tenant1 customers';
  END IF;

  RAISE NOTICE 'RLS tests passed';
END $$;
```

#### Performance Tests
**File**: `scripts/benchmark_queries.sql`

```sql
-- Benchmark event insertion
EXPLAIN ANALYZE
INSERT INTO events (tenant_id, event_type, properties, occurred_at, source, idempotency_key)
VALUES ('...', 'purchase', '{"amount": 50}'::jsonb, NOW(), 'api', 'test-key-1');

-- Benchmark rule lookup
EXPLAIN ANALYZE
SELECT * FROM rules
WHERE tenant_id = '...' AND event_type = 'purchase' AND active = true;
```

### 7. Backup & Restore

Create backup script:
**File**: `scripts/backup.sh`

```bash
#!/bin/bash
pg_dump -U postgres -d loyalty -F c -f loyalty_backup_$(date +%Y%m%d_%H%M%S).dump
```

Create restore script:
**File**: `scripts/restore.sh`

```bash
#!/bin/bash
pg_restore -U postgres -d loyalty -c $1
```

## Completion Criteria

- [ ] All migrations created and tested
- [ ] All sqlc queries implemented
- [ ] Code generation working
- [ ] RLS policies verified
- [ ] Indexes optimized
- [ ] Database functions created
- [ ] Backup/restore procedures documented
- [ ] Performance benchmarks meet targets
