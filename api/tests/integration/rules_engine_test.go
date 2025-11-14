package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bmachimbira/loyalty/api/internal/budget"
	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/rules"
	"github.com/bmachimbira/loyalty/api/internal/testutil"
)

func TestRulesEngine_SimpleCondition(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	// Setup: Create tenant, customer, budget, campaign, reward, rule
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID,
		testutil.WithRewardCost(10.0),
		testutil.WithRewardFace(10.0),
	)

	// Create rule: amount >= 20
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID,
		testutil.WithConditions(map[string]interface{}{
			">=": []interface{}{
				map[string]interface{}{"var": "amount"},
				20,
			},
		}),
	)

	// Update rule to link to campaign
	ctx := context.Background()
	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Create event with amount = 25
	event := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithProperties(map[string]interface{}{
			"amount":   25.0,
			"currency": "USD",
		}),
	)

	// Process event
	issuances, err := engine.ProcessEvent(ctx, event)

	// Assert
	require.NoError(t, err)
	assert.Len(t, issuances, 1, "Expected 1 issuance")
	assert.Equal(t, reward.ID, issuances[0].RewardID)
	assert.Equal(t, customer.ID, issuances[0].CustomerID)
	assert.Equal(t, event.ID, issuances[0].EventID)
	assert.Equal(t, "reserved", issuances[0].Status)
}

func TestRulesEngine_NoMatch(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID)

	// Create rule: amount >= 50
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID,
		testutil.WithConditions(map[string]interface{}{
			">=": []interface{}{
				map[string]interface{}{"var": "amount"},
				50,
			},
		}),
	)

	ctx := context.Background()
	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Create event with amount = 25 (less than 50)
	event := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithProperties(map[string]interface{}{
			"amount": 25.0,
		}),
	)

	// Process event
	issuances, err := engine.ProcessEvent(ctx, event)

	// Assert: No issuance because condition not met
	require.NoError(t, err)
	assert.Len(t, issuances, 0, "Expected no issuances")
}

func TestRulesEngine_PerUserCap(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID)

	// Create rule with per-user cap of 2
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID,
		testutil.WithPerUserCap(2),
	)

	ctx := context.Background()
	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Create and process first event
	event1 := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithIdempotencyKey("key1"),
	)
	issuances1, err := engine.ProcessEvent(ctx, event1)
	require.NoError(t, err)
	assert.Len(t, issuances1, 1, "First event should trigger issuance")

	// Create and process second event
	event2 := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithIdempotencyKey("key2"),
	)
	issuances2, err := engine.ProcessEvent(ctx, event2)
	require.NoError(t, err)
	assert.Len(t, issuances2, 1, "Second event should trigger issuance")

	// Create and process third event (should be blocked by cap)
	event3 := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithIdempotencyKey("key3"),
	)
	issuances3, err := engine.ProcessEvent(ctx, event3)
	require.NoError(t, err)
	assert.Len(t, issuances3, 0, "Third event should be blocked by per-user cap")
}

func TestRulesEngine_GlobalCap(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer1 := testutil.CreateTestCustomer(t, queries, tenant.ID,
		testutil.WithPhone("+263771111111"),
	)
	customer2 := testutil.CreateTestCustomer(t, queries, tenant.ID,
		testutil.WithPhone("+263772222222"),
	)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID)

	// Create rule with global cap of 2
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID,
		testutil.WithGlobalCap(2),
	)

	ctx := context.Background()
	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Customer 1, Event 1
	event1 := testutil.CreateTestEvent(t, queries, tenant.ID, customer1.ID,
		testutil.WithIdempotencyKey("c1-e1"),
	)
	issuances1, err := engine.ProcessEvent(ctx, event1)
	require.NoError(t, err)
	assert.Len(t, issuances1, 1, "Customer 1, Event 1 should trigger issuance")

	// Customer 2, Event 1
	event2 := testutil.CreateTestEvent(t, queries, tenant.ID, customer2.ID,
		testutil.WithIdempotencyKey("c2-e1"),
	)
	issuances2, err := engine.ProcessEvent(ctx, event2)
	require.NoError(t, err)
	assert.Len(t, issuances2, 1, "Customer 2, Event 1 should trigger issuance")

	// Customer 1, Event 2 (should be blocked by global cap)
	event3 := testutil.CreateTestEvent(t, queries, tenant.ID, customer1.ID,
		testutil.WithIdempotencyKey("c1-e2"),
	)
	issuances3, err := engine.ProcessEvent(ctx, event3)
	require.NoError(t, err)
	assert.Len(t, issuances3, 0, "Customer 1, Event 2 should be blocked by global cap")
}

func TestRulesEngine_Cooldown(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID)

	// Create rule with 24 hour cooldown
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID,
		testutil.WithCooldownHours(24),
	)

	ctx := context.Background()
	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// First event should succeed
	event1 := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithIdempotencyKey("cooldown-1"),
	)
	issuances1, err := engine.ProcessEvent(ctx, event1)
	require.NoError(t, err)
	assert.Len(t, issuances1, 1, "First event should trigger issuance")

	// Second event immediately after should be blocked
	event2 := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithIdempotencyKey("cooldown-2"),
	)
	issuances2, err := engine.ProcessEvent(ctx, event2)
	require.NoError(t, err)
	assert.Len(t, issuances2, 0, "Second event should be blocked by cooldown")
}

func TestRulesEngine_BudgetEnforcement(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	// Setup with small budget
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)

	// Create budget with only 15 USD (enough for 1 reward at 10 USD)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID,
		testutil.WithBudgetBalance(15.0),
		testutil.WithBudgetCaps(15.0, 15.0),
	)

	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID,
		testutil.WithRewardCost(10.0),
	)

	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID)

	ctx := context.Background()
	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// First event should succeed
	event1 := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithIdempotencyKey("budget-1"),
	)
	issuances1, err := engine.ProcessEvent(ctx, event1)
	require.NoError(t, err)
	assert.Len(t, issuances1, 1, "First event should trigger issuance")

	// Second event should fail due to insufficient budget
	event2 := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithIdempotencyKey("budget-2"),
	)
	issuances2, err := engine.ProcessEvent(ctx, event2)
	require.NoError(t, err)
	assert.Len(t, issuances2, 0, "Second event should be blocked by budget cap")
}

func TestRulesEngine_ConcurrentEvents(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID,
		testutil.WithBudgetBalance(1000.0),
		testutil.WithBudgetCaps(1000.0, 1000.0),
	)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID)

	// Create rule with per-user cap of 5
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID,
		testutil.WithPerUserCap(5),
	)

	ctx := context.Background()
	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Process 10 events concurrently
	concurrency := 10
	results := make(chan []db.Issuance, concurrency)
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			event := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
				testutil.WithIdempotencyKey(fmt.Sprintf("concurrent-%d", index)),
			)
			issuances, err := engine.ProcessEvent(ctx, event)
			if err != nil {
				errors <- err
				return
			}
			results <- issuances
		}(i)
	}

	// Collect results
	totalIssuances := 0
	for i := 0; i < concurrency; i++ {
		select {
		case err := <-errors:
			require.NoError(t, err)
		case issuances := <-results:
			totalIssuances += len(issuances)
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// Should have exactly 5 issuances (per-user cap)
	assert.Equal(t, 5, totalIssuances, "Expected exactly 5 issuances due to per-user cap")
}

func TestRulesEngine_CompleteFlow(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)
	budgetSvc := budget.NewService(pool, queries, logger)

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID,
		testutil.WithRewardCost(10.0),
	)

	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID)

	ctx := context.Background()
	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Step 1: Create event
	event := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID)

	// Step 2: Rules engine processes and creates issuance
	issuances, err := engine.ProcessEvent(ctx, event)
	require.NoError(t, err)
	require.Len(t, issuances, 1)

	issuance := issuances[0]
	assert.Equal(t, "reserved", issuance.Status)

	// Step 3: Verify budget was reserved
	ledgerEntries, err := queries.GetLedgerEntries(ctx, db.GetLedgerEntriesParams{
		TenantID: tenant.ID,
		BudgetID: testBudget.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)
	assert.Greater(t, len(ledgerEntries), 0, "Should have ledger entry for reservation")

	// Find the reservation entry
	var reservationEntry *db.LedgerEntry
	for _, entry := range ledgerEntries {
		if entry.Type == "reserve" {
			reservationEntry = &entry
			break
		}
	}
	require.NotNil(t, reservationEntry, "Should have reservation entry")

	// Step 4: Simulate redemption by charging the budget
	err = budgetSvc.ChargeReservation(ctx, testBudget.ID, issuance.ID)
	require.NoError(t, err)

	// Step 5: Verify budget was charged
	ledgerEntries, err = queries.GetLedgerEntries(ctx, db.GetLedgerEntriesParams{
		TenantID: tenant.ID,
		BudgetID: testBudget.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)

	var chargeEntry *db.LedgerEntry
	for _, entry := range ledgerEntries {
		if entry.Type == "charge" {
			chargeEntry = &entry
			break
		}
	}
	require.NotNil(t, chargeEntry, "Should have charge entry")
}
