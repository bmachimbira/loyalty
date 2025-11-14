package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/bmachimbira/loyalty/api/internal/db"
)

// SetupTestDB creates a test database connection and runs migrations
func SetupTestDB(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	// Run migrations
	runMigrations(t, pool)

	// Register cleanup
	t.Cleanup(func() {
		cleanupDB(t, pool)
		pool.Close()
	})

	queries := db.New(pool)
	return pool, queries
}

// SetupTestDBWithTx creates a test database connection and starts a transaction
// The transaction is rolled back on cleanup, providing better isolation
func SetupTestDBWithTx(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	// Run migrations
	runMigrations(t, pool)

	// Start a transaction for this test
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	// Register cleanup - rollback transaction
	t.Cleanup(func() {
		tx.Rollback(ctx)
		pool.Close()
	})

	queries := db.New(tx)
	return pool, queries
}

// runMigrations executes all migration files in order
func runMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	// Get migrations directory
	migrationsDir := filepath.Join("..", "..", "..", "migrations")

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Try alternative path (when running from project root)
		migrationsDir = filepath.Join("migrations")
	}

	migrations := []string{
		"001_initial_schema.sql",
		"002_seed_data.sql",
		"003_indexes_optimization.sql",
		"004_voucher_pool.sql",
		"005_functions.sql",
		"006_webhook_deliveries.sql",
	}

	for _, migration := range migrations {
		migrationPath := filepath.Join(migrationsDir, migration)
		sql, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Logf("Warning: Could not read migration %s: %v", migration, err)
			continue
		}

		_, err = pool.Exec(ctx, string(sql))
		if err != nil {
			// Ignore errors if tables already exist
			t.Logf("Migration %s may have already been applied: %v", migration, err)
		}
	}
}

// cleanupDB truncates all tables to ensure clean state
func cleanupDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	// List of tables to truncate (in reverse dependency order)
	tables := []string{
		"audit_logs",
		"webhook_deliveries",
		"ledger_entries",
		"issuances",
		"events",
		"voucher_codes",
		"rules",
		"campaigns",
		"budgets",
		"consents",
		"customers",
		"ussd_sessions",
		"whatsapp_sessions",
		"staff_users",
	}

	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: Could not truncate table %s: %v", table, err)
		}
	}
}

// TruncateTables truncates specific tables
func TruncateTables(t *testing.T, pool *pgxpool.Pool, tables ...string) {
	t.Helper()

	ctx := context.Background()
	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(t, err, "Failed to truncate table %s", table)
	}
}

// SetTenantContext sets the app.tenant_id session variable for RLS
func SetTenantContext(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()

	ctx := context.Background()
	_, err := pool.Exec(ctx, "SET app.tenant_id = $1", tenantID)
	require.NoError(t, err, "Failed to set tenant context")
}

// ClearTenantContext clears the app.tenant_id session variable
func ClearTenantContext(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	_, err := pool.Exec(ctx, "RESET app.tenant_id")
	require.NoError(t, err, "Failed to clear tenant context")
}

// WithTenantContext executes a function with tenant context set
func WithTenantContext(t *testing.T, pool *pgxpool.Pool, tenantID string, fn func()) {
	t.Helper()

	SetTenantContext(t, pool, tenantID)
	defer ClearTenantContext(t, pool)
	fn()
}
