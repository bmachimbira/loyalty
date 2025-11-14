# Rules Engine Agent

## Mission
Implement the JsonLogic-based rules evaluation engine with cap enforcement and performance optimization.

## Prerequisites
- Go 1.21+
- Understanding of JsonLogic specification
- Knowledge of PostgreSQL transactions and locking

## Tasks

### 1. JsonLogic Evaluator

**File**: `api/internal/rules/jsonlogic.go`

```go
package rules

import (
    "encoding/json"
)

type Evaluator struct{}

// Supported operators:
// - Comparison: ==, !=, >, >=, <, <=
// - Logic: all, any, none
// - Array: in
// - Variable: var
// - Custom: within_days, nth_event_in_period, distinct_visit_days

func (e *Evaluator) Evaluate(logic json.RawMessage, data map[string]interface{}) (bool, error) {
    // Parse logic
    // Evaluate recursively
    // Return boolean result
}
```

**Implementation checklist**:
- [ ] Parse JsonLogic conditions
- [ ] Implement comparison operators (==, >=, <=, in)
- [ ] Implement logical operators (all, any)
- [ ] Implement var operator (access event properties)
- [ ] Add custom operators:
  - `within_days`: Check if event within N days
  - `nth_event_in_period`: Count events in time window
  - `distinct_visit_days`: Count unique visit days

### 2. Custom Operators

**File**: `api/internal/rules/custom_operators.go`

```go
package rules

import (
    "context"
    "time"
)

type CustomOperators struct {
    queries *db.Queries
}

// within_days checks if occurred_at is within N days from now
func (c *CustomOperators) WithinDays(occurredAt time.Time, days int) bool {
    return occurredAt.After(time.Now().AddDate(0, 0, -days))
}

// nth_event_in_period counts events for customer in time window
// Returns true if this is the Nth event
func (c *CustomOperators) NthEventInPeriod(
    ctx context.Context,
    tenantID, customerID string,
    eventType string,
    n, periodDays int,
) (bool, error) {
    // Query events table
    // Count events in last periodDays
    // Return count == n
}

// distinct_visit_days counts unique days with visits
func (c *CustomOperators) DistinctVisitDays(
    ctx context.Context,
    tenantID, customerID string,
    periodDays int,
) (int, error) {
    // Query events with event_type='visit'
    // Count distinct dates
}
```

### 3. Rule Matching Engine

**File**: `api/internal/rules/engine.go`

```go
package rules

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Engine struct {
    pool      *pgxpool.Pool
    queries   *db.Queries
    evaluator *Evaluator
    cache     *RuleCache
}

func NewEngine(pool *pgxpool.Pool, queries *db.Queries) *Engine {
    return &Engine{
        pool:      pool,
        queries:   queries,
        evaluator: &Evaluator{},
        cache:     NewRuleCache(),
    }
}

// ProcessEvent evaluates all matching rules for an event
func (e *Engine) ProcessEvent(ctx context.Context, event *db.Event) ([]*db.Issuance, error) {
    // 1. Get active rules for event_type
    rules, err := e.getMatchingRules(ctx, event)
    if err != nil {
        return nil, err
    }

    var issuances []*db.Issuance

    // 2. Evaluate each rule
    for _, rule := range rules {
        triggered, err := e.evaluateRule(ctx, rule, event)
        if err != nil {
            continue // Log error, continue with other rules
        }

        if !triggered {
            continue
        }

        // 3. Check caps
        if !e.checkCaps(ctx, rule, event) {
            continue
        }

        // 4. Issue reward
        issuance, err := e.issueReward(ctx, rule, event)
        if err != nil {
            continue // Log error
        }

        issuances = append(issuances, issuance)
    }

    return issuances, nil
}

func (e *Engine) getMatchingRules(ctx context.Context, event *db.Event) ([]*db.Rule, error) {
    // Check cache first
    cacheKey := fmt.Sprintf("%s:%s", event.TenantID, event.EventType)
    if rules := e.cache.Get(cacheKey); rules != nil {
        return rules, nil
    }

    // Query database
    rules, err := e.queries.GetActiveRulesForEvent(ctx, db.GetActiveRulesForEventParams{
        TenantID:  event.TenantID,
        EventType: event.EventType,
    })

    if err != nil {
        return nil, err
    }

    // Cache for 30 seconds
    e.cache.Set(cacheKey, rules, 30*time.Second)

    return rules, nil
}

func (e *Engine) evaluateRule(ctx context.Context, rule *db.Rule, event *db.Event) (bool, error) {
    // Build data context
    data := map[string]interface{}{
        "event_type":  event.EventType,
        "occurred_at": event.OccurredAt,
        "properties":  event.Properties,
    }

    // Evaluate JsonLogic conditions
    return e.evaluator.Evaluate(rule.Conditions, data)
}
```

### 4. Cap Enforcement

**File**: `api/internal/rules/caps.go`

```go
package rules

import (
    "context"
    "time"
)

// checkCaps verifies per_user_cap, global_cap, and cool_down_sec
func (e *Engine) checkCaps(ctx context.Context, rule *db.Rule, event *db.Event) bool {
    // Per-user cap
    if rule.PerUserCap > 0 {
        count, err := e.queries.GetCustomerRuleIssuanceCount(ctx, db.GetCustomerRuleIssuanceCountParams{
            TenantID:   event.TenantID,
            CustomerID: event.CustomerID,
            RuleID:     rule.ID,
        })
        if err != nil || count >= rule.PerUserCap {
            return false
        }
    }

    // Global cap
    if rule.GlobalCap.Valid && rule.GlobalCap.Int32 > 0 {
        count, err := e.queries.GetRuleIssuanceCount(ctx, db.GetRuleIssuanceCountParams{
            TenantID: event.TenantID,
            RuleID:   rule.ID,
            Since:    time.Time{}, // All time
        })
        if err != nil || count >= int64(rule.GlobalCap.Int32) {
            return false
        }
    }

    // Cool-down
    if rule.CoolDownSec > 0 {
        // Check last issuance time
        lastIssuance, err := e.queries.GetLastIssuanceForCustomerRule(ctx, db.GetLastIssuanceForCustomerRuleParams{
            TenantID:   event.TenantID,
            CustomerID: event.CustomerID,
            RuleID:     rule.ID,
        })
        if err == nil && lastIssuance != nil {
            cooldownExpiry := lastIssuance.IssuedAt.Add(time.Duration(rule.CoolDownSec) * time.Second)
            if time.Now().Before(cooldownExpiry) {
                return false
            }
        }
    }

    return true
}
```

### 5. Reward Issuance

**File**: `api/internal/rules/issuance.go`

```go
package rules

import (
    "context"
    "github.com/jackc/pgx/v5"
)

func (e *Engine) issueReward(ctx context.Context, rule *db.Rule, event *db.Event) (*db.Issuance, error) {
    // Start transaction
    tx, err := e.pool.Begin(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback(ctx)

    qtx := e.queries.WithTx(tx)

    // Advisory lock on (tenant_id, rule_id, customer_id)
    lockKey := hashLock(event.TenantID, rule.ID, event.CustomerID)
    _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey)
    if err != nil {
        return nil, err
    }

    // Re-check caps (race condition protection)
    if !e.checkCaps(ctx, rule, event) {
        return nil, errors.New("cap exceeded")
    }

    // Get reward details
    reward, err := qtx.GetRewardByID(ctx, db.GetRewardByIDParams{
        TenantID: event.TenantID,
        ID:       rule.RewardID,
    })
    if err != nil {
        return nil, err
    }

    // Reserve budget if campaign has budget
    if rule.CampaignID.Valid {
        campaign, err := qtx.GetCampaignByID(ctx, db.GetCampaignByIDParams{
            TenantID: event.TenantID,
            ID:       rule.CampaignID.UUID,
        })
        if err != nil {
            return nil, err
        }

        if campaign.BudgetID.Valid {
            // Check and reserve budget
            success, err := reserveBudget(ctx, qtx, campaign.BudgetID.UUID, reward.FaceValue)
            if err != nil || !success {
                return nil, errors.New("budget exceeded")
            }
        }
    }

    // Create issuance (reserved state)
    issuance, err := qtx.ReserveIssuance(ctx, db.ReserveIssuanceParams{
        TenantID:   event.TenantID,
        CustomerID: event.CustomerID,
        CampaignID: rule.CampaignID,
        RewardID:   reward.ID,
        Currency:   reward.Currency,
        FaceAmount: reward.FaceValue,
        CostAmount: reward.FaceValue,
    })
    if err != nil {
        return nil, err
    }

    // Commit transaction
    if err := tx.Commit(ctx); err != nil {
        return nil, err
    }

    // Asynchronously process reward (move to issued state)
    go e.processRewardAsync(issuance.ID)

    return issuance, nil
}

func hashLock(tenantID, ruleID, customerID string) int64 {
    // Hash to int64 for advisory lock
    h := fnv.New64a()
    h.Write([]byte(tenantID + ruleID + customerID))
    return int64(h.Sum64())
}
```

### 6. Rule Cache

**File**: `api/internal/rules/cache.go`

```go
package rules

import (
    "sync"
    "time"
)

type RuleCache struct {
    mu    sync.RWMutex
    items map[string]*cacheItem
}

type cacheItem struct {
    rules     []*db.Rule
    expiresAt time.Time
}

func NewRuleCache() *RuleCache {
    cache := &RuleCache{
        items: make(map[string]*cacheItem),
    }
    go cache.cleanupExpired()
    return cache
}

func (c *RuleCache) Get(key string) []*db.Rule {
    c.mu.RLock()
    defer c.mu.RUnlock()

    item, ok := c.items[key]
    if !ok || time.Now().After(item.expiresAt) {
        return nil
    }

    return item.rules
}

func (c *RuleCache) Set(key string, rules []*db.Rule, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.items[key] = &cacheItem{
        rules:     rules,
        expiresAt: time.Now().Add(ttl),
    }
}

func (c *RuleCache) cleanupExpired() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        c.mu.Lock()
        now := time.Now()
        for key, item := range c.items {
            if now.After(item.expiresAt) {
                delete(c.items, key)
            }
        }
        c.mu.Unlock()
    }
}
```

### 7. Testing

**File**: `api/internal/rules/engine_test.go`

Test cases:
- [ ] Simple condition (amount >= 20)
- [ ] Complex condition (all + in)
- [ ] Custom operators (within_days, nth_event)
- [ ] Per-user cap enforcement
- [ ] Global cap enforcement
- [ ] Cool-down enforcement
- [ ] Concurrent events (race conditions)
- [ ] Budget exhaustion

**File**: `api/internal/rules/jsonlogic_test.go`

Test JsonLogic evaluation:
```go
func TestEvaluator_SimpleComparison(t *testing.T) {
    e := &Evaluator{}

    logic := json.RawMessage(`{">=": [{"var": "amount"}, 20]}`)
    data := map[string]interface{}{"amount": 25.50}

    result, err := e.Evaluate(logic, data)
    assert.NoError(t, err)
    assert.True(t, result)
}
```

### 8. Performance Optimization

- [ ] Index on rules(tenant_id, event_type, active)
- [ ] Rule cache with 30s TTL
- [ ] Advisory locks for concurrency
- [ ] Minimize database queries
- [ ] Batch rule evaluation

**Performance target**: < 25ms per event

### 9. Monitoring

**File**: `api/internal/rules/metrics.go`

```go
package rules

// Track metrics:
// - Rule evaluation time
// - Rules matched per event
// - Cap rejections
// - Budget rejections
// - Issuance success/failure
```

## Completion Criteria

- [ ] JsonLogic evaluator complete
- [ ] Custom operators implemented
- [ ] Cap enforcement working
- [ ] Concurrency handling correct
- [ ] Rule cache implemented
- [ ] Tests passing (>80% coverage)
- [ ] Performance target met (<25ms)
- [ ] Advisory locks preventing race conditions
