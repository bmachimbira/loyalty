# Test Suite

This directory contains comprehensive tests for the Zimbabwe Loyalty Platform backend.

## Directory Structure

```
tests/
├── integration/      # Integration tests (cross-component)
│   ├── rules_engine_test.go
│   ├── event_flow_test.go
│   ├── idempotency_test.go
│   └── tenant_isolation_test.go
├── api/             # API endpoint tests
│   └── customers_test.go
└── performance/     # Performance benchmarks
    └── event_ingestion_test.go
```

## Running Tests

### All Tests
```bash
cd /home/user/loyalty
make test-all
```

### Specific Test Suites
```bash
# Integration tests
make test-integration

# API tests
make test-api

# Performance tests
make test-performance
```

### Individual Test Files
```bash
# Run specific test file
go test -v ./tests/integration/rules_engine_test.go

# Run specific test
go test -v -run TestRulesEngine_SimpleCondition ./tests/integration/

# Run with coverage
go test -v -coverprofile=coverage.out ./tests/integration/
```

## Test Categories

### Integration Tests (27 tests)

Test interactions between multiple components:

- **Rules Engine** (9 tests)
  - Simple and complex conditions
  - Per-user and global caps
  - Cooldown enforcement
  - Budget integration
  - Concurrent processing

- **Event Flow** (5 tests)
  - End-to-end ingestion
  - Multiple rules triggering
  - Inactive rules
  - Event type filtering

- **Idempotency** (5 tests)
  - Duplicate prevention
  - Key uniqueness
  - Cross-tenant keys

- **Tenant Isolation** (8 tests)
  - RLS policy verification
  - Cross-tenant access prevention
  - All entity types tested

### API Tests (10+ tests)

Test HTTP endpoints:

- **Customers API**
  - CRUD operations
  - Validation
  - Pagination
  - Cross-tenant access

### Performance Tests (4 tests)

Benchmark critical paths:

- Event ingestion throughput
- Latency measurement (P95 < 150ms)
- Sustained load (100 RPS)
- Concurrent processing

## Prerequisites

1. **PostgreSQL Database**
   ```bash
   docker-compose up db -d
   ```

2. **Test Database**
   ```bash
   psql -U postgres -c "CREATE DATABASE loyalty_test;"
   ```

3. **Migrations**
   ```bash
   migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable" up
   ```

4. **Environment Variable**
   ```bash
   export DATABASE_URL=postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable
   ```

## Test Utilities

See `/api/internal/testutil/` for helper functions:

- **Database Setup**: `SetupTestDB(t)`
- **Test Fixtures**: `CreateTestTenant()`, `CreateTestCustomer()`, etc.
- **HTTP Helpers**: `MakeGinRequest()`, `AssertJSONResponse()`, etc.

## Coverage

Generate coverage report:
```bash
make coverage
```

Check coverage meets 80% threshold:
```bash
make coverage-check
```

View HTML report:
```bash
open api/coverage.html
```

## CI/CD

Tests run automatically on:
- Every push to main/develop
- Every pull request

See `.github/workflows/test.yml` for configuration.

## Writing New Tests

1. Use test utilities from `testutil` package
2. Follow existing patterns
3. Use table-driven tests for multiple scenarios
4. Add cleanup with `t.Cleanup()`
5. Use descriptive test names

Example:
```go
func TestMyFeature(t *testing.T) {
    pool, queries := testutil.SetupTestDB(t)
    tenant := testutil.CreateTestTenant(t, queries)

    // Your test logic

    // Cleanup is automatic
}
```

## Documentation

- [TESTING.md](../../TESTING.md) - Comprehensive testing guide
- [TEST_SUMMARY.md](../../TEST_SUMMARY.md) - Implementation summary

## Support

For questions or issues:
1. Check existing tests for patterns
2. Review TESTING.md documentation
3. Run tests locally before pushing
