# Testing Guide

This document describes the comprehensive testing strategy for the Zimbabwe Loyalty Platform.

## Table of Contents

- [Overview](#overview)
- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Test Coverage](#test-coverage)
- [Writing Tests](#writing-tests)
- [CI/CD Integration](#cicd-integration)

## Overview

The platform has a comprehensive test suite covering:

1. **Unit Tests** - Test individual functions and packages in isolation
2. **Integration Tests** - Test interactions between components
3. **API Tests** - Test HTTP endpoints
4. **Performance Tests** - Benchmark critical paths
5. **Frontend Tests** - Test React components and UI

### Test Targets

- **Backend Coverage**: >80%
- **Frontend Coverage**: >70%
- **Event Ingestion P95 Latency**: <150ms
- **Rule Evaluation**: <25ms per event
- **Sustained Load**: 100 RPS

## Test Structure

```
api/
├── internal/                  # Unit tests alongside source code
│   ├── rules/
│   │   ├── engine.go
│   │   ├── engine_test.go    # Unit tests
│   │   ├── jsonlogic_test.go
│   │   └── benchmark_test.go # Benchmarks
│   ├── budget/
│   │   ├── service.go
│   │   └── service_test.go
│   └── testutil/             # Test utilities
│       ├── db.go             # Database test helpers
│       ├── fixtures.go       # Test data fixtures
│       └── http.go           # HTTP test helpers
├── tests/
│   ├── integration/          # Integration tests
│   │   ├── rules_engine_test.go
│   │   ├── event_flow_test.go
│   │   ├── idempotency_test.go
│   │   ├── tenant_isolation_test.go
│   │   └── ...
│   ├── api/                  # API endpoint tests
│   │   ├── customers_test.go
│   │   ├── events_test.go
│   │   └── ...
│   └── performance/          # Performance benchmarks
│       └── event_ingestion_test.go

web/
└── src/
    ├── components/
    │   └── __tests__/        # Component tests
    └── lib/
        └── __tests__/        # Utility tests
```

## Running Tests

### Prerequisites

1. PostgreSQL database running:
   ```bash
   docker-compose up db -d
   ```

2. Create test database:
   ```bash
   psql -U postgres -c "CREATE DATABASE loyalty_test;"
   ```

3. Run migrations:
   ```bash
   migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable" up
   ```

### Using Make Commands

```bash
# Run all tests
make test-all

# Run specific test suites
make test-unit          # Unit tests only
make test-integration   # Integration tests only
make test-api           # API tests only
make test-performance   # Performance benchmarks
make test-backend       # All backend tests
make test-frontend      # Frontend tests

# Generate coverage report
make coverage

# Check coverage meets threshold
make coverage-check

# Run benchmarks
make benchmark

# Clean coverage files
make clean-coverage
```

### Manual Test Execution

#### Backend Unit Tests

```bash
cd api
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable
go test ./internal/... -v -coverprofile=coverage.out
```

#### Integration Tests

```bash
cd api
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable
go test ./tests/integration/... -v
```

#### API Tests

```bash
cd api
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable
go test ./tests/api/... -v
```

#### Performance Tests

```bash
cd api
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable
go test ./tests/performance/... -bench=. -benchmem -benchtime=10s
```

#### Frontend Tests

```bash
cd web
npm test                    # Run once
npm test -- --watch        # Watch mode
npm test -- --coverage     # With coverage
```

### Running Individual Tests

```bash
# Run specific test
go test -v -run TestRulesEngine_SimpleCondition ./tests/integration/

# Run specific benchmark
go test -bench=BenchmarkEventIngestion ./tests/performance/

# Run with race detector
go test -race ./...

# Run with verbose output
go test -v ./...
```

## Test Coverage

### Generating Coverage Reports

```bash
# Generate coverage for all packages
make coverage

# View coverage in terminal
cd api
go tool cover -func=coverage.out

# View HTML coverage report
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # or xdg-open on Linux
```

### Coverage Requirements

The CI pipeline enforces the following coverage thresholds:

- **Backend**: Minimum 80% coverage
- **Frontend**: Minimum 70% coverage

To check if your code meets the threshold:

```bash
make coverage-check
```

### Current Coverage Status

Run this command to see current coverage:

```bash
cd api
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

## Writing Tests

### Test Utilities

The `testutil` package provides helpers for common test scenarios:

#### Database Helpers

```go
import "github.com/bmachimbira/loyalty/api/internal/testutil"

func TestMyFeature(t *testing.T) {
    // Setup test database (automatically cleans up)
    pool, queries := testutil.SetupTestDB(t)

    // Create test data
    tenant := testutil.CreateTestTenant(t, queries)
    customer := testutil.CreateTestCustomer(t, queries, tenant.ID,
        testutil.WithPhone("+263771234567"),
    )

    // Your test code here...
}
```

#### HTTP Helpers

```go
// Create test router
router := testutil.NewTestRouter(t)

// Make request
w := testutil.MakeGinRequest(t, router, "POST", "/endpoint", body)

// Assert response
testutil.AssertGinJSONResponse(t, w, http.StatusOK, &response)
```

#### Fixtures

Use option functions for flexible test data creation:

```go
// Create with defaults
budget := testutil.CreateTestBudget(t, queries, tenantID)

// Create with custom values
budget := testutil.CreateTestBudget(t, queries, tenantID,
    testutil.WithBudgetBalance(5000.0),
    testutil.WithBudgetCaps(10000.0, 15000.0),
    testutil.WithBudgetCurrency("USD"),
)
```

### Test Patterns

#### Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expected    bool
        expectError bool
    }{
        {"valid phone", "+263771234567", true, false},
        {"invalid phone", "invalid", false, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ValidatePhone(tt.input)
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

#### Integration Test Pattern

```go
func TestFeature_EndToEnd(t *testing.T) {
    // Setup
    pool, queries := testutil.SetupTestDB(t)
    tenant := testutil.CreateTestTenant(t, queries)

    // Execute
    // ... your test logic

    // Verify
    // ... assertions

    // Cleanup is automatic with t.Cleanup()
}
```

### Benchmarking

```go
func BenchmarkMyFunction(b *testing.B) {
    // Setup
    pool, queries := testutil.SetupTestDB(&testing.T{})

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        // Code to benchmark
        MyFunction()
    }
}
```

## CI/CD Integration

### GitHub Actions Workflows

The project has two main workflows:

#### 1. Test Workflow (.github/workflows/test.yml)

Runs on every push and pull request:

- Backend unit tests
- Integration tests
- API tests
- Performance benchmarks
- Frontend tests
- Coverage reporting
- Coverage threshold check (80%)

#### 2. Build Workflow (.github/workflows/build.yml)

Runs on every push and pull request:

- Backend build
- Frontend build
- Docker image builds
- Security scanning (Trivy)
- Linting and formatting checks

### Local Pre-commit Checks

Before committing, run:

```bash
# Format code
make fmt

# Run linters
make lint

# Run all tests
make test-all

# Check coverage
make coverage-check
```

## Performance Testing

### Event Ingestion Performance

Target: P95 latency < 150ms

```bash
cd api
go test ./tests/performance/... -bench=BenchmarkEventIngestion -benchtime=10s
```

### Load Testing

Test sustained load at 100 RPS:

```bash
go test ./tests/performance/... -run TestEventIngestion_SustainedLoad -timeout 30s
```

### Concurrent Testing

Test with concurrent goroutines:

```bash
go test ./tests/performance/... -run TestEventIngestion_ConcurrentLoad
```

## Troubleshooting

### Database Connection Issues

If tests fail with database connection errors:

1. Check PostgreSQL is running:
   ```bash
   docker-compose ps
   ```

2. Verify database exists:
   ```bash
   psql -U postgres -l | grep loyalty_test
   ```

3. Check DATABASE_URL:
   ```bash
   echo $DATABASE_URL
   ```

### Flaky Tests

If tests are flaky:

1. Run with race detector:
   ```bash
   go test -race ./...
   ```

2. Run multiple times:
   ```bash
   go test -count=10 ./tests/integration/...
   ```

### Coverage Issues

If coverage is below threshold:

1. Generate detailed coverage report:
   ```bash
   make coverage
   ```

2. Open HTML report to identify gaps:
   ```bash
   open api/coverage.html
   ```

3. Add tests for uncovered code

## Best Practices

1. **Isolation**: Each test should be independent and not rely on others
2. **Cleanup**: Use `t.Cleanup()` for automatic cleanup
3. **Fixtures**: Use testutil helpers for consistent test data
4. **Naming**: Use descriptive test names: `TestFeature_Scenario_ExpectedResult`
5. **Table-Driven**: Use table-driven tests for multiple scenarios
6. **Fast Tests**: Keep unit tests fast (<100ms)
7. **Parallel**: Use `t.Parallel()` for independent tests
8. **Assertions**: Use testify/assert for better error messages
9. **Coverage**: Aim for high coverage but don't sacrifice quality
10. **Documentation**: Document complex test scenarios

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Testing in Go](https://golang.org/doc/tutorial/add-a-test)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
