package performance

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/rules"
	"github.com/bmachimbira/loyalty/api/internal/testutil"
)

// BenchmarkEventIngestion benchmarks event ingestion performance
// Target: p95 < 150ms
func BenchmarkEventIngestion(b *testing.B) {
	pool, queries := testutil.SetupTestDB(&testing.T{})
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := rules.NewEngine(pool, logger)

	ctx := context.Background()

	// Setup
	tenant := testutil.CreateTestTenant(&testing.T{}, queries)
	customer := testutil.CreateTestCustomer(&testing.T{}, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(&testing.T{}, queries, tenant.ID,
		testutil.WithBudgetBalance(100000.0),
		testutil.WithBudgetCaps(100000.0, 100000.0),
	)
	campaign := testutil.CreateTestCampaign(&testing.T{}, queries, tenant.ID, testBudget.ID)
	reward := testutil.CreateTestReward(&testing.T{}, queries, tenant.ID)
	rule := testutil.CreateTestRule(&testing.T{}, queries, tenant.ID, reward.ID)

	_, err := queries.UpdateRule(ctx, db.UpdateRuleParams{
		ID:         rule.ID,
		Name:       rule.Name,
		EventType:  rule.EventType,
		Conditions: rule.Conditions,
		Status:     rule.Status,
		CampaignID: campaign.ID,
	})
	require.NoError(&testing.T{}, err)

	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		event := testutil.CreateTestEvent(&testing.T{}, queries, tenant.ID, customer.ID,
			testutil.WithIdempotencyKey(fmt.Sprintf("bench-%d", i)),
		)

		_, err := engine.ProcessEvent(ctx, event)
		if err != nil {
			b.Fatalf("Event processing failed: %v", err)
		}
	}
}

// TestEventIngestion_Latency measures event ingestion latency
func TestEventIngestion_Latency(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := rules.NewEngine(pool, logger)

	ctx := context.Background()

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID,
		testutil.WithBudgetBalance(100000.0),
		testutil.WithBudgetCaps(100000.0, 100000.0),
	)
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

	// Measure latencies
	iterations := 100
	latencies := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		event := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
			testutil.WithIdempotencyKey(fmt.Sprintf("latency-%d", i)),
		)

		start := time.Now()
		_, err := engine.ProcessEvent(ctx, event)
		latency := time.Since(start)

		require.NoError(t, err)
		latencies[i] = latency
	}

	// Calculate statistics
	var total time.Duration
	min := latencies[0]
	max := latencies[0]

	for _, latency := range latencies {
		total += latency
		if latency < min {
			min = latency
		}
		if latency > max {
			max = latency
		}
	}

	avg := total / time.Duration(iterations)

	// Calculate p95
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	p95Index := int(float64(iterations) * 0.95)
	p95 := sorted[p95Index]

	t.Logf("Event Ingestion Latency Statistics:")
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)
	t.Logf("  Avg: %v", avg)
	t.Logf("  P95: %v", p95)

	// Assert p95 < 150ms
	require.Less(t, p95, 150*time.Millisecond, "P95 latency should be less than 150ms")
}

// TestEventIngestion_SustainedLoad tests sustained load at 100 RPS
func TestEventIngestion_SustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sustained load test in short mode")
	}

	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := rules.NewEngine(pool, logger)

	ctx := context.Background()

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID,
		testutil.WithBudgetBalance(100000.0),
		testutil.WithBudgetCaps(100000.0, 100000.0),
	)
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

	// Test parameters
	targetRPS := 100
	duration := 10 * time.Second
	interval := time.Second / time.Duration(targetRPS)

	start := time.Now()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	successCount := 0
	errorCount := 0
	counter := 0

	for time.Since(start) < duration {
		<-ticker.C

		event := testutil.CreateTestEvent(t, queries, tenant.ID, customer.ID,
			testutil.WithIdempotencyKey(fmt.Sprintf("sustained-%d", counter)),
		)

		_, err := engine.ProcessEvent(ctx, event)
		if err != nil {
			errorCount++
			t.Logf("Error processing event: %v", err)
		} else {
			successCount++
		}

		counter++
	}

	elapsed := time.Since(start)
	actualRPS := float64(counter) / elapsed.Seconds()

	t.Logf("Sustained Load Test Results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Total Events: %d", counter)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Actual RPS: %.2f", actualRPS)
	t.Logf("  Success Rate: %.2f%%", float64(successCount)/float64(counter)*100)

	// Assert reasonable success rate
	successRate := float64(successCount) / float64(counter)
	require.Greater(t, successRate, 0.95, "Success rate should be > 95%")
}

// TestEventIngestion_ConcurrentLoad tests concurrent event processing
func TestEventIngestion_ConcurrentLoad(t *testing.T) {
	pool, queries := testutil.SetupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := rules.NewEngine(pool, logger)

	ctx := context.Background()

	// Setup
	tenant := testutil.CreateTestTenant(t, queries)
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
	testBudget := testutil.CreateTestBudget(t, queries, tenant.ID,
		testutil.WithBudgetBalance(100000.0),
		testutil.WithBudgetCaps(100000.0, 100000.0),
	)
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

	// Test with 10 concurrent goroutines
	concurrency := 10
	eventsPerGoroutine := 10

	type result struct {
		success bool
		latency time.Duration
		err     error
	}

	results := make(chan result, concurrency*eventsPerGoroutine)

	start := time.Now()

	for g := 0; g < concurrency; g++ {
		go func(goroutineID int) {
			for e := 0; e < eventsPerGoroutine; e++ {
				event := testutil.CreateTestEvent(&testing.T{}, queries, tenant.ID, customer.ID,
					testutil.WithIdempotencyKey(fmt.Sprintf("concurrent-%d-%d", goroutineID, e)),
				)

				eventStart := time.Now()
				_, err := engine.ProcessEvent(ctx, event)
				latency := time.Since(eventStart)

				results <- result{
					success: err == nil,
					latency: latency,
					err:     err,
				}
			}
		}(g)
	}

	// Collect results
	successCount := 0
	errorCount := 0
	totalLatency := time.Duration(0)

	for i := 0; i < concurrency*eventsPerGoroutine; i++ {
		res := <-results
		if res.success {
			successCount++
			totalLatency += res.latency
		} else {
			errorCount++
			t.Logf("Concurrent error: %v", res.err)
		}
	}

	elapsed := time.Since(start)
	avgLatency := totalLatency / time.Duration(successCount)

	t.Logf("Concurrent Load Test Results:")
	t.Logf("  Concurrency: %d", concurrency)
	t.Logf("  Total Events: %d", concurrency*eventsPerGoroutine)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Total Duration: %v", elapsed)
	t.Logf("  Average Latency: %v", avgLatency)

	// Assert
	require.Greater(t, successCount, concurrency*eventsPerGoroutine*9/10, "At least 90% should succeed")
}
