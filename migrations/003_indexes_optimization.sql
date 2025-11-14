-- Performance optimization indexes
-- Version: 1.0
-- Date: 2025-11-14

-- =============================================================================
-- EVENTS TABLE OPTIMIZATIONS
-- =============================================================================

-- Index for customer event history lookup (descending for recent-first)
CREATE INDEX idx_events_customer_occurred ON events(customer_id, occurred_at DESC)
WHERE customer_id IS NOT NULL;

-- Index for event lookup by idempotency key (already has unique constraint, but explicit index helps)
-- Note: UNIQUE constraint already creates an index, so this would be redundant
-- CREATE INDEX idx_events_idempotency ON events(tenant_id, idempotency_key);

-- Index for event source filtering
CREATE INDEX idx_events_source ON events(tenant_id, source, occurred_at DESC);

-- =============================================================================
-- ISSUANCES TABLE OPTIMIZATIONS
-- =============================================================================

-- Index for finding issuances by status and expiry (for expiry worker)
CREATE INDEX idx_issuances_status_expires ON issuances(status, expires_at)
WHERE status IN ('issued', 'reserved') AND expires_at IS NOT NULL;

-- Index for customer reward history (descending for recent-first)
CREATE INDEX idx_issuances_customer_issued ON issuances(customer_id, issued_at DESC)
WHERE issued_at IS NOT NULL;

-- Index for campaign performance analytics
CREATE INDEX idx_issuances_campaign_status ON issuances(campaign_id, status, issued_at DESC)
WHERE campaign_id IS NOT NULL;

-- Index for reward analytics
CREATE INDEX idx_issuances_reward_issued ON issuances(reward_id, issued_at DESC)
WHERE issued_at IS NOT NULL;

-- Index for code lookup (for redemption)
CREATE INDEX idx_issuances_code ON issuances(tenant_id, code)
WHERE code IS NOT NULL;

-- =============================================================================
-- RULES TABLE OPTIMIZATIONS
-- =============================================================================

-- Index for active rules by campaign
CREATE INDEX idx_rules_campaign ON rules(campaign_id)
WHERE active = true AND campaign_id IS NOT NULL;

-- Composite index for rule matching (tenant + event type + active status)
CREATE INDEX idx_rules_tenant_event_active ON rules(tenant_id, event_type, active)
WHERE active = true;

-- Index for rule management (list all rules for tenant)
CREATE INDEX idx_rules_tenant_created ON rules(tenant_id, created_at DESC);

-- =============================================================================
-- CUSTOMERS TABLE OPTIMIZATIONS
-- =============================================================================

-- Index for customer status filtering
CREATE INDEX idx_customers_status ON customers(tenant_id, status);

-- Index for external ref lookup (already has unique constraint)
-- CREATE INDEX idx_customers_external_ref ON customers(tenant_id, external_ref);

-- =============================================================================
-- LEDGER ENTRIES TABLE OPTIMIZATIONS
-- =============================================================================

-- Index for ledger filtering by entry type
CREATE INDEX idx_ledger_entries_type ON ledger_entries(tenant_id, budget_id, entry_type, created_at DESC);

-- Index for reference lookups
CREATE INDEX idx_ledger_entries_ref ON ledger_entries(tenant_id, ref_type, ref_id)
WHERE ref_type IS NOT NULL AND ref_id IS NOT NULL;

-- =============================================================================
-- CAMPAIGNS TABLE OPTIMIZATIONS
-- =============================================================================

-- Index for active campaigns in date range
CREATE INDEX idx_campaigns_active_dates ON campaigns(tenant_id, status, start_at, end_at)
WHERE status = 'active';

-- Index for budget-based campaign lookup
CREATE INDEX idx_campaigns_budget ON campaigns(budget_id)
WHERE budget_id IS NOT NULL;

-- =============================================================================
-- BUDGETS TABLE OPTIMIZATIONS
-- =============================================================================

-- Index for budget filtering by currency and period
CREATE INDEX idx_budgets_currency_period ON budgets(tenant_id, currency, period);

-- =============================================================================
-- CONSENTS TABLE OPTIMIZATIONS
-- =============================================================================

-- Index for consent lookup by customer and channel
CREATE INDEX idx_consents_customer_channel ON consents(customer_id, channel, occurred_at DESC);

-- Index for latest consent per purpose
CREATE INDEX idx_consents_customer_purpose ON consents(customer_id, channel, purpose, occurred_at DESC);

-- =============================================================================
-- WHATSAPP SESSIONS OPTIMIZATIONS
-- =============================================================================

-- Index for session lookup by customer
CREATE INDEX idx_wa_sessions_customer ON wa_sessions(customer_id, last_msg_at DESC)
WHERE customer_id IS NOT NULL;

-- Index for active sessions (last message in last 24 hours)
CREATE INDEX idx_wa_sessions_active ON wa_sessions(tenant_id, last_msg_at DESC)
WHERE last_msg_at > now() - interval '24 hours';

-- =============================================================================
-- USSD SESSIONS OPTIMIZATIONS
-- =============================================================================

-- Index for session lookup by customer
CREATE INDEX idx_ussd_sessions_customer ON ussd_sessions(customer_id, last_input_at DESC)
WHERE customer_id IS NOT NULL;

-- Index for active sessions
CREATE INDEX idx_ussd_sessions_active ON ussd_sessions(tenant_id, last_input_at DESC)
WHERE last_input_at > now() - interval '1 hour';

-- =============================================================================
-- AUDIT LOGS OPTIMIZATIONS
-- =============================================================================

-- Index for audit log filtering by action
CREATE INDEX idx_audit_logs_action ON audit_logs(tenant_id, action, created_at DESC);

-- Index for audit log filtering by resource
CREATE INDEX idx_audit_logs_resource ON audit_logs(tenant_id, resource_type, resource_id, created_at DESC)
WHERE resource_type IS NOT NULL;

-- Index for audit log filtering by actor
CREATE INDEX idx_audit_logs_actor ON audit_logs(tenant_id, actor_type, actor_id, created_at DESC)
WHERE actor_id IS NOT NULL;

-- =============================================================================
-- REWARD CATALOG OPTIMIZATIONS
-- =============================================================================

-- Index for active rewards by type
CREATE INDEX idx_reward_catalog_type ON reward_catalog(tenant_id, type, active)
WHERE active = true;

-- Index for inventory type filtering
CREATE INDEX idx_reward_catalog_inventory ON reward_catalog(tenant_id, inventory)
WHERE active = true;

-- =============================================================================
-- STAFF USERS OPTIMIZATIONS
-- =============================================================================

-- Index for staff lookup by role
CREATE INDEX idx_staff_users_role ON staff_users(tenant_id, role);

-- =============================================================================
-- WEBHOOKS OPTIMIZATIONS
-- =============================================================================

-- Index for active webhooks
CREATE INDEX idx_webhooks_active ON webhooks(tenant_id, active)
WHERE active = true;

-- =============================================================================
-- COMPLETION
-- =============================================================================

-- Log optimization completion
DO $$
BEGIN
  RAISE NOTICE 'Performance optimization indexes created successfully';
  RAISE NOTICE 'Indexes created for:';
  RAISE NOTICE '  - Events: customer history, source filtering';
  RAISE NOTICE '  - Issuances: status/expiry, customer history, campaign/reward analytics, code lookup';
  RAISE NOTICE '  - Rules: campaign, event matching, tenant management';
  RAISE NOTICE '  - Ledger: entry type, reference lookups';
  RAISE NOTICE '  - Campaigns: active date ranges, budget lookup';
  RAISE NOTICE '  - Consents: channel/purpose lookups';
  RAISE NOTICE '  - Sessions: customer and active session lookups';
  RAISE NOTICE '  - Audit logs: action, resource, actor filtering';
  RAISE NOTICE '  - Other: catalog, staff, webhooks';
END $$;
