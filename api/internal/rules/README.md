# Rules Engine - Phase 2 Implementation

## Overview

The Rules Engine is a JsonLogic-based evaluation system that processes events and automatically issues rewards based on configurable rules. It provides high-performance rule evaluation with built-in cap enforcement, cooldown management, and concurrency safety.

## Architecture

### Core Components

1. **JsonLogic Evaluator** (`jsonlogic.go`)
   - Evaluates JsonLogic expressions against event data
   - Supports standard operators: `==`, `!=`, `>`, `>=`, `<`, `<=`
   - Supports logical operators: `all`, `any`, `none`, `!`, `and`, `or`
   - Supports array operators: `in`
   - Variable access with `var` operator
   - Custom time-based operators

2. **Custom Operators** (`custom_operators.go`)
   - `within_days`: Check if event occurred within N days
   - `nth_event_in_period`: Check if this is the Nth event in a time window
   - `distinct_visit_days`: Count unique days with visits

3. **Rules Engine** (`engine.go`)
   - Main entry point for event processing
   - Manages rule matching and evaluation
   - Coordinates with cap enforcement and reward issuance
   - Provides caching for active rules
   - Comprehensive logging for observability

4. **Cap Enforcement** (`caps.go`)
   - Per-user cap checking
   - Global cap checking
   - Cooldown period enforcement
   - Uses database functions for accurate counting

5. **Reward Issuance** (`issuance.go`)
   - Transaction-based issuance creation
   - PostgreSQL advisory locks for concurrency safety
   - Budget reservation and validation
   - Creates issuances in 'reserved' state

6. **Rule Cache** (`cache.go`)
   - Thread-safe in-memory cache
   - TTL-based expiration (default: 5 minutes)
   - Automatic background cleanup
   - Reduces database queries

## Usage

### Initializing the Engine

```go
import (
    "log/slog"
    "github.com/bmachimbira/loyalty/api/internal/rules"
    "github.com/jackc/pgx/v5/pgxpool"
)

// Create rules engine
pool := // ... your database pool
logger := slog.Default()
engine := rules.NewEngine(pool, logger)
```

### Processing Events

```go
// Process an event through the rules engine
issuances, err := engine.ProcessEvent(ctx, event)
if err != nil {
    // Handle error
}

// issuances contains all rewards issued for this event
for _, issuance := range issuances {
    fmt.Printf("Issued reward: %s\n", issuance.ID)
}
```

### Cache Management

```go
// Clear entire cache
engine.InvalidateCache()

// Clear cache for specific tenant
engine.InvalidateCacheForTenant(tenantID)
```

## JsonLogic Examples

### Simple Comparison

Purchase amount >= $20:
```json
{">=": [{"var": "amount"}, 20]}
```

### Multiple Conditions (AND)

Purchase of $20+ in USD:
```json
{
  "all": [
    {">=": [{"var": "amount"}, 20]},
    {"==": [{"var": "currency"}, "USD"]}
  ]
}
```

### Multiple Conditions (OR)

High value purchase OR premium customer:
```json
{
  "any": [
    {">=": [{"var": "amount"}, 100]},
    {"==": [{"var": "customer_tier"}, "premium"]}
  ]
}
```

### Array Membership

Purchase at specific locations:
```json
{
  "in": [{"var": "location"}, ["store_1", "store_2", "store_3"]]
}
```

### Complex Nested Conditions

```json
{
  "all": [
    {">=": [{"var": "amount"}, 20]},
    {"==": [{"var": "event_type"}, "purchase"]},
    {
      "any": [
        {"in": [{"var": "location"}, ["store_1", "store_2"]]},
        {"==": [{"var": "customer_tier"}, "vip"]}
      ]
    }
  ]
}
```

### Custom Operators

Event within last 7 days:
```json
{"within_days": [{"var": "occurred_at"}, 7]}
```

Third purchase in last 30 days:
```json
{"nth_event_in_period": ["purchase", 3, 30]}
```

## Performance

### Targets

- Rule evaluation: **< 25ms per rule**
- Event processing: **< 150ms p95** (including database operations)
- Cache hit rate: **> 90%** for active rules

### Optimization Strategies

1. **Caching**: Active rules cached for 5 minutes
2. **Advisory Locks**: Prevent race conditions without blocking
3. **Efficient Database Functions**: Cap checking uses optimized PostgreSQL functions
4. **Minimal Allocations**: JsonLogic evaluator designed for low GC pressure

### Benchmarks

Run benchmarks with:
```bash
go test -bench=. -benchmem ./internal/rules/
```

Expected results (approximate):
- Simple rule evaluation: ~10-50 µs
- Complex rule evaluation: ~50-200 µs
- Cache hit: ~100-500 ns
- Cache miss: ~1-5 µs

## Concurrency Safety

### Advisory Locks

The engine uses PostgreSQL advisory locks to prevent race conditions:

```go
// Lock key based on (tenant_id, rule_id, customer_id)
lockKey := hashLock(tenantID, ruleID, customerID)
tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey)
```

This ensures:
- No duplicate issuances for the same rule
- Accurate cap enforcement under concurrent load
- No deadlocks (transaction-scoped locks)

### Thread-Safe Cache

The rule cache uses read-write locks for thread safety:
- Multiple concurrent reads allowed
- Writes are exclusive
- Background cleanup runs safely

## Database Functions

The engine relies on these PostgreSQL functions:

### Cap Checking

```sql
-- Get customer issuance count for a rule
SELECT get_customer_rule_issuance_count($tenant_id, $customer_id, $rule_id);

-- Get global issuance count for a rule
SELECT get_rule_global_issuance_count($tenant_id, $rule_id);

-- Check if customer is within cooldown period
SELECT is_within_cooldown($tenant_id, $customer_id, $rule_id, $cooldown_sec);
```

### Budget Management

```sql
-- Reserve budget for an issuance
SELECT reserve_budget($tenant_id, $budget_id, $amount, $currency, $ref_id);

-- Check if budget has capacity
SELECT check_budget_capacity($budget_id, $amount);
```

## Error Handling

The engine handles errors gracefully:

1. **Rule Evaluation Errors**: Logged but don't stop processing other rules
2. **Cap Check Errors**: Rule skipped, error logged
3. **Issuance Errors**: Rule skipped, error logged, no partial state
4. **Database Errors**: Proper transaction rollback

Events are always created successfully even if rules processing fails.

## Testing

### Unit Tests

```bash
go test ./internal/rules/jsonlogic_test.go -v
go test ./internal/rules/cache_test.go -v
```

### Integration Tests

Integration tests require a running PostgreSQL database with the loyalty schema.

### Performance Tests

```bash
go test -bench=BenchmarkSimpleRule -benchtime=10s
go test -bench=BenchmarkComplexRule -benchtime=10s
```

## Monitoring & Observability

### Structured Logging

The engine uses structured logging (slog) with these events:

- `event_id`: Unique event identifier
- `event_type`: Type of event (purchase, visit, etc.)
- `rules_count`: Number of rules evaluated
- `issuances_count`: Number of rewards issued
- `duration_ms`: Processing time in milliseconds
- `error`: Error details if any

### Key Metrics to Track

1. **Performance**:
   - Rule evaluation time
   - Total event processing time
   - Cache hit rate

2. **Business**:
   - Rules triggered per event
   - Issuances created per event
   - Cap rejection rate
   - Budget exhaustion rate

3. **Errors**:
   - Rule evaluation errors
   - Cap check errors
   - Issuance errors

## Future Enhancements

Potential improvements for Phase 3+:

1. **Advanced Operators**:
   - Date/time manipulation
   - Mathematical operations
   - String operations
   - Regex matching

2. **Performance**:
   - Distributed caching (Redis)
   - Rule compilation for frequently used rules
   - Batch event processing

3. **Features**:
   - A/B testing support
   - Rule versioning
   - Rule simulation/testing tools
   - Real-time rule updates without cache clear

## Dependencies

- **pgx/v5**: PostgreSQL driver and connection pooling
- **slog**: Structured logging (Go 1.21+)

## References

- [JsonLogic Specification](http://jsonlogic.com/)
- [PostgreSQL Advisory Locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS)
- [Phase 2 Requirements](/agents/rules-engine.md)
