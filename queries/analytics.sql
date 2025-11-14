-- Analytics queries
-- sqlc query file for dashboard and reporting

-- name: GetDashboardStats :one
SELECT
  (SELECT COUNT(*) FROM customers WHERE customers.tenant_id = $1 AND status = 'active') as active_customers,
  (SELECT COUNT(*) FROM events WHERE events.tenant_id = $1 AND occurred_at >= $2) as events_today,
  (SELECT COUNT(*) FROM issuances WHERE issuances.tenant_id = $1 AND issued_at >= $2) as rewards_issued_today,
  (SELECT COUNT(*) FROM issuances WHERE issuances.tenant_id = $1 AND status = 'redeemed' AND redeemed_at >= $2) as rewards_redeemed_today;

-- name: GetRedemptionRate :one
SELECT
  COUNT(*) FILTER (WHERE status IN ('issued', 'redeemed')) as issued,
  COUNT(*) FILTER (WHERE status = 'redeemed') as redeemed,
  CASE
    WHEN COUNT(*) FILTER (WHERE status IN ('issued', 'redeemed')) = 0 THEN 0
    ELSE (COUNT(*) FILTER (WHERE status = 'redeemed')::float / COUNT(*) FILTER (WHERE status IN ('issued', 'redeemed'))::float) * 100
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
  COALESCE(SUM(i.cost_amount), 0) as total_cost
FROM reward_catalog r
LEFT JOIN issuances i ON i.reward_id = r.id
  AND i.tenant_id = $1
  AND i.issued_at >= $2
  AND i.issued_at <= $3
WHERE r.tenant_id = $1
GROUP BY r.id, r.name, r.type
ORDER BY issuance_count DESC
LIMIT $4;

-- name: GetIssuancesByDateRange :many
SELECT
  DATE(issued_at) as issue_date,
  COUNT(*) as issuance_count,
  COALESCE(SUM(cost_amount), 0) as total_cost
FROM issuances
WHERE tenant_id = $1
  AND issued_at >= $2
  AND issued_at <= $3
GROUP BY DATE(issued_at)
ORDER BY issue_date DESC;

-- name: GetIssuancesByStatus :one
SELECT
  COUNT(*) FILTER (WHERE status = 'reserved') as reserved,
  COUNT(*) FILTER (WHERE status = 'issued') as issued,
  COUNT(*) FILTER (WHERE status = 'redeemed') as redeemed,
  COUNT(*) FILTER (WHERE status = 'expired') as expired,
  COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled,
  COUNT(*) FILTER (WHERE status = 'failed') as failed
FROM issuances
WHERE tenant_id = $1
  AND issued_at >= $2
  AND issued_at <= $3;

-- name: GetCustomerActivity :many
SELECT
  c.id,
  c.phone_e164,
  c.external_ref,
  COUNT(DISTINCT e.id) as event_count,
  COUNT(DISTINCT i.id) as reward_count,
  MAX(e.occurred_at) as last_event_at
FROM customers c
LEFT JOIN events e ON e.customer_id = c.id AND e.tenant_id = $1
LEFT JOIN issuances i ON i.customer_id = c.id AND i.tenant_id = $1
WHERE c.tenant_id = $1
GROUP BY c.id, c.phone_e164, c.external_ref
ORDER BY event_count DESC
LIMIT $2 OFFSET $3;

-- name: GetCampaignPerformance :many
SELECT
  c.id,
  c.name,
  COUNT(i.id) as issuance_count,
  COUNT(i.id) FILTER (WHERE i.status = 'redeemed') as redeemed_count,
  COALESCE(SUM(i.cost_amount), 0) as total_cost,
  CASE
    WHEN COUNT(i.id) = 0 THEN 0
    ELSE (COUNT(i.id) FILTER (WHERE i.status = 'redeemed')::float / COUNT(i.id)::float) * 100
  END as redemption_rate
FROM campaigns c
LEFT JOIN issuances i ON i.campaign_id = c.id
  AND i.tenant_id = $1
  AND i.issued_at >= $2
  AND i.issued_at <= $3
WHERE c.tenant_id = $1
GROUP BY c.id, c.name
ORDER BY issuance_count DESC;

-- name: GetBudgetUtilization :many
SELECT
  b.id,
  b.name,
  b.currency,
  b.balance,
  b.soft_cap,
  b.hard_cap,
  CASE
    WHEN b.hard_cap = 0 THEN 0
    ELSE (b.balance / b.hard_cap) * 100
  END as utilization_percent,
  COALESCE(SUM(l.amount) FILTER (WHERE l.entry_type = 'reserve'), 0) as total_reserved,
  COALESCE(SUM(l.amount) FILTER (WHERE l.entry_type = 'charge'), 0) as total_charged
FROM budgets b
LEFT JOIN ledger_entries l ON l.budget_id = b.id
  AND l.tenant_id = $1
  AND l.created_at >= $2
  AND l.created_at <= $3
WHERE b.tenant_id = $1
GROUP BY b.id, b.name, b.currency, b.balance, b.soft_cap, b.hard_cap
ORDER BY utilization_percent DESC;

-- name: GetEventsByType :many
SELECT
  event_type,
  COUNT(*) as event_count,
  COUNT(DISTINCT customer_id) as unique_customers
FROM events
WHERE tenant_id = $1
  AND occurred_at >= $2
  AND occurred_at <= $3
GROUP BY event_type
ORDER BY event_count DESC;

-- name: GetRuleEffectiveness :many
SELECT
  r.id,
  r.name,
  r.event_type,
  COUNT(i.id) as issuance_count,
  r.per_user_cap,
  r.global_cap,
  CASE
    WHEN r.global_cap IS NULL THEN 0
    ELSE (COUNT(i.id)::float / r.global_cap::float) * 100
  END as cap_utilization_percent
FROM rules r
LEFT JOIN campaigns c ON c.id = r.campaign_id
LEFT JOIN issuances i ON i.campaign_id = c.id
  AND i.tenant_id = $1
  AND i.issued_at >= $2
  AND i.issued_at <= $3
WHERE r.tenant_id = $1
  AND r.active = true
GROUP BY r.id, r.name, r.event_type, r.per_user_cap, r.global_cap
ORDER BY issuance_count DESC;
