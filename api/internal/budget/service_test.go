package budget

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bmachimbira/loyalty/api/internal/db"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Note: These tests require a running PostgreSQL database
	// Set DATABASE_URL environment variable to run tests
	os.Exit(m.Run())
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*pgxpool.Pool, *db.Queries, func()) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration tests")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)

	queries := db.New(pool)

	cleanup := func() {
		pool.Close()
	}

	return pool, queries, cleanup
}

// createTestTenant creates a test tenant
func createTestTenant(t *testing.T, queries *db.Queries) pgtype.UUID {
	tenant := pgtype.UUID{}
	err := tenant.Scan("00000000-0000-0000-0000-000000000001")
	require.NoError(t, err)
	return tenant
}

// createTestBudget creates a test budget
func createTestBudget(t *testing.T, queries *db.Queries, tenantID pgtype.UUID, softCap, hardCap, balance string) *db.Budget {
	ctx := context.Background()

	// Note: In real tests, you would set tenant context via middleware
	// For unit tests, we're skipping this as the database functions handle it

	softCapNumeric := pgtype.Numeric{}
	err = softCapNumeric.Scan(softCap)
	require.NoError(t, err)

	hardCapNumeric := pgtype.Numeric{}
	err = hardCapNumeric.Scan(hardCap)
	require.NoError(t, err)

	balanceNumeric := pgtype.Numeric{}
	err = balanceNumeric.Scan(balance)
	require.NoError(t, err)

	budget, err := queries.CreateBudget(ctx, db.CreateBudgetParams{
		TenantID: tenantID,
		Name:     "Test Budget",
		Currency: CurrencyUSD,
		SoftCap:  softCapNumeric,
		HardCap:  hardCapNumeric,
		Balance:  balanceNumeric,
		Period:   "rolling",
	})
	require.NoError(t, err)

	return &budget
}

func TestReserveBudget_Success(t *testing.T) {
	pool, queries, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(pool, queries, logger)

	tenantID := createTestTenant(t, queries)
	budget := createTestBudget(t, queries, tenantID, "8000.00", "10000.00", "0.00")

	refID := pgtype.UUID{}
	err := refID.Scan("00000000-0000-0000-0000-000000000002")
	require.NoError(t, err)

	// Test successful reservation
	result, err := service.ReserveBudget(context.Background(), ReserveBudgetParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "1000.00",
		Currency: CurrencyUSD,
		RefID:    refID,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "1000.00", result.Amount)
	assert.Equal(t, 1000.0, result.NewBalance)
	assert.False(t, result.SoftCapExceeded)
	assert.Equal(t, 10.0, result.Utilization)
}

func TestReserveBudget_InsufficientFunds(t *testing.T) {
	pool, queries, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(pool, queries, logger)

	tenantID := createTestTenant(t, queries)
	budget := createTestBudget(t, queries, tenantID, "8000.00", "10000.00", "9500.00")

	refID := pgtype.UUID{}
	err := refID.Scan("00000000-0000-0000-0000-000000000003")
	require.NoError(t, err)

	// Test reservation that exceeds hard cap
	result, err := service.ReserveBudget(context.Background(), ReserveBudgetParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "1000.00", // Would exceed hard cap of 10000
		Currency: CurrencyUSD,
		RefID:    refID,
	})

	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientFunds, err)
	assert.Nil(t, result)
}

func TestReserveBudget_SoftCapExceeded(t *testing.T) {
	pool, queries, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(pool, queries, logger)

	tenantID := createTestTenant(t, queries)
	budget := createTestBudget(t, queries, tenantID, "5000.00", "10000.00", "0.00")

	refID := pgtype.UUID{}
	err := refID.Scan("00000000-0000-0000-0000-000000000004")
	require.NoError(t, err)

	// Test reservation that exceeds soft cap but not hard cap
	result, err := service.ReserveBudget(context.Background(), ReserveBudgetParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "6000.00", // Exceeds soft cap of 5000 but within hard cap of 10000
		Currency: CurrencyUSD,
		RefID:    refID,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.SoftCapExceeded)
	assert.Equal(t, 6000.0, result.NewBalance)
}

func TestReserveBudget_CurrencyMismatch(t *testing.T) {
	pool, queries, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(pool, queries, logger)

	tenantID := createTestTenant(t, queries)
	budget := createTestBudget(t, queries, tenantID, "8000.00", "10000.00", "0.00")

	refID := pgtype.UUID{}
	err := refID.Scan("00000000-0000-0000-0000-000000000005")
	require.NoError(t, err)

	// Test with wrong currency
	result, err := service.ReserveBudget(context.Background(), ReserveBudgetParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "1000.00",
		Currency: CurrencyZWG, // Budget is USD
		RefID:    refID,
	})

	assert.Error(t, err)
	assert.Equal(t, ErrCurrencyMismatch, err)
	assert.Nil(t, result)
}

func TestChargeReservation_Success(t *testing.T) {
	pool, queries, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(pool, queries, logger)

	tenantID := createTestTenant(t, queries)
	budget := createTestBudget(t, queries, tenantID, "8000.00", "10000.00", "0.00")

	refID := pgtype.UUID{}
	err := refID.Scan("00000000-0000-0000-0000-000000000006")
	require.NoError(t, err)

	// First reserve
	_, err = service.ReserveBudget(context.Background(), ReserveBudgetParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "1000.00",
		Currency: CurrencyUSD,
		RefID:    refID,
	})
	require.NoError(t, err)

	// Then charge
	err = service.ChargeReservation(context.Background(), ChargeReservationParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "1000.00",
		Currency: CurrencyUSD,
		RefID:    refID,
	})

	assert.NoError(t, err)
}

func TestChargeReservation_AlreadyCharged(t *testing.T) {
	pool, queries, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(pool, queries, logger)

	tenantID := createTestTenant(t, queries)
	budget := createTestBudget(t, queries, tenantID, "8000.00", "10000.00", "0.00")

	refID := pgtype.UUID{}
	err := refID.Scan("00000000-0000-0000-0000-000000000007")
	require.NoError(t, err)

	// Reserve and charge once
	_, err = service.ReserveBudget(context.Background(), ReserveBudgetParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "1000.00",
		Currency: CurrencyUSD,
		RefID:    refID,
	})
	require.NoError(t, err)

	err = service.ChargeReservation(context.Background(), ChargeReservationParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "1000.00",
		Currency: CurrencyUSD,
		RefID:    refID,
	})
	require.NoError(t, err)

	// Try to charge again
	err = service.ChargeReservation(context.Background(), ChargeReservationParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "1000.00",
		Currency: CurrencyUSD,
		RefID:    refID,
	})

	assert.Error(t, err)
	assert.Equal(t, ErrAlreadyCharged, err)
}

func TestReleaseReservation_Success(t *testing.T) {
	pool, queries, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(pool, queries, logger)

	tenantID := createTestTenant(t, queries)
	budget := createTestBudget(t, queries, tenantID, "8000.00", "10000.00", "0.00")

	refID := pgtype.UUID{}
	err := refID.Scan("00000000-0000-0000-0000-000000000008")
	require.NoError(t, err)

	// First reserve
	result, err := service.ReserveBudget(context.Background(), ReserveBudgetParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "1000.00",
		Currency: CurrencyUSD,
		RefID:    refID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1000.0, result.NewBalance)

	// Then release
	err = service.ReleaseReservation(context.Background(), ReleaseReservationParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "1000.00",
		Currency: CurrencyUSD,
		RefID:    refID,
	})

	assert.NoError(t, err)

	// Verify balance returned to 0
	updatedBudget, err := queries.GetBudgetByID(context.Background(), db.GetBudgetByIDParams{
		ID:       budget.ID,
		TenantID: tenantID,
	})
	require.NoError(t, err)

	var balance float64
	updatedBudget.Balance.Float(&balance)
	assert.Equal(t, 0.0, balance)
}

func TestTopupBudget_Success(t *testing.T) {
	pool, queries, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(pool, queries, logger)

	tenantID := createTestTenant(t, queries)
	budget := createTestBudget(t, queries, tenantID, "8000.00", "10000.00", "0.00")

	// Topup budget
	result, err := service.TopupBudget(context.Background(), TopupBudgetParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "5000.00",
		Currency: CurrencyUSD,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "5000.00", result.Amount)
	assert.Equal(t, 5000.0, result.NewBalance)
}

func TestReconcileBudget_NoDiscrepancy(t *testing.T) {
	pool, queries, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(pool, queries, logger)

	tenantID := createTestTenant(t, queries)
	budget := createTestBudget(t, queries, tenantID, "8000.00", "10000.00", "0.00")

	// Perform some operations
	refID := pgtype.UUID{}
	err := refID.Scan("00000000-0000-0000-0000-000000000009")
	require.NoError(t, err)

	_, err = service.ReserveBudget(context.Background(), ReserveBudgetParams{
		TenantID: tenantID,
		BudgetID: budget.ID,
		Amount:   "1000.00",
		Currency: CurrencyUSD,
		RefID:    refID,
	})
	require.NoError(t, err)

	// Reconcile
	result, err := service.ReconcileBudget(context.Background(), tenantID, budget.ID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasDiscrepancy)
	assert.Equal(t, 0.0, result.Discrepancy)
}

func TestConcurrentReservations(t *testing.T) {
	pool, queries, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(pool, queries, logger)

	tenantID := createTestTenant(t, queries)
	budget := createTestBudget(t, queries, tenantID, "8000.00", "10000.00", "0.00")

	// Try to reserve concurrently
	const numGoroutines = 10
	const amountPerReservation = "500.00"

	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			refID := pgtype.UUID{}
			err := refID.Scan(fmt.Sprintf("00000000-0000-0000-0000-%012d", idx))
			if err != nil {
				results <- err
				return
			}

			_, err = service.ReserveBudget(context.Background(), ReserveBudgetParams{
				TenantID: tenantID,
				BudgetID: budget.ID,
				Amount:   amountPerReservation,
				Currency: CurrencyUSD,
				RefID:    refID,
			})
			results <- err
		}(i)
	}

	// Collect results
	successCount := 0
	failureCount := 0

	for i := 0; i < numGoroutines; i++ {
		err := <-results
		if err == nil {
			successCount++
		} else if err == ErrInsufficientFunds {
			failureCount++
		} else {
			t.Errorf("unexpected error: %v", err)
		}
	}

	// At least some should succeed, some might fail due to hard cap
	// Total successful = balance / 500, should not exceed hard cap
	assert.True(t, successCount > 0, "at least some reservations should succeed")
	assert.True(t, successCount <= 20, "should not exceed hard cap (10000/500=20)")

	t.Logf("Concurrent test: %d successes, %d failures", successCount, failureCount)
}

func TestCurrencyValidation(t *testing.T) {
	assert.True(t, IsValidCurrency(CurrencyUSD))
	assert.True(t, IsValidCurrency(CurrencyZWG))
	assert.False(t, IsValidCurrency("EUR"))
	assert.False(t, IsValidCurrency(""))
}

func TestAlertThresholds(t *testing.T) {
	thresholds := DefaultAlertThresholds()
	assert.Equal(t, 80.0, thresholds.SoftCapPercent)
	assert.Equal(t, 95.0, thresholds.HardCapPercent)
}

func TestDateRangeCreation(t *testing.T) {
	today := NewDateRange("today")
	assert.True(t, today.From.Before(today.To))

	week := NewDateRange("week")
	assert.True(t, week.To.Sub(week.From).Hours() >= 168) // 7 days

	month := NewDateRange("month")
	assert.Equal(t, 1, month.From.Day())

	year := NewDateRange("year")
	assert.Equal(t, time.January, year.From.Month())
	assert.Equal(t, 1, year.From.Day())
}
