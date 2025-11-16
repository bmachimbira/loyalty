-- Seed data for development and testing
-- Version: 1.0
-- Date: 2025-11-14

-- =============================================================================
-- DEMO TENANT
-- =============================================================================

-- Create demo tenant
INSERT INTO tenants (id, name, country_code, default_ccy, theme) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Demo Shop Zimbabwe', 'ZW', 'ZWG',
   '{"logo_url": "/logo.png", "primary_color": "#1a73e8", "secondary_color": "#fbbc04"}'::jsonb);

-- =============================================================================
-- STAFF USERS
-- =============================================================================
-- Password for all demo users: "password123" (bcrypt hash)
-- In production, use proper password hashing in application code

INSERT INTO staff_users (tenant_id, email, full_name, role, pwd_hash) VALUES
  -- Owner: full access
  ('00000000-0000-0000-0000-000000000001', 'owner@demoshop.zw', 'Demo Owner', 'owner',
   '$2a$10$Bu.s4sJvQcW9N.ta0oYb0OGBcg1Zcw9H3Djgj6Yk7chtHuAT1WXoG'),

  -- Admin: manage rules, budgets, rewards
  ('00000000-0000-0000-0000-000000000001', 'admin@demoshop.zw', 'Demo Admin', 'admin',
   '$2a$10$Bu.s4sJvQcW9N.ta0oYb0OGBcg1Zcw9H3Djgj6Yk7chtHuAT1WXoG'),

  -- Staff: basic operations (redeem, view customers)
  ('00000000-0000-0000-0000-000000000001', 'staff@demoshop.zw', 'Demo Staff', 'staff',
   '$2a$10$Bu.s4sJvQcW9N.ta0oYb0OGBcg1Zcw9H3Djgj6Yk7chtHuAT1WXoG'),

  -- Viewer: read-only access
  ('00000000-0000-0000-0000-000000000001', 'viewer@demoshop.zw', 'Demo Viewer', 'viewer',
   '$2a$10$Bu.s4sJvQcW9N.ta0oYb0OGBcg1Zcw9H3Djgj6Yk7chtHuAT1WXoG');

-- =============================================================================
-- SAMPLE CUSTOMERS
-- =============================================================================

INSERT INTO customers (id, tenant_id, phone_e164, external_ref, status) VALUES
  ('10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '+263771234001', 'CUST-001', 'active'),
  ('10000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '+263771234002', 'CUST-002', 'active'),
  ('10000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '+263771234003', 'CUST-003', 'active'),
  ('10000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '+263771234004', 'CUST-004', 'active'),
  ('10000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', '+263771234005', 'CUST-005', 'active'),
  ('10000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', '+263771234006', 'CUST-006', 'active'),
  ('10000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', '+263771234007', 'CUST-007', 'active'),
  ('10000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', '+263771234008', 'CUST-008', 'active'),
  ('10000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', '+263771234009', 'CUST-009', 'active'),
  ('10000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', '+263771234010', 'CUST-010', 'active');

-- =============================================================================
-- CUSTOMER CONSENTS
-- =============================================================================

-- Seed consents for WhatsApp marketing
INSERT INTO consents (tenant_id, customer_id, channel, purpose, granted, occurred_at) VALUES
  ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'whatsapp', 'loyalty', true, now() - interval '10 days'),
  ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002', 'whatsapp', 'loyalty', true, now() - interval '9 days'),
  ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000003', 'whatsapp', 'loyalty', true, now() - interval '8 days'),
  ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000004', 'whatsapp', 'loyalty', true, now() - interval '7 days'),
  ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000005', 'whatsapp', 'loyalty', true, now() - interval '6 days'),
  ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000006', 'whatsapp', 'marketing', true, now() - interval '5 days'),
  ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000007', 'whatsapp', 'marketing', true, now() - interval '4 days'),
  ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000008', 'whatsapp', 'marketing', false, now() - interval '3 days'),
  ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000009', 'whatsapp', 'loyalty', true, now() - interval '2 days'),
  ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000010', 'whatsapp', 'loyalty', true, now() - interval '1 day');

-- =============================================================================
-- BUDGETS
-- =============================================================================

INSERT INTO budgets (id, tenant_id, name, currency, soft_cap, hard_cap, balance, period) VALUES
  -- ZWG budgets
  ('20000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001',
   'ZWG Monthly Budget', 'ZWG', 50000.00, 75000.00, 10000.00, 'monthly'),

  ('20000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001',
   'ZWG Rolling Budget', 'ZWG', 25000.00, 30000.00, 5000.00, 'rolling'),

  -- USD budgets
  ('20000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001',
   'USD Monthly Budget', 'USD', 1000.00, 1500.00, 250.00, 'monthly'),

  ('20000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001',
   'USD Rolling Budget', 'USD', 500.00, 750.00, 100.00, 'rolling');

-- Initial funding ledger entries
INSERT INTO ledger_entries (tenant_id, budget_id, entry_type, currency, amount, ref_type, ref_id) VALUES
  ('00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', 'fund', 'ZWG', 10000.00, 'topup', NULL),
  ('00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002', 'fund', 'ZWG', 5000.00, 'topup', NULL),
  ('00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000003', 'fund', 'USD', 250.00, 'topup', NULL),
  ('00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000004', 'fund', 'USD', 100.00, 'topup', NULL);

-- =============================================================================
-- REWARD CATALOG
-- =============================================================================

INSERT INTO reward_catalog (id, tenant_id, name, type, face_value, currency, inventory, metadata, active) VALUES
  -- Discount rewards
  ('30000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001',
   'ZWG 5 Discount', 'discount', 5.00, 'ZWG', 'none',
   '{"discount_type": "amount", "amount": 5.00, "currency": "ZWG", "min_basket": 20.00, "valid_days": 7}'::jsonb, true),

  ('30000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001',
   '10% Off Next Purchase', 'discount', NULL, 'ZWG', 'none',
   '{"discount_type": "percent", "percent": 10, "min_basket": 50.00, "valid_days": 14}'::jsonb, true),

  -- External vouchers (airtime/data)
  ('30000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001',
   '200MB Data Bundle', 'external_voucher', 2.00, 'USD', 'jit_external',
   '{"provider": "airtime_provider", "product_id": "DATA_200MB", "validity_days": 30}'::jsonb, true),

  ('30000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001',
   'USD 1 Airtime', 'external_voucher', 1.00, 'USD', 'jit_external',
   '{"provider": "airtime_provider", "product_id": "AIRTIME_1USD", "validity_days": 90}'::jsonb, true),

  -- Voucher codes (from pool)
  ('30000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001',
   'Free Coffee Voucher', 'voucher_code', 3.50, 'USD', 'pool',
   '{"partner": "Coffee Shop", "instructions": "Present code at counter", "valid_days": 30}'::jsonb, true),

  -- Points credit
  ('30000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001',
   '100 Loyalty Points', 'points_credit', 100.00, NULL, 'none',
   '{"points": 100, "description": "Redeem for rewards in catalog"}'::jsonb, true),

  -- Physical item
  ('30000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001',
   'Free T-Shirt', 'physical_item', 15.00, 'USD', 'none',
   '{"size_options": ["S", "M", "L", "XL"], "fulfillment": "pickup_in_store"}'::jsonb, true);

-- =============================================================================
-- CAMPAIGNS
-- =============================================================================

INSERT INTO campaigns (id, tenant_id, name, start_at, end_at, budget_id, status) VALUES
  -- Active campaign with ZWG budget
  ('40000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001',
   'November Welcome Campaign', now() - interval '5 days', now() + interval '25 days',
   '20000000-0000-0000-0000-000000000001', 'active'),

  -- Active campaign with USD budget
  ('40000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001',
   'High Spender Rewards', now() - interval '3 days', now() + interval '27 days',
   '20000000-0000-0000-0000-000000000003', 'active'),

  -- Planned campaign
  ('40000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001',
   'December Holiday Special', now() + interval '10 days', now() + interval '40 days',
   '20000000-0000-0000-0000-000000000002', 'active');

-- =============================================================================
-- RULES
-- =============================================================================

INSERT INTO rules (id, tenant_id, campaign_id, name, event_type, conditions, reward_id, per_user_cap, global_cap, cool_down_sec, active) VALUES
  -- Rule 1: Purchase >= ZWG 20 → ZWG 5 discount
  ('50000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001',
   '40000000-0000-0000-0000-000000000001',
   'Spend ZWG 20+ get ZWG 5 off', 'purchase',
   '{"all": [{">=": [{"var": "properties.amount"}, 20.00]}, {"==": [{"var": "properties.currency"}, "ZWG"]}]}'::jsonb,
   '30000000-0000-0000-0000-000000000001', 3, 1000, 86400, true),

  -- Rule 2: Purchase >= USD 10 → 200MB data
  ('50000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001',
   '40000000-0000-0000-0000-000000000002',
   'Spend USD 10+ get 200MB data', 'purchase',
   '{"all": [{">=": [{"var": "properties.amount"}, 10.00]}, {"==": [{"var": "properties.currency"}, "USD"]}]}'::jsonb,
   '30000000-0000-0000-0000-000000000003', 2, 500, 604800, true),

  -- Rule 3: First purchase → Welcome discount
  ('50000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001',
   '40000000-0000-0000-0000-000000000001',
   'First Purchase Welcome', 'purchase',
   '{"all": [{">=": [{"var": "properties.amount"}, 5.00]}]}'::jsonb,
   '30000000-0000-0000-0000-000000000002', 1, NULL, 0, true),

  -- Rule 4: Store visit → Points
  ('50000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001',
   '40000000-0000-0000-0000-000000000001',
   'Visit Store Get 100 Points', 'visit',
   '{}'::jsonb,
   '30000000-0000-0000-0000-000000000006', 10, NULL, 43200, true),

  -- Rule 5: Referral → Reward for referrer
  ('50000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001',
   '40000000-0000-0000-0000-000000000002',
   'Refer a Friend', 'referral',
   '{"all": [{"==": [{"var": "properties.referral_type"}, "completed"]}]}'::jsonb,
   '30000000-0000-0000-0000-000000000004', 5, 100, 0, true);

-- =============================================================================
-- SAMPLE EVENTS
-- =============================================================================

-- Create some sample purchase events
INSERT INTO events (id, tenant_id, customer_id, event_type, properties, occurred_at, source, idempotency_key) VALUES
  -- Recent purchases
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001',
   'purchase', '{"amount": 25.50, "currency": "ZWG", "receipt_id": "R-001"}'::jsonb,
   now() - interval '2 hours', 'pos', 'DEMO-EVENT-001'),

  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002',
   'purchase', '{"amount": 15.00, "currency": "USD", "receipt_id": "R-002"}'::jsonb,
   now() - interval '4 hours', 'pos', 'DEMO-EVENT-002'),

  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000003',
   'purchase', '{"amount": 50.00, "currency": "ZWG", "receipt_id": "R-003"}'::jsonb,
   now() - interval '1 day', 'pos', 'DEMO-EVENT-003'),

  -- Store visits
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000004',
   'visit', '{}'::jsonb,
   now() - interval '3 hours', 'whatsapp', 'DEMO-EVENT-004'),

  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000005',
   'visit', '{}'::jsonb,
   now() - interval '5 hours', 'ussd', 'DEMO-EVENT-005');

-- =============================================================================
-- SAMPLE ISSUANCES
-- =============================================================================

-- Create sample issuances in various states
INSERT INTO issuances (id, tenant_id, customer_id, campaign_id, reward_id, status, code, currency, cost_amount, face_amount, issued_at, expires_at) VALUES
  -- Issued rewards
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001',
   '40000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001',
   'issued', 'DISC-' || substr(md5(random()::text), 1, 8), 'ZWG', 5.00, 5.00,
   now() - interval '2 hours', now() + interval '5 days'),

  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002',
   '40000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000003',
   'issued', 'DATA-' || substr(md5(random()::text), 1, 10), 'USD', 2.00, 2.00,
   now() - interval '4 hours', now() + interval '28 days'),

  -- Redeemed reward
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000003',
   '40000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001',
   'redeemed', 'DISC-' || substr(md5(random()::text), 1, 8), 'ZWG', 5.00, 5.00,
   now() - interval '1 day', now() + interval '6 days', now() - interval '12 hours'),

  -- Reserved (pending)
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000004',
   '40000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000006',
   'reserved', NULL, NULL, 0.50, 100.00,
   now() - interval '10 minutes', NULL);

-- =============================================================================
-- SAMPLE AUDIT LOGS
-- =============================================================================

INSERT INTO audit_logs (tenant_id, actor_type, actor_id, action, resource_type, resource_id, details, ip_address) VALUES
  ('00000000-0000-0000-0000-000000000001', 'staff', NULL, 'login', 'staff_user', NULL,
   '{"email": "admin@demoshop.zw"}'::jsonb, '192.168.1.1'::inet),

  ('00000000-0000-0000-0000-000000000001', 'staff', NULL, 'create', 'rule', '50000000-0000-0000-0000-000000000001',
   '{"rule_name": "Spend ZWG 20+ get ZWG 5 off"}'::jsonb, '192.168.1.1'::inet),

  ('00000000-0000-0000-0000-000000000001', 'staff', NULL, 'create', 'budget', '20000000-0000-0000-0000-000000000001',
   '{"budget_name": "ZWG Monthly Budget", "amount": 10000.00}'::jsonb, '192.168.1.1'::inet);

-- =============================================================================
-- COMPLETION
-- =============================================================================

-- Log seed completion
DO $$
BEGIN
  RAISE NOTICE 'Seed data created successfully';
  RAISE NOTICE '  - Tenant: Demo Shop Zimbabwe (ID: 00000000-0000-0000-0000-000000000001)';
  RAISE NOTICE '  - Staff users: 4 (owner, admin, staff, viewer)';
  RAISE NOTICE '  - Customers: 10';
  RAISE NOTICE '  - Budgets: 4 (2 ZWG, 2 USD)';
  RAISE NOTICE '  - Rewards: 7';
  RAISE NOTICE '  - Campaigns: 3';
  RAISE NOTICE '  - Rules: 5';
  RAISE NOTICE '  - Sample events: 5';
  RAISE NOTICE '  - Sample issuances: 4';
END $$;
