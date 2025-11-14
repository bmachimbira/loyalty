package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Status represents the health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckDetail `json:"checks,omitempty"`
}

// CheckDetail represents details of a specific check
type CheckDetail struct {
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// Checker performs health checks
type Checker struct {
	pool *pgxpool.Pool
}

// NewChecker creates a new health checker
func NewChecker(pool *pgxpool.Pool) *Checker {
	return &Checker{
		pool: pool,
	}
}

// Check performs a basic health check
func (c *Checker) Check(ctx context.Context) CheckResult {
	result := CheckResult{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Checks:    make(map[string]CheckDetail),
	}

	// Check database connection
	dbCheck := c.checkDatabase(ctx)
	result.Checks["database"] = dbCheck

	// If database is unhealthy, overall status is unhealthy
	if dbCheck.Status == StatusUnhealthy {
		result.Status = StatusUnhealthy
	}

	return result
}

// Ready performs a readiness check (more comprehensive than health check)
func (c *Checker) Ready(ctx context.Context) CheckResult {
	result := CheckResult{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Checks:    make(map[string]CheckDetail),
	}

	// Run all checks in parallel
	var wg sync.WaitGroup
	checkFuncs := map[string]func(context.Context) CheckDetail{
		"database":           c.checkDatabase,
		"database_pool":      c.checkDatabasePool,
		"database_migration": c.checkDatabaseMigration,
	}

	results := make(map[string]CheckDetail)
	var mu sync.Mutex

	for name, checkFunc := range checkFuncs {
		wg.Add(1)
		go func(n string, fn func(context.Context) CheckDetail) {
			defer wg.Done()
			detail := fn(ctx)
			mu.Lock()
			results[n] = detail
			mu.Unlock()
		}(name, checkFunc)
	}

	wg.Wait()

	// Aggregate results
	hasUnhealthy := false
	hasDegraded := false

	for name, detail := range results {
		result.Checks[name] = detail
		if detail.Status == StatusUnhealthy {
			hasUnhealthy = true
		} else if detail.Status == StatusDegraded {
			hasDegraded = true
		}
	}

	// Determine overall status
	if hasUnhealthy {
		result.Status = StatusUnhealthy
	} else if hasDegraded {
		result.Status = StatusDegraded
	}

	return result
}

// checkDatabase checks basic database connectivity
func (c *Checker) checkDatabase(ctx context.Context) CheckDetail {
	start := time.Now()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := c.pool.Ping(pingCtx)
	latency := time.Since(start)

	if err != nil {
		return CheckDetail{
			Status:  StatusUnhealthy,
			Message: "Database ping failed",
			Error:   err.Error(),
			Latency: latency.String(),
		}
	}

	return CheckDetail{
		Status:  StatusHealthy,
		Message: "Database connection OK",
		Latency: latency.String(),
	}
}

// checkDatabasePool checks database connection pool stats
func (c *Checker) checkDatabasePool(ctx context.Context) CheckDetail {
	stats := c.pool.Stat()

	// Check if we have idle connections available
	if stats.IdleConns() == 0 && stats.AcquireCount() > 0 {
		return CheckDetail{
			Status:  StatusDegraded,
			Message: fmt.Sprintf("No idle connections available (total: %d, acquired: %d, idle: %d)",
				stats.TotalConns(), stats.AcquiredConns(), stats.IdleConns()),
		}
	}

	// Check if pool is exhausted
	maxConns := stats.MaxConns()
	totalConns := stats.TotalConns()
	if totalConns >= maxConns {
		return CheckDetail{
			Status:  StatusDegraded,
			Message: fmt.Sprintf("Connection pool at capacity (total: %d, max: %d)", totalConns, maxConns),
		}
	}

	return CheckDetail{
		Status:  StatusHealthy,
		Message: fmt.Sprintf("Pool healthy (total: %d, acquired: %d, idle: %d)",
			totalConns, stats.AcquiredConns(), stats.IdleConns()),
	}
}

// checkDatabaseMigration checks if database schema is up to date
func (c *Checker) checkDatabaseMigration(ctx context.Context) CheckDetail {
	// Query to check if migrations table exists
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'schema_migrations'
		)
	`

	err := c.pool.QueryRow(queryCtx, query).Scan(&exists)
	if err != nil {
		return CheckDetail{
			Status:  StatusUnhealthy,
			Message: "Failed to check migration status",
			Error:   err.Error(),
		}
	}

	if !exists {
		return CheckDetail{
			Status:  StatusDegraded,
			Message: "Migrations table not found - database may not be initialized",
		}
	}

	// Check if we have basic tables (tenants, customers, etc.)
	query = `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name IN ('tenants', 'customers', 'events', 'rules', 'rewards')
	`

	var tableCount int
	err = c.pool.QueryRow(queryCtx, query).Scan(&tableCount)
	if err != nil {
		return CheckDetail{
			Status:  StatusDegraded,
			Message: "Failed to verify schema tables",
			Error:   err.Error(),
		}
	}

	if tableCount < 5 {
		return CheckDetail{
			Status:  StatusDegraded,
			Message: fmt.Sprintf("Some required tables are missing (found: %d/5)", tableCount),
		}
	}

	return CheckDetail{
		Status:  StatusHealthy,
		Message: "Database schema verified",
	}
}

// CheckExternal checks external service health (placeholder for future implementation)
func (c *Checker) CheckExternal(ctx context.Context, serviceName, url string) CheckDetail {
	// This is a placeholder for checking external services
	// In production, you would:
	// 1. Make HTTP request to external service health endpoint
	// 2. Check circuit breaker state
	// 3. Verify credentials/API keys work

	return CheckDetail{
		Status:  StatusHealthy,
		Message: fmt.Sprintf("%s service check not implemented", serviceName),
	}
}
