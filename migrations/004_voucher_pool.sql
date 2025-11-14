-- Voucher code pool table
-- Version: 1.0
-- Date: 2025-11-14

-- =============================================================================
-- VOUCHER CODES TABLE
-- =============================================================================

-- Table to store pre-loaded voucher codes for pool-based rewards
CREATE TABLE voucher_codes (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  reward_id     uuid NOT NULL REFERENCES reward_catalog(id),
  code          text NOT NULL,
  status        text NOT NULL CHECK (status IN ('available','reserved','issued','invalid')),
  issuance_id   uuid REFERENCES issuances(id),
  created_at    timestamptz NOT NULL DEFAULT now(),
  issued_at     timestamptz,
  UNIQUE (tenant_id, reward_id, code)
);

-- =============================================================================
-- INDEXES
-- =============================================================================

-- Index for quickly finding available codes for a reward (critical for performance)
CREATE INDEX idx_voucher_codes_available ON voucher_codes(reward_id, status)
WHERE status = 'available';

-- Index for finding codes by tenant
CREATE INDEX idx_voucher_codes_tenant ON voucher_codes(tenant_id, reward_id);

-- Index for lookup by issuance
CREATE INDEX idx_voucher_codes_issuance ON voucher_codes(issuance_id)
WHERE issuance_id IS NOT NULL;

-- Index for issued codes tracking
CREATE INDEX idx_voucher_codes_issued ON voucher_codes(reward_id, status, issued_at DESC)
WHERE status = 'issued';

-- =============================================================================
-- ROW LEVEL SECURITY (RLS)
-- =============================================================================

-- Enable RLS for tenant isolation
ALTER TABLE voucher_codes ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_voucher_codes
  ON voucher_codes
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- Force RLS (prevent superuser bypass for app connections)
ALTER TABLE voucher_codes FORCE ROW LEVEL SECURITY;

-- =============================================================================
-- HELPER FUNCTION: Get Available Code Count
-- =============================================================================

-- Function to quickly get count of available codes for a reward
CREATE OR REPLACE FUNCTION get_available_code_count(
  p_reward_id uuid
) RETURNS bigint AS $$
BEGIN
  RETURN (
    SELECT COUNT(*)
    FROM voucher_codes
    WHERE reward_id = p_reward_id
      AND status = 'available'
  );
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- SAMPLE VOUCHER CODES FOR DEMO
-- =============================================================================

-- Insert sample voucher codes for the Coffee Shop reward from seed data
-- Only insert if the seed tenant exists
DO $$
DECLARE
  v_tenant_id uuid := '00000000-0000-0000-0000-000000000001';
  v_reward_id uuid := '30000000-0000-0000-0000-000000000005'; -- Free Coffee Voucher
  v_count int := 0;
BEGIN
  -- Check if demo tenant exists
  SELECT COUNT(*) INTO v_count FROM tenants WHERE id = v_tenant_id;

  IF v_count > 0 THEN
    -- Generate 50 sample voucher codes
    INSERT INTO voucher_codes (tenant_id, reward_id, code, status)
    SELECT
      v_tenant_id,
      v_reward_id,
      'COFFEE-' || UPPER(substr(md5(random()::text), 1, 8)),
      'available'
    FROM generate_series(1, 50);

    RAISE NOTICE 'Created 50 sample voucher codes for Coffee Shop reward';
  ELSE
    RAISE NOTICE 'Demo tenant not found, skipping sample voucher codes';
  END IF;
END $$;

-- =============================================================================
-- COMPLETION
-- =============================================================================

DO $$
BEGIN
  RAISE NOTICE 'Voucher pool table created successfully';
  RAISE NOTICE '  - Table: voucher_codes';
  RAISE NOTICE '  - Indexes: available codes, tenant lookup, issuance tracking';
  RAISE NOTICE '  - RLS policies: enabled with tenant isolation';
  RAISE NOTICE '  - Helper function: get_available_code_count()';
END $$;
