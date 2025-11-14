package integration

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/rules"
	"github.com/bmachimbira/loyalty/api/internal/testutil"
)

func TestIdempotency_DuplicateEvent(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	ctx := context.Background()

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID)
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID)

	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Create first event with idempotency key
	idempotencyKey := "test-idempotency-key-123"
	event1 := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithIdempotencyKey(idempotencyKey),
	)

	// Process first event
	issuances1, err := engine.ProcessEvent(ctx, event1)
	require.NoError(t, err)
	assert.Len(t, issuances1, 1, "First event should create issuance")

	// Try to create duplicate event with same idempotency key
	// This should fail at the database level due to unique constraint
	_, err = queries.CreateEvent(ctx, db.CreateEventParams{
		TenantID:       tenant.ID,
		CustomerID:     customer.ID,
		EventType:      "purchase",
		Properties:     []byte(`{"amount": 25.0}`),
		Source:         "api",
		IdempotencyKey: idempotencyKey,
	})
	assert.Error(t, err, "Duplicate event creation should fail")

	// Verify only one event exists
	events, err := queries.ListEvents(ctx, db.ListEventsParams{
		TenantID: tenant.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	// Count events with our idempotency key
	count := 0
	for _, event := range events {
		if event.IdempotencyKey == idempotencyKey {
			count++
		}
	}
	assert.Equal(t, 1, count, "Should only have one event with the idempotency key")

	// Verify only one issuance exists
	issuances, err := queries.ListIssuances(ctx, db.ListIssuancesParams{
		TenantID: tenant.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	// Count issuances for the event
	issuanceCount := 0
	for _, issuance := range issuances {
		if issuance.EventID == event1.ID {
			issuanceCount++
		}
	}
	assert.Equal(t, 1, issuanceCount, "Should only have one issuance for the event")
}

func TestIdempotency_ProcessingSameEventTwice(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	ctx := context.Background()

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID)
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID)

	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Create event
	event := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID)

	// Process event first time
	issuances1, err := engine.ProcessEvent(ctx, event)
	require.NoError(t, err)
	assert.Len(t, issuances1, 1, "First processing should create issuance")

	issuanceID1 := issuances1[0].ID

	// Process same event second time
	// The engine should detect the event was already processed
	issuances2, err := engine.ProcessEvent(ctx, event)
	require.NoError(t, err)

	// Should return existing issuance or empty (depending on implementation)
	if len(issuances2) > 0 {
		assert.Equal(t, issuanceID1, issuances2[0].ID, "Should return same issuance")
	}

	// Verify only one issuance exists in database
	allIssuances, err := queries.ListIssuances(ctx, db.ListIssuancesParams{
		TenantID: tenant.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	count := 0
	for _, issuance := range allIssuances {
		if issuance.EventID == event.ID {
			count++
		}
	}
	assert.Equal(t, 1, count, "Should only have one issuance for the event")
}

func TestIdempotency_DifferentKeys(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	ctx := context.Background()

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID)
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID)

	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Create first event with key1
	event1 := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithIdempotencyKey("key-1"),
	)
	issuances1, err := engine.ProcessEvent(ctx, event1)
	require.NoError(t, err)
	assert.Len(t, issuances1, 1)

	// Create second event with key2
	event2 := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithIdempotencyKey("key-2"),
	)
	issuances2, err := engine.ProcessEvent(ctx, event2)
	require.NoError(t, err)
	assert.Len(t, issuances2, 1)

	// Verify both events were processed
	events, err := queries.ListEvents(ctx, db.ListEventsParams{
		TenantID: tenant.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 2, "Should have at least 2 events")

	// Verify both issuances exist
	allIssuances, err := queries.ListIssuances(ctx, db.ListIssuancesParams{
		TenantID: tenant.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(allIssuances), 2, "Should have at least 2 issuances")
}

func TestIdempotency_SameKeyDifferentTenants(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	ctx := context.Background()

	// Create two tenants
	tenant1 := testutil.CreateTestTenant(t, queries, testutil.WithTenantName("Tenant 1"))
	tenant2 := testutil.CreateTestTenant(t, queries, testutil.WithTenantName("Tenant 2"))

	// Create customers, budgets, campaigns, rewards, rules for both tenants
	customer1 := testutil.CreateTestCustomer(t, queries, tenant1.ID)
	customer2 := testutil.CreateTestCustomer(t, queries, tenant2.ID)

	budget1 := testutil.CreateTestBudget(t, queries, tenant1.ID)
	budget2 := testutil.CreateTestBudget(t, queries, tenant2.ID)

	campaign1 := testutil.CreateTestCampaign(t, queries, tenant1.ID, budget1.ID)
	campaign2 := testutil.CreateTestCampaign(t, queries, tenant2.ID, budget2.ID)

	reward1 := testutil.CreateTestReward(t, queries, tenant1.ID)
	reward2 := testutil.CreateTestReward(t, queries, tenant2.ID)

	rule1 := testutil.CreateTestRule(t, queries, tenant1.ID, reward1.ID)
	rule2 := testutil.CreateTestRule(t, queries, tenant2.ID, reward2.ID)

	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule1.ID,
		Name:       rule1.Name,
		EventType:  rule1.EventType,
		Conditions: rule1.Conditions,
		Status:     rule1.Status,
		CampaignID: campaign1.ID,
	})
	require.NoError(t, err)

	_, err = queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule2.ID,
		Name:       rule2.Name,
		EventType:  rule2.EventType,
		Conditions: rule2.Conditions,
		Status:     rule2.Status,
		CampaignID: campaign2.ID,
	})
	require.NoError(t, err)

	// Use same idempotency key for both tenants
	idempotencyKey := "shared-key-123"

	// Create event for tenant1
	event1 := testutil.CreateTestEvent(t, queries, tenant1.ID, customer1.ID,
		testutil.WithIdempotencyKey(idempotencyKey),
	)
	issuances1, err := engine.ProcessEvent(ctx, event1)
	require.NoError(t, err)
	assert.Len(t, issuances1, 1)

	// Create event for tenant2 with same idempotency key
	// This should succeed because idempotency is scoped per tenant
	event2 := testutil.CreateTestEvent(t, queries, tenant2.ID, customer2.ID,
		testutil.WithIdempotencyKey(idempotencyKey),
	)
	issuances2, err := engine.ProcessEvent(ctx, event2)
	require.NoError(t, err)
	assert.Len(t, issuances2, 1, "Same idempotency key should work for different tenants")

	// Verify both events exist
	events, err := queries.ListEvents(ctx, db.ListEventsParams{
		TenantID: tenant1.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 1, "Tenant 1 should have events")

	events, err = queries.ListEvents(ctx, db.ListEventsParams{
		TenantID: tenant2.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 1, "Tenant 2 should have events")
}
