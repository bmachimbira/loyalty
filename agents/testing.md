# Testing Agent

## Mission
Create comprehensive test coverage for all components of the loyalty platform.

## Prerequisites
- Go 1.21+
- Node.js 18+ (for frontend tests)
- Understanding of testing patterns

## Tasks

### 1. Go Testing Setup

**File**: `api/internal/testutil/db.go`

```go
package testutil

import (
    "context"
    "testing"
    "github.com/jackc/pgx/v5/pgxpool"
)

func SetupTestDB(t *testing.T) *pgxpool.Pool {
    ctx := context.Background()

    // Connect to test database
    pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable")
    if err != nil {
        t.Fatalf("Failed to connect to test DB: %v", err)
    }

    // Run migrations
    runMigrations(t, pool)

    // Cleanup after test
    t.Cleanup(func() {
        cleanupDB(pool)
        pool.Close()
    })

    return pool
}

func runMigrations(t *testing.T, pool *pgxpool.Pool) {
    // Execute migration files
}

func cleanupDB(pool *pgxpool.Pool) {
    // Truncate all tables
}
```

**File**: `api/internal/testutil/fixtures.go`

```go
package testutil

func CreateTestTenant(t *testing.T, queries *db.Queries) *db.Tenant {
    tenant, err := queries.CreateTenant(ctx, db.CreateTenantParams{
        Name:       "Test Tenant",
        DefaultCcy: "USD",
    })
    if err != nil {
        t.Fatalf("Failed to create test tenant: %v", err)
    }
    return tenant
}

func CreateTestCustomer(t *testing.T, queries *db.Queries, tenantID string) *db.Customer {
    // Create test customer
}

func CreateTestReward(t *testing.T, queries *db.Queries, tenantID string) *db.Reward {
    // Create test reward
}

func CreateTestRule(t *testing.T, queries *db.Queries, tenantID, rewardID string) *db.Rule {
    // Create test rule
}
```

### 2. Unit Tests

#### Rules Engine Tests
**File**: `api/internal/rules/engine_test.go`

```go
package rules_test

func TestEngine_SimpleCondition(t *testing.T) {
    pool := testutil.SetupTestDB(t)
    queries := db.New(pool)
    engine := rules.NewEngine(pool, queries)

    // Setup: Create tenant, customer, reward, rule
    tenant := testutil.CreateTestTenant(t, queries)
    customer := testutil.CreateTestCustomer(t, queries, tenant.ID)
    reward := testutil.CreateTestReward(t, queries, tenant.ID)

    // Rule: amount >= 20
    rule := testutil.CreateTestRule(t, queries, tenant.ID, reward.ID)

    // Create event with amount = 25
    event := &db.Event{
        TenantID:   tenant.ID,
        CustomerID: customer.ID,
        EventType:  "purchase",
        Properties: json.RawMessage(`{"amount": 25, "currency": "USD"}`),
    }

    // Process event
    issuances, err := engine.ProcessEvent(ctx, event)

    // Assert
    assert.NoError(t, err)
    assert.Len(t, issuances, 1)
    assert.Equal(t, reward.ID, issuances[0].RewardID)
}

func TestEngine_PerUserCap(t *testing.T) {
    // Test per-user cap enforcement
}

func TestEngine_GlobalCap(t *testing.T) {
    // Test global cap enforcement
}

func TestEngine_Cooldown(t *testing.T) {
    // Test cooldown enforcement
}

func TestEngine_ConcurrentEvents(t *testing.T) {
    // Test concurrent event processing (race conditions)
}
```

#### Budget Tests
**File**: `api/internal/budget/service_test.go`

```go
package budget_test

func TestService_ReserveBudget(t *testing.T) {
    // Test successful reservation
}

func TestService_HardCapExceeded(t *testing.T) {
    // Test hard cap rejection
}

func TestService_ChargeReservation(t *testing.T) {
    // Test charge converts reservation
}

func TestService_ReleaseReservation(t *testing.T) {
    // Test release returns funds
}

func TestService_Reconciliation(t *testing.T) {
    // Test balance reconciliation
}
```

#### Reward Tests
**File**: `api/internal/reward/service_test.go`

```go
package reward_test

func TestService_ProcessDiscount(t *testing.T) {
    // Test discount code generation
}

func TestService_ProcessVoucherCode(t *testing.T) {
    // Test voucher code reservation
}

func TestService_StateTransitions(t *testing.T) {
    // Test valid/invalid state transitions
}

func TestService_Redemption(t *testing.T) {
    // Test redemption flow
}

func TestService_Expiry(t *testing.T) {
    // Test expiry processing
}
```

### 3. Integration Tests

**File**: `api/test/integration/events_test.go`

```go
package integration_test

func TestEventIngestion_EndToEnd(t *testing.T) {
    // Setup: Start API server, database
    // Create tenant, customer, reward, rule

    // POST /v1/tenants/:tid/events
    // Assert: Event created, rules evaluated, issuance created

    // GET /v1/tenants/:tid/issuances
    // Assert: Issuance exists

    // POST /v1/tenants/:tid/issuances/:id/redeem
    // Assert: Issuance redeemed, budget charged
}

func TestIdempotency(t *testing.T) {
    // POST same event twice with same idempotency key
    // Assert: Only one event/issuance created
}

func TestTenantIsolation(t *testing.T) {
    // Create two tenants
    // Attempt to access tenant2 data with tenant1 credentials
    // Assert: Forbidden or not found
}
```

### 4. RLS Policy Tests

**File**: `api/test/integration/rls_test.go`

```go
package integration_test

func TestRLS_CustomerIsolation(t *testing.T) {
    pool := testutil.SetupTestDB(t)
    ctx := context.Background()

    // Create two tenants
    tenant1ID := createTenant(t, pool, "Tenant 1")
    tenant2ID := createTenant(t, pool, "Tenant 2")

    // Set context to tenant1
    _, err := pool.Exec(ctx, "SET app.tenant_id = $1", tenant1ID)
    require.NoError(t, err)

    // Create customer for tenant1
    createCustomer(t, pool, tenant1ID)

    // Query customers
    rows, err := pool.Query(ctx, "SELECT * FROM customers")
    require.NoError(t, err)

    count := 0
    for rows.Next() {
        count++
    }
    assert.Equal(t, 1, count, "Should only see tenant1's customers")

    // Switch to tenant2
    _, err = pool.Exec(ctx, "SET app.tenant_id = $1", tenant2ID)
    require.NoError(t, err)

    // Query customers
    rows, err = pool.Query(ctx, "SELECT * FROM customers")
    require.NoError(t, err)

    count = 0
    for rows.Next() {
        count++
    }
    assert.Equal(t, 0, count, "Should see no customers for tenant2")
}
```

### 5. Performance Tests

**File**: `api/test/performance/load_test.go`

```go
package performance_test

func BenchmarkEventIngestion(b *testing.B) {
    // Setup
    pool := setupBenchDB(b)
    client := setupAPIClient()

    b.ResetTimer()

    // Run b.N times
    for i := 0; i < b.N; i++ {
        // POST event
        _, err := client.PostEvent(event)
        if err != nil {
            b.Fatalf("Failed: %v", err)
        }
    }

    // Assert: p95 < 150ms
}

func BenchmarkRuleEvaluation(b *testing.B) {
    // Benchmark rule evaluation
    // Target: < 25ms per event
}
```

**File**: `api/test/performance/concurrent_test.go`

```go
package performance_test

func TestConcurrentEventIngestion(t *testing.T) {
    // Spawn 100 goroutines
    // Each posts events concurrently
    // Assert: No errors, all events processed correctly
}
```

### 6. API Tests

**File**: `api/test/api/customers_test.go`

```go
package api_test

func TestCustomersAPI_Create(t *testing.T) {
    // POST /v1/tenants/:tid/customers
    // Assert: 201 Created, customer returned
}

func TestCustomersAPI_Get(t *testing.T) {
    // GET /v1/tenants/:tid/customers/:id
    // Assert: 200 OK, customer data correct
}

func TestCustomersAPI_Unauthorized(t *testing.T) {
    // Request without JWT token
    // Assert: 401 Unauthorized
}

func TestCustomersAPI_CrossTenantAccess(t *testing.T) {
    // Tenant1 tries to access Tenant2's customer
    // Assert: 404 Not Found (due to RLS)
}
```

### 7. Frontend Tests

**File**: `web/src/components/customers/CreateCustomerForm.test.tsx`

```typescript
import { render, screen, fireEvent } from '@testing-library/react';
import { CreateCustomerForm } from './CreateCustomerForm';

test('validates phone number format', () => {
  render(<CreateCustomerForm onSuccess={jest.fn()} />);

  const phoneInput = screen.getByLabelText('Phone Number');
  fireEvent.change(phoneInput, { target: { value: 'invalid' } });

  expect(screen.getByText('Invalid phone format')).toBeInTheDocument();
});

test('submits valid form', async () => {
  const onSuccess = jest.fn();
  render(<CreateCustomerForm onSuccess={onSuccess} />);

  // Fill form
  fireEvent.change(screen.getByLabelText('Phone Number'), {
    target: { value: '+263771234567' },
  });

  // Submit
  fireEvent.click(screen.getByText('Create Customer'));

  // Assert
  await waitFor(() => expect(onSuccess).toHaveBeenCalled());
});
```

### 8. E2E Tests

**File**: `web/e2e/customer-flow.spec.ts`

```typescript
import { test, expect } from '@playwright/test';

test('complete customer enrollment flow', async ({ page }) => {
  // Login
  await page.goto('/login');
  await page.fill('[name=email]', 'admin@test.com');
  await page.fill('[name=password]', 'password');
  await page.click('button[type=submit]');

  // Navigate to customers
  await page.click('a[href="/customers"]');

  // Create customer
  await page.click('button:has-text("Add Customer")');
  await page.fill('[name=phone]', '+263771234567');
  await page.click('button:has-text("Create")');

  // Assert customer appears in list
  await expect(page.locator('text=+263771234567')).toBeVisible();
});
```

### 9. Test Coverage

Run coverage reports:

```bash
# Go tests
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Frontend tests
npm run test:coverage
```

Target: >80% code coverage

### 10. CI Pipeline

**File**: `.github/workflows/test.yml`

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run migrations
        run: make migrate

      - name: Run tests
        run: go test ./... -v -cover

      - name: Frontend tests
        run: |
          cd web
          npm ci
          npm test
```

## Completion Criteria

- [ ] Unit tests for all packages (>80% coverage)
- [ ] Integration tests passing
- [ ] RLS policy tests verified
- [ ] Performance benchmarks meet targets
- [ ] API tests comprehensive
- [ ] Frontend tests complete
- [ ] E2E tests covering main flows
- [ ] CI pipeline configured
