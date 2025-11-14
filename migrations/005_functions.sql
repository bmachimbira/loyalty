-- Database functions for loyalty platform
-- Version: 1.0
-- Date: 2025-11-14

-- =============================================================================
-- BUDGET MANAGEMENT FUNCTIONS
-- =============================================================================

-- Function to check if budget has capacity for an amount
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

  -- Return false if budget not found
  IF NOT FOUND THEN
    RETURN false;
  END IF;

  -- Check if adding the amount would exceed hard cap
  RETURN (v_balance + p_amount) <= v_hard_cap;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to reserve budget (transactional)
CREATE OR REPLACE FUNCTION reserve_budget(
  p_tenant_id uuid,
  p_budget_id uuid,
  p_amount numeric,
  p_currency text,
  p_ref_id uuid
) RETURNS boolean AS $$
DECLARE
  v_balance numeric;
  v_hard_cap numeric;
BEGIN
  -- Lock the budget row for update
  SELECT balance, hard_cap
  INTO v_balance, v_hard_cap
  FROM budgets
  WHERE id = p_budget_id AND tenant_id = p_tenant_id
  FOR UPDATE;

  -- Check if budget exists
  IF NOT FOUND THEN
    RAISE EXCEPTION 'Budget not found: %', p_budget_id;
  END IF;

  -- Check capacity
  IF (v_balance + p_amount) > v_hard_cap THEN
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

-- Function to charge (consume) a reservation
CREATE OR REPLACE FUNCTION charge_budget(
  p_tenant_id uuid,
  p_budget_id uuid,
  p_amount numeric,
  p_currency text,
  p_ref_id uuid
) RETURNS boolean AS $$
BEGIN
  -- Insert ledger entry (balance already reserved, just recording the charge)
  INSERT INTO ledger_entries (tenant_id, budget_id, entry_type, currency, amount, ref_type, ref_id)
  VALUES (p_tenant_id, p_budget_id, 'charge', p_currency, p_amount, 'issuance', p_ref_id);

  RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Function to release (refund) a reservation
CREATE OR REPLACE FUNCTION release_budget(
  p_tenant_id uuid,
  p_budget_id uuid,
  p_amount numeric,
  p_currency text,
  p_ref_id uuid
) RETURNS boolean AS $$
BEGIN
  -- Decrease balance (return funds)
  UPDATE budgets
  SET balance = balance - p_amount
  WHERE id = p_budget_id AND tenant_id = p_tenant_id;

  -- Insert ledger entry
  INSERT INTO ledger_entries (tenant_id, budget_id, entry_type, currency, amount, ref_type, ref_id)
  VALUES (p_tenant_id, p_budget_id, 'release', p_currency, -p_amount, 'issuance', p_ref_id);

  RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Function to fund (topup) a budget
CREATE OR REPLACE FUNCTION fund_budget(
  p_tenant_id uuid,
  p_budget_id uuid,
  p_amount numeric,
  p_currency text
) RETURNS boolean AS $$
BEGIN
  -- Increase balance
  UPDATE budgets
  SET balance = balance + p_amount
  WHERE id = p_budget_id AND tenant_id = p_tenant_id;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'Budget not found: %', p_budget_id;
  END IF;

  -- Insert ledger entry
  INSERT INTO ledger_entries (tenant_id, budget_id, entry_type, currency, amount, ref_type, ref_id)
  VALUES (p_tenant_id, p_budget_id, 'fund', p_currency, p_amount, 'topup', NULL);

  RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Function to get budget utilization percentage
CREATE OR REPLACE FUNCTION get_budget_utilization(
  p_budget_id uuid
) RETURNS numeric AS $$
DECLARE
  v_balance numeric;
  v_hard_cap numeric;
BEGIN
  SELECT balance, hard_cap
  INTO v_balance, v_hard_cap
  FROM budgets
  WHERE id = p_budget_id;

  IF NOT FOUND OR v_hard_cap = 0 THEN
    RETURN 0;
  END IF;

  RETURN (v_balance / v_hard_cap) * 100;
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- ISSUANCE HELPER FUNCTIONS
-- =============================================================================

-- Function to get customer issuance count for a rule
CREATE OR REPLACE FUNCTION get_customer_rule_issuance_count(
  p_tenant_id uuid,
  p_customer_id uuid,
  p_rule_id uuid
) RETURNS bigint AS $$
DECLARE
  v_campaign_id uuid;
BEGIN
  -- Get campaign_id from rule
  SELECT campaign_id INTO v_campaign_id
  FROM rules
  WHERE id = p_rule_id AND tenant_id = p_tenant_id;

  IF NOT FOUND THEN
    RETURN 0;
  END IF;

  -- Count issuances for this customer and campaign
  RETURN (
    SELECT COUNT(*)
    FROM issuances
    WHERE tenant_id = p_tenant_id
      AND customer_id = p_customer_id
      AND campaign_id = v_campaign_id
      AND status IN ('reserved', 'issued', 'redeemed')
  );
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to get global issuance count for a rule
CREATE OR REPLACE FUNCTION get_rule_global_issuance_count(
  p_tenant_id uuid,
  p_rule_id uuid
) RETURNS bigint AS $$
DECLARE
  v_campaign_id uuid;
BEGIN
  -- Get campaign_id from rule
  SELECT campaign_id INTO v_campaign_id
  FROM rules
  WHERE id = p_rule_id AND tenant_id = p_tenant_id;

  IF NOT FOUND THEN
    RETURN 0;
  END IF;

  -- Count issuances for this campaign
  RETURN (
    SELECT COUNT(*)
    FROM issuances
    WHERE tenant_id = p_tenant_id
      AND campaign_id = v_campaign_id
      AND status IN ('reserved', 'issued', 'redeemed')
  );
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to check if customer is within cooldown period for a rule
CREATE OR REPLACE FUNCTION is_within_cooldown(
  p_tenant_id uuid,
  p_customer_id uuid,
  p_rule_id uuid,
  p_cooldown_seconds int
) RETURNS boolean AS $$
DECLARE
  v_campaign_id uuid;
  v_last_issued timestamptz;
BEGIN
  -- If no cooldown, always return false
  IF p_cooldown_seconds = 0 THEN
    RETURN false;
  END IF;

  -- Get campaign_id from rule
  SELECT campaign_id INTO v_campaign_id
  FROM rules
  WHERE id = p_rule_id AND tenant_id = p_tenant_id;

  IF NOT FOUND THEN
    RETURN false;
  END IF;

  -- Get most recent issuance time
  SELECT MAX(issued_at) INTO v_last_issued
  FROM issuances
  WHERE tenant_id = p_tenant_id
    AND customer_id = p_customer_id
    AND campaign_id = v_campaign_id
    AND status IN ('reserved', 'issued', 'redeemed');

  -- If no previous issuance, not in cooldown
  IF v_last_issued IS NULL THEN
    RETURN false;
  END IF;

  -- Check if within cooldown period
  RETURN (now() - v_last_issued) < (p_cooldown_seconds || ' seconds')::interval;
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- EXPIRY HELPER FUNCTIONS
-- =============================================================================

-- Function to expire old issuances (for cron/worker)
CREATE OR REPLACE FUNCTION expire_old_issuances()
RETURNS TABLE(
  expired_count bigint,
  budget_released numeric
) AS $$
DECLARE
  v_expired_count bigint := 0;
  v_budget_released numeric := 0;
  v_rec record;
BEGIN
  -- Find and update expired issuances
  FOR v_rec IN
    SELECT id, tenant_id, campaign_id, cost_amount, currency
    FROM issuances
    WHERE status IN ('issued', 'reserved')
      AND expires_at IS NOT NULL
      AND expires_at < now()
    FOR UPDATE SKIP LOCKED
  LOOP
    -- Update status to expired
    UPDATE issuances
    SET status = 'expired'
    WHERE id = v_rec.id;

    -- Release budget if there's a campaign with budget
    IF v_rec.campaign_id IS NOT NULL AND v_rec.cost_amount IS NOT NULL THEN
      DECLARE
        v_budget_id uuid;
      BEGIN
        SELECT budget_id INTO v_budget_id
        FROM campaigns
        WHERE id = v_rec.campaign_id;

        IF v_budget_id IS NOT NULL THEN
          PERFORM release_budget(
            v_rec.tenant_id,
            v_budget_id,
            v_rec.cost_amount,
            v_rec.currency,
            v_rec.id
          );

          v_budget_released := v_budget_released + v_rec.cost_amount;
        END IF;
      END;
    END IF;

    v_expired_count := v_expired_count + 1;
  END LOOP;

  RETURN QUERY SELECT v_expired_count, v_budget_released;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- ANALYTICS HELPER FUNCTIONS
-- =============================================================================

-- Function to calculate redemption rate for a tenant
CREATE OR REPLACE FUNCTION get_redemption_rate(
  p_tenant_id uuid,
  p_from_date timestamptz,
  p_to_date timestamptz
) RETURNS numeric AS $$
DECLARE
  v_issued_count bigint;
  v_redeemed_count bigint;
BEGIN
  SELECT
    COUNT(*) FILTER (WHERE status IN ('issued', 'redeemed')),
    COUNT(*) FILTER (WHERE status = 'redeemed')
  INTO v_issued_count, v_redeemed_count
  FROM issuances
  WHERE tenant_id = p_tenant_id
    AND issued_at >= p_from_date
    AND issued_at <= p_to_date;

  IF v_issued_count = 0 THEN
    RETURN 0;
  END IF;

  RETURN (v_redeemed_count::numeric / v_issued_count::numeric) * 100;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to reconcile budget (check for discrepancies)
CREATE OR REPLACE FUNCTION reconcile_budget(
  p_budget_id uuid
) RETURNS TABLE(
  current_balance numeric,
  calculated_balance numeric,
  discrepancy numeric
) AS $$
DECLARE
  v_current_balance numeric;
  v_calculated_balance numeric;
BEGIN
  -- Get current balance from budgets table
  SELECT balance INTO v_current_balance
  FROM budgets
  WHERE id = p_budget_id;

  -- Calculate balance from ledger entries
  SELECT COALESCE(SUM(
    CASE
      WHEN entry_type IN ('fund', 'reserve') THEN amount
      WHEN entry_type IN ('release') THEN amount  -- release entries are negative
      ELSE 0
    END
  ), 0) INTO v_calculated_balance
  FROM ledger_entries
  WHERE budget_id = p_budget_id;

  RETURN QUERY SELECT
    v_current_balance,
    v_calculated_balance,
    v_current_balance - v_calculated_balance as discrepancy;
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- COMPLETION
-- =============================================================================

DO $$
BEGIN
  RAISE NOTICE 'Database functions created successfully';
  RAISE NOTICE 'Budget functions:';
  RAISE NOTICE '  - check_budget_capacity(budget_id, amount)';
  RAISE NOTICE '  - reserve_budget(tenant_id, budget_id, amount, currency, ref_id)';
  RAISE NOTICE '  - charge_budget(tenant_id, budget_id, amount, currency, ref_id)';
  RAISE NOTICE '  - release_budget(tenant_id, budget_id, amount, currency, ref_id)';
  RAISE NOTICE '  - fund_budget(tenant_id, budget_id, amount, currency)';
  RAISE NOTICE '  - get_budget_utilization(budget_id)';
  RAISE NOTICE 'Issuance functions:';
  RAISE NOTICE '  - get_customer_rule_issuance_count(tenant_id, customer_id, rule_id)';
  RAISE NOTICE '  - get_rule_global_issuance_count(tenant_id, rule_id)';
  RAISE NOTICE '  - is_within_cooldown(tenant_id, customer_id, rule_id, cooldown_seconds)';
  RAISE NOTICE 'Expiry functions:';
  RAISE NOTICE '  - expire_old_issuances()';
  RAISE NOTICE 'Analytics functions:';
  RAISE NOTICE '  - get_redemption_rate(tenant_id, from_date, to_date)';
  RAISE NOTICE '  - reconcile_budget(budget_id)';
END $$;
