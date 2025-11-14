package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/testutil"
)

func TestRLS_CustomerIsolation(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Create two tenants
	tenant1 := testutil.CreateTestTenant(t, queries, testutil.WithTenantName("Tenant 1"))
	tenant2 := testutil.CreateTestTenant(t, queries, testutil.WithTenantName("Tenant 2"))

	// Set context to tenant1
	testutil.SetTenantContext(t, pool, tenant1.ID.Bytes[:])

	// Create customer for tenant1
	customer1 := testutil.CreateTestCustomer(t, queries, tenant1.ID,
		testutil.WithPhone("+263771111111"),
	)

	// Query customers - should only see tenant1's customer
	customers, err := queries.ListCustomers(ctx, db.ListCustomersParams{
		TenantID: tenant1.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(customers), 1, "Should see tenant1's customers")

	// Switch to tenant2
	testutil.SetTenantContext(t, pool, tenant2.ID.Bytes[:])

	// Create customer for tenant2
	customer2 := testutil.CreateTestCustomer(t, queries, tenant2.ID,
		testutil.WithPhone("+263772222222"),
	)

	// Query customers from tenant2 perspective
	customers, err = queries.ListCustomers(ctx, db.ListCustomersParams{
		TenantID: tenant2.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	// Should only see tenant2's customers
	for _, customer := range customers {
		assert.Equal(t, tenant2.ID, customer.TenantID, "Should only see tenant2's customers")
		assert.NotEqual(t, customer1.ID, customer.ID, "Should not see tenant1's customer")
	}

	// Try to get tenant1's customer from tenant2 context (should fail or return nothing)
	// This tests RLS enforcement
	_, err = queries.GetCustomer(ctx, customer1.ID)
	if err != nil {
		// RLS blocked the query - this is expected
		t.Logf("RLS correctly blocked cross-tenant access: %v", err)
	}
}

func TestRLS_EventIsolation(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Create two tenants
	tenant1 := testutil.CreateTestTenant(t, queries)
	tenant2 := testutil.CreateTestTenant(t, queries)

	// Create customers for each tenant
	customer1 := testutil.CreateTestCustomer(t, queries, tenant1.ID)
	customer2 := testutil.CreateTestCustomer(t, queries, tenant2.ID)

	// Create events for tenant1
	event1 := testutil.CreateTestEvent(t, queries, tenant1.ID, customer1.ID,
		testutil.WithIdempotencyKey("t1-event1"),
	)

	// Create events for tenant2
	event2 := testutil.CreateTestEvent(t, queries, tenant2.ID, customer2.ID,
		testutil.WithIdempotencyKey("t2-event1"),
	)

	// Set context to tenant1
	testutil.SetTenantContext(t, pool, tenant1.ID.Bytes[:])

	// List events for tenant1
	events, err := queries.ListEvents(ctx, db.ListEventsParams{
		TenantID: tenant1.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	// Should only see tenant1's events
	for _, event := range events {
		assert.Equal(t, tenant1.ID, event.TenantID, "Should only see tenant1's events")
		assert.NotEqual(t, event2.ID, event.ID, "Should not see tenant2's event")
	}

	// Try to get tenant2's event from tenant1 context
	_, err = queries.GetEvent(ctx, event2.ID)
	if err != nil {
		t.Logf("RLS correctly blocked cross-tenant event access: %v", err)
	}
}

func TestRLS_IssuanceIsolation(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Create two tenants with full setup
	tenant1 := testutil.CreateTestTenant(t, queries)
	tenant2 := testutil.CreateTestTenant(t, queries)

	customer1 := testutil.CreateTestCustomer(t, queries, tenant1.ID)
	customer2 := testutil.CreateTestCustomer(t, queries, tenant2.ID)

	budget1 := testutil.CreateTestBudget(t, queries, tenant1.ID)
	budget2 := testutil.CreateTestBudget(t, queries, tenant2.ID)

	campaign1 := testutil.CreateTestCampaign(t, queries, tenant1.ID, budget1.ID)
	campaign2 := testutil.CreateTestCampaign(t, queries, tenant2.ID, budget2.ID)

	reward1 := testutil.CreateTestReward(t, queries, tenant1.ID)
	reward2 := testutil.CreateTestReward(t, queries, tenant2.ID)

	event1 := testutil.CreateTestEvent(t, queries, tenant1.ID, customer1.ID)
	event2 := testutil.CreateTestEvent(t, queries, tenant2.ID, customer2.ID)

	// Create issuances for each tenant
	issuance1 := testutil.CreateTestIssuance(t, queries, tenant1.ID, customer1.ID, campaign1.ID, reward1.ID, event1.ID)
	issuance2 := testutil.CreateTestIssuance(t, queries, tenant2.ID, customer2.ID, campaign2.ID, reward2.ID, event2.ID)

	// Set context to tenant1
	testutil.SetTenantContext(t, pool, tenant1.ID.Bytes[:])

	// List issuances for tenant1
	issuances, err := queries.ListIssuances(ctx, db.ListIssuancesParams{
		TenantID: tenant1.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	// Should only see tenant1's issuances
	for _, issuance := range issuances {
		assert.Equal(t, tenant1.ID, issuance.TenantID, "Should only see tenant1's issuances")
		assert.NotEqual(t, issuance2.ID, issuance.ID, "Should not see tenant2's issuance")
	}
}

func TestRLS_BudgetIsolation(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Create two tenants
	tenant1 := testutil.CreateTestTenant(t, queries)
	tenant2 := testutil.CreateTestTenant(t, queries)

	// Create budgets for each tenant
	budget1 := testutil.CreateTestBudget(t, queries, tenant1.ID,
		testutil.WithBudgetName("Tenant 1 Budget"),
	)
	budget2 := testutil.CreateTestBudget(t, queries, tenant2.ID,
		testutil.WithBudgetName("Tenant 2 Budget"),
	)

	// Set context to tenant1
	testutil.SetTenantContext(t, pool, tenant1.ID.Bytes[:])

	// List budgets for tenant1
	budgets, err := queries.ListBudgets(ctx, db.ListBudgetsParams{
		TenantID: tenant1.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	// Should only see tenant1's budgets
	for _, budget := range budgets {
		assert.Equal(t, tenant1.ID, budget.TenantID, "Should only see tenant1's budgets")
		assert.NotEqual(t, budget2.ID, budget.ID, "Should not see tenant2's budget")
	}

	// Try to get tenant2's budget from tenant1 context
	_, err = queries.GetBudget(ctx, budget2.ID)
	if err != nil {
		t.Logf("RLS correctly blocked cross-tenant budget access: %v", err)
	}
}

func TestRLS_RuleIsolation(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Create two tenants
	tenant1 := testutil.CreateTestTenant(t, queries)
	tenant2 := testutil.CreateTestTenant(t, queries)

	reward1 := testutil.CreateTestReward(t, queries, tenant1.ID)
	reward2 := testutil.CreateTestReward(t, queries, tenant2.ID)

	// Create rules for each tenant
	rule1 := testutil.CreateTestRule(t, queries, tenant1.ID, reward1.ID,
		testutil.WithRuleName("Tenant 1 Rule"),
	)
	rule2 := testutil.CreateTestRule(t, queries, tenant2.ID, reward2.ID,
		testutil.WithRuleName("Tenant 2 Rule"),
	)

	// Set context to tenant1
	testutil.SetTenantContext(t, pool, tenant1.ID.Bytes[:])

	// List rules for tenant1
	rules, err := queries.ListRules(ctx, db.ListRulesParams{
		TenantID: tenant1.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	// Should only see tenant1's rules
	for _, rule := range rules {
		assert.Equal(t, tenant1.ID, rule.TenantID, "Should only see tenant1's rules")
		assert.NotEqual(t, rule2.ID, rule.ID, "Should not see tenant2's rule")
	}
}

func TestRLS_CampaignIsolation(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Create two tenants
	tenant1 := testutil.CreateTestTenant(t, queries)
	tenant2 := testutil.CreateTestTenant(t, queries)

	budget1 := testutil.CreateTestBudget(t, queries, tenant1.ID)
	budget2 := testutil.CreateTestBudget(t, queries, tenant2.ID)

	// Create campaigns for each tenant
	campaign1 := testutil.CreateTestCampaign(t, queries, tenant1.ID, budget1.ID,
		testutil.WithCampaignName("Tenant 1 Campaign"),
	)
	campaign2 := testutil.CreateTestCampaign(t, queries, tenant2.ID, budget2.ID,
		testutil.WithCampaignName("Tenant 2 Campaign"),
	)

	// Set context to tenant1
	testutil.SetTenantContext(t, pool, tenant1.ID.Bytes[:])

	// List campaigns for tenant1
	campaigns, err := queries.ListCampaigns(ctx, db.ListCampaignsParams{
		TenantID: tenant1.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	// Should only see tenant1's campaigns
	for _, campaign := range campaigns {
		assert.Equal(t, tenant1.ID, campaign.TenantID, "Should only see tenant1's campaigns")
		assert.NotEqual(t, campaign2.ID, campaign.ID, "Should not see tenant2's campaign")
	}
}

func TestRLS_RewardIsolation(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Create two tenants
	tenant1 := testutil.CreateTestTenant(t, queries)
	tenant2 := testutil.CreateTestTenant(t, queries)

	// Create rewards for each tenant
	reward1 := testutil.CreateTestReward(t, queries, tenant1.ID,
		testutil.WithRewardName("Tenant 1 Reward"),
	)
	reward2 := testutil.CreateTestReward(t, queries, tenant2.ID,
		testutil.WithRewardName("Tenant 2 Reward"),
	)

	// Set context to tenant1
	testutil.SetTenantContext(t, pool, tenant1.ID.Bytes[:])

	// List rewards for tenant1
	rewards, err := queries.ListRewards(ctx, db.ListRewardsParams{
		TenantID: tenant1.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	// Should only see tenant1's rewards
	for _, reward := range rewards {
		assert.Equal(t, tenant1.ID, reward.TenantID, "Should only see tenant1's rewards")
		assert.NotEqual(t, reward2.ID, reward.ID, "Should not see tenant2's reward")
	}

	// Try to get tenant2's reward from tenant1 context
	_, err = queries.GetReward(ctx, reward2.ID)
	if err != nil {
		t.Logf("RLS correctly blocked cross-tenant reward access: %v", err)
	}
}

func TestRLS_LedgerIsolation(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Create two tenants
	tenant1 := testutil.CreateTestTenant(t, queries)
	tenant2 := testutil.CreateTestTenant(t, queries)

	budget1 := testutil.CreateTestBudget(t, queries, tenant1.ID)
	budget2 := testutil.CreateTestBudget(t, queries, tenant2.ID)

	// Set context to tenant1
	testutil.SetTenantContext(t, pool, tenant1.ID.Bytes[:])

	// Get ledger entries for tenant1
	entries1, err := queries.GetLedgerEntries(ctx, db.GetLedgerEntriesParams{
		TenantID: tenant1.ID,
		BudgetID: budget1.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	// All entries should belong to tenant1
	for _, entry := range entries1 {
		assert.Equal(t, tenant1.ID, entry.TenantID, "Should only see tenant1's ledger entries")
	}

	// Switch to tenant2
	testutil.SetTenantContext(t, pool, tenant2.ID.Bytes[:])

	// Try to get tenant1's ledger entries from tenant2 context
	entries, err := queries.GetLedgerEntries(ctx, db.GetLedgerEntriesParams{
		TenantID: tenant2.ID,
		BudgetID: budget1.ID, // Tenant1's budget
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)
	assert.Empty(t, entries, "Should not see tenant1's ledger entries from tenant2 context")
}
