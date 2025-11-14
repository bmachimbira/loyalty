package integration

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/rules"
	"github.com/bmachimbira/loyalty/api/internal/testutil"
)

// TestEventIngestion_EndToEnd tests the complete event ingestion flow:
// 1. Create event
// 2. Rules engine processes
// 3. Budget reserved
// 4. Reward issued
// 5. Verify database state
func TestEventIngestion_EndToEnd(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	ctx := context.Background()

	// Setup: Create tenant, customer, budget, campaign, reward, rule
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID,
		testutil.WithBudgetBalance(1000.0),
		testutil.WithBudgetCaps(1000.0, 1000.0),
	)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(t, queries, tenant.ID,
		testutil.WithRewardName("10 USD Discount"),
		testutil.WithRewardType("discount"),
		testutil.WithRewardCost(10.0),
		testutil.WithRewardFace(10.0),
	)

	// Create rule: Purchase amount >= 50
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID,
		testutil.WithRuleName("Spend $50, Get $10 Off"),
		testutil.WithEventType("purchase"),
		testutil.WithConditions(map[string]interface{}{
			">=": []interface{}{
				map[string]interface{}{"var": "amount"},
				50,
			},
		}),
	)

	// Update rule to link to campaign
	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Step 1: Create event (customer makes a $75 purchase)
	properties := map[string]interface{}{
		"amount":      75.0,
		"currency":    "USD",
		"product_id":  "PROD123",
		"location_id": "LOC001",
	}
	event := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithEventType("purchase"),
		testutil.WithProperties(properties),
		testutil.WithIdempotencyKey("purchase-event-1"),
	)

	require.NotEmpty(t, event.ID)
	assert.Equal(t, "purchase", event.EventType)

	// Step 2: Rules engine processes event
	issuances, err := engine.ProcessEvent(ctx, event)
	require.NoError(t, err, "Rules engine should process event successfully")
	require.Len(t, issuances, 1, "Should create 1 issuance")

	issuance := issuances[0]
	assert.Equal(t, tenant.ID, issuance.TenantID)
	assert.Equal(t, customer.ID, issuance.CustomerID)
	assert.Equal(t, campaign.ID, issuance.CampaignID)
	assert.Equal(t, reward.ID, issuance.RewardID)
	assert.Equal(t, event.ID, issuance.EventID)
	assert.Equal(t, "reserved", issuance.Status)

	// Step 3: Verify budget was reserved
	budgetAfter, err := queries.GetBudget(ctx, testBudget.ID)
	require.NoError(t, err)

	// Balance should be less than original due to reservation
	assert.True(t, budgetAfter.Balance.Int.Cmp(testBudget.Balance.Int) < 0,
		"Budget balance should decrease after reservation")

	// Step 4: Verify ledger entry was created
	ledgerEntries, err := queries.GetLedgerEntries(ctx, db.GetLedgerEntriesParams{
		TenantID: tenant.ID,
		BudgetID: testBudget.ID,
		Limit:    10,
		Offset:   0,
	})
	require.NoError(t, err)
	assert.Greater(t, len(ledgerEntries), 0, "Should have at least one ledger entry")

	// Find reservation entry
	var foundReservation bool
	for _, entry := range ledgerEntries {
		if entry.Type == "reserve" && entry.IssuanceID.Valid {
			foundReservation = true
			assert.Equal(t, issuance.ID, entry.IssuanceID)
			break
		}
	}
	assert.True(t, foundReservation, "Should have a reservation ledger entry")

	// Step 5: Verify issuance exists in database
	fetchedIssuance, err := queries.GetIssuance(ctx, issuance.ID)
	require.NoError(t, err)
	assert.Equal(t, "reserved", fetchedIssuance.Status)
	assert.Equal(t, event.ID, fetchedIssuance.EventID)
}

func TestEventIngestion_MultipleRulesTriggered(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := rules.NewEngine(pool, logger)

	ctx := context.Background()

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID,
		testutil.WithBudgetBalance(1000.0),
		testutil.WithBudgetCaps(1000.0, 1000.0),
	)
	campaign := testutil.CreateTestCampaign(t, queries, tenant.ID, testBudget.ID)

	// Create multiple rewards
	reward1 := testutil.CreateTestReward(t, queries, tenant.ID,
		testutil.WithRewardName("5 USD Discount"),
		testutil.WithRewardCost(5.0),
	)
	reward2 := testutil.CreateTestReward(t, queries, tenant.ID,
		testutil.WithRewardName("10 USD Discount"),
		testutil.WithRewardCost(10.0),
	)

	// Create multiple rules with different thresholds
	rule1 := testutil.CreateTestRule(t, queries, tenant.ID, reward1.ID,
		testutil.WithRuleName("Spend $20, Get $5 Off"),
		testutil.WithConditions(map[string]interface{}{
			">=": []interface{}{
				map[string]interface{}{"var": "amount"},
				20,
			},
		}),
	)
	rule2 := testutil.CreateTestRule(t, queries, tenant.ID, reward2.ID,
		testutil.WithRuleName("Spend $50, Get $10 Off"),
		testutil.WithConditions(map[string]interface{}{
			">=": []interface{}{
				map[string]interface{}{"var": "amount"},
				50,
			},
		}),
	)

	// Update rules to link to campaign
	for _, rule := range []db.Rule{rule1, rule2} {
		_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
			ID:         rule.ID,
			Name:       rule.Name,
			EventType:  rule.EventType,
			Conditions: rule.Conditions,
			Status:     rule.Status,
			CampaignID: campaign.ID,
		})
		require.NoError(t, err)
	}

	// Create event with amount = $60 (should trigger both rules)
	event := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithProperties(map[string]interface{}{
			"amount": 60.0,
		}),
	)

	// Process event
	issuances, err := engine.ProcessEvent(ctx, event)
	require.NoError(t, err)

	// Both rules should trigger
	assert.Len(t, issuances, 2, "Both rules should be triggered")

	// Verify both rewards were issued
	rewardIDs := make(map[string]bool)
	for _, issuance := range issuances {
		rewardIDStr := issuance.RewardID.Bytes[:]
		rewardIDs[string(rewardIDStr)] = true
	}
	assert.Len(t, rewardIDs, 2, "Should have 2 different rewards")
}

func TestEventIngestion_InactiveRule(t *testing.T) {
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

	// Create inactive rule
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID,
		testutil.WithRuleStatus("inactive"),
	)

	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     "inactive",
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Create event
	event := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID)

	// Process event
	issuances, err := engine.ProcessEvent(ctx, event)
	require.NoError(t, err)

	// Inactive rule should not trigger
	assert.Len(t, issuances, 0, "Inactive rule should not trigger")
}

func TestEventIngestion_WrongEventType(t *testing.T) {
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

	// Create rule for "purchase" events
	rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID,
		testutil.WithEventType("purchase"),
	)

	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  "purchase",
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(t, err)

	// Create "signup" event (different type)
	event := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
		testutil.WithEventType("signup"),
	)

	// Process event
	issuances, err := engine.ProcessEvent(ctx, event)
	require.NoError(t, err)

	// Rule should not trigger for wrong event type
	assert.Len(t, issuances, 0, "Rule should not trigger for wrong event type")
}
