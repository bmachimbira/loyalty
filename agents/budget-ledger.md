# Budget & Ledger Agent

## Mission
Implement budget tracking, ledger entries, and financial controls for the loyalty platform.

## Prerequisites
- Go 1.21+
- Understanding of double-entry accounting
- PostgreSQL transactions

## Tasks

### 1. Budget Service

**File**: `api/internal/budget/service.go`

```go
package budget

import (
    "context"
    "errors"
)

type Service struct {
    queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
    return &Service{queries: queries}
}

// ReserveBudget reserves amount for issuance
func (s *Service) ReserveBudget(ctx context.Context, budgetID string, amount float64, currency string, issuanceID string) error {
    // Check budget capacity
    budget, err := s.queries.GetBudgetByID(ctx, ...)
    if err != nil {
        return err
    }

    // Verify currency matches
    if budget.Currency != currency {
        return errors.New("currency mismatch")
    }

    // Check hard cap
    newBalance := budget.Balance + amount
    if newBalance > budget.HardCap {
        return errors.New("hard cap exceeded")
    }

    // Check soft cap (warning)
    if newBalance > budget.SoftCap {
        // Log warning, trigger alert
        s.alertSoftCapExceeded(ctx, budgetID, newBalance, budget.SoftCap)
    }

    // Update balance
    err = s.queries.UpdateBudgetBalance(ctx, db.UpdateBudgetBalanceParams{
        ID:       budgetID,
        TenantID: budget.TenantID,
        Amount:   amount,
    })
    if err != nil {
        return err
    }

    // Insert ledger entry
    _, err = s.queries.InsertLedgerEntry(ctx, db.InsertLedgerEntryParams{
        TenantID:  budget.TenantID,
        BudgetID:  budgetID,
        EntryType: "reserve",
        Currency:  currency,
        Amount:    amount,
        RefType:   "issuance",
        RefID:     issuanceID,
    })

    return err
}

// ChargeReservation converts reservation to charge (on redemption)
func (s *Service) ChargeReservation(ctx context.Context, issuanceID string) error {
    // Get issuance
    issuance, err := s.queries.GetIssuanceByID(ctx, ...)
    if err != nil {
        return err
    }

    // Find original reserve entry
    // Create charge entry (no balance change, just ledger record)
    _, err = s.queries.InsertLedgerEntry(ctx, db.InsertLedgerEntryParams{
        TenantID:  issuance.TenantID,
        BudgetID:  issuance.BudgetID,
        EntryType: "charge",
        Currency:  issuance.Currency,
        Amount:    issuance.CostAmount,
        RefType:   "issuance",
        RefID:     issuanceID,
    })

    return err
}

// ReleaseReservation returns funds (on expiry/cancel)
func (s *Service) ReleaseReservation(ctx context.Context, issuanceID string) error {
    issuance, err := s.queries.GetIssuanceByID(ctx, ...)
    if err != nil {
        return err
    }

    // Decrease balance
    err = s.queries.UpdateBudgetBalance(ctx, db.UpdateBudgetBalanceParams{
        ID:       issuance.BudgetID,
        TenantID: issuance.TenantID,
        Amount:   -issuance.CostAmount, // Negative to decrease
    })
    if err != nil {
        return err
    }

    // Insert release entry
    _, err = s.queries.InsertLedgerEntry(ctx, db.InsertLedgerEntryParams{
        TenantID:  issuance.TenantID,
        BudgetID:  issuance.BudgetID,
        EntryType: "release",
        Currency:  issuance.Currency,
        Amount:    issuance.CostAmount,
        RefType:   "issuance",
        RefID:     issuanceID,
    })

    return err
}
```

### 2. Budget Topup

**File**: `api/internal/budget/topup.go`

```go
package budget

func (s *Service) TopupBudget(ctx context.Context, budgetID string, amount float64) error {
    budget, err := s.queries.GetBudgetByID(ctx, ...)
    if err != nil {
        return err
    }

    // Topup doesn't affect balance (balance is reserved/charged amounts)
    // Just record the fund entry for audit

    _, err = s.queries.InsertLedgerEntry(ctx, db.InsertLedgerEntryParams{
        TenantID:  budget.TenantID,
        BudgetID:  budgetID,
        EntryType: "fund",
        Currency:  budget.Currency,
        Amount:    amount,
        RefType:   "topup",
        RefID:     nil,
    })

    return err
}
```

### 3. Ledger Queries

Add to `queries/budgets.sql`:

```sql
-- name: GetLedgerBalance :one
SELECT
  SUM(CASE WHEN entry_type = 'fund' THEN amount ELSE 0 END) as total_funded,
  SUM(CASE WHEN entry_type = 'reserve' THEN amount ELSE 0 END) as total_reserved,
  SUM(CASE WHEN entry_type = 'charge' THEN amount ELSE 0 END) as total_charged,
  SUM(CASE WHEN entry_type = 'release' THEN amount ELSE 0 END) as total_released,
  SUM(CASE WHEN entry_type = 'fund' THEN amount ELSE 0 END) -
  SUM(CASE WHEN entry_type IN ('charge', 'expire') THEN amount ELSE 0 END) as available
FROM ledger_entries
WHERE tenant_id = $1 AND budget_id = $2;

-- name: GetBudgetUtilization :one
SELECT
  b.hard_cap,
  b.soft_cap,
  b.balance as reserved,
  (b.balance / NULLIF(b.hard_cap, 0) * 100) as utilization_percent
FROM budgets b
WHERE b.id = $1 AND b.tenant_id = $2;
```

### 4. Reconciliation

**File**: `api/internal/budget/reconciliation.go`

```go
package budget

// ReconcileBudget verifies budget.balance matches ledger
func (s *Service) ReconcileBudget(ctx context.Context, budgetID string) error {
    // Calculate from ledger
    ledgerBalance, err := s.queries.GetLedgerBalance(ctx, ...)
    if err != nil {
        return err
    }

    // Compare with budget.balance
    budget, err := s.queries.GetBudgetByID(ctx, ...)
    if err != nil {
        return err
    }

    expectedBalance := ledgerBalance.TotalReserved - ledgerBalance.TotalReleased

    if budget.Balance != expectedBalance {
        // Log discrepancy
        return errors.New("balance mismatch")
    }

    return nil
}

// Run reconciliation daily as background job
```

### 5. Multi-Currency Support

**File**: `api/internal/budget/currency.go`

```go
package budget

// Budgets are per-currency
// Validate operations match budget currency

func ValidateCurrency(budget *db.Budget, currency string) error {
    if budget.Currency != currency {
        return errors.New("currency mismatch")
    }
    return nil
}

// Support ZWG and USD
const (
    CurrencyZWG = "ZWG"
    CurrencyUSD = "USD"
)
```

### 6. Alerts

**File**: `api/internal/budget/alerts.go`

```go
package budget

func (s *Service) alertSoftCapExceeded(ctx context.Context, budgetID string, balance, softCap float64) {
    // Send notification to tenant admins
    // Log alert event

    alert := map[string]interface{}{
        "type":     "soft_cap_exceeded",
        "budgetID": budgetID,
        "balance":  balance,
        "softCap":  softCap,
        "percent":  (balance / softCap) * 100,
    }

    // Trigger webhook or email notification
}

func (s *Service) alertHardCapApproaching(ctx context.Context, budgetID string, balance, hardCap float64) {
    // Alert when within 90% of hard cap
    if balance >= (hardCap * 0.9) {
        // Send warning
    }
}
```

### 7. Period Budgets

**File**: `api/internal/budget/period.go`

```go
package budget

// For monthly budgets, reset balance at start of month
func (s *Service) ResetMonthlyBudgets(ctx context.Context) error {
    // Find budgets with period = 'monthly'
    budgets, err := s.queries.GetMonthlyBudgets(ctx)
    if err != nil {
        return err
    }

    for _, budget := range budgets {
        // Reset balance to 0
        err := s.queries.ResetBudgetBalance(ctx, budget.ID)
        if err != nil {
            continue
        }

        // Log reset event
    }

    return nil
}

// Run as cron job on 1st of each month
```

### 8. Reporting

**File**: `api/internal/budget/reports.go`

```go
package budget

type BudgetReport struct {
    BudgetID        string
    Name            string
    Currency        string
    TotalFunded     float64
    TotalReserved   float64
    TotalCharged    float64
    Available       float64
    Utilization     float64
    IssuanceCount   int
    RedemptionCount int
}

func (s *Service) GenerateReport(ctx context.Context, budgetID string, from, to time.Time) (*BudgetReport, error) {
    // Query ledger entries
    // Aggregate by entry_type
    // Calculate metrics
    // Return report
}
```

### 9. Testing

**File**: `api/internal/budget/service_test.go`

Test cases:
- [ ] Reserve budget (success)
- [ ] Reserve budget (hard cap exceeded)
- [ ] Soft cap warning triggered
- [ ] Charge reservation
- [ ] Release reservation
- [ ] Balance reconciliation
- [ ] Currency validation
- [ ] Monthly reset
- [ ] Concurrent reservations (race conditions)

## Completion Criteria

- [ ] Budget reservation implemented
- [ ] Ledger entries recorded correctly
- [ ] Hard cap enforcement working
- [ ] Soft cap alerts working
- [ ] Charge/release logic correct
- [ ] Reconciliation passing
- [ ] Multi-currency support
- [ ] Period budgets working
- [ ] Tests passing (>80% coverage)
