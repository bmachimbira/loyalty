package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/logging"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Engine is the rules evaluation engine
type Engine struct {
	pool      *pgxpool.Pool
	queries   *db.Queries
	evaluator *Evaluator
	cache     *RuleCache
	logger    *logging.Logger
}

// NewEngine creates a new rules engine
func NewEngine(pool *pgxpool.Pool, logger *logging.Logger) *Engine {
	queries := db.New(pool)
	customOps := NewCustomOperators(pool)
	evaluator := NewEvaluator(customOps)
	cache := NewRuleCache(5 * time.Minute) // 5 minute TTL

	return &Engine{
		pool:      pool,
		queries:   queries,
		evaluator: evaluator,
		cache:     cache,
		logger:    logger,
	}
}

// ProcessEvent evaluates all matching rules for an event and issues rewards
func (e *Engine) ProcessEvent(ctx context.Context, event db.Event) ([]db.Issuance, error) {
	startTime := time.Now()

	// Get active rules for this event type
	rules, err := e.getMatchingRules(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("failed to get matching rules: %w", err)
	}

	if len(rules) == 0 {
		e.logger.Debug("no matching rules for event",
			"event_id", event.ID,
			"event_type", event.EventType,
		)
		return []db.Issuance{}, nil
	}

	e.logger.Info("evaluating rules for event",
		"event_id", event.ID,
		"event_type", event.EventType,
		"rules_count", len(rules),
	)

	var issuances []db.Issuance

	// Evaluate each rule
	for _, rule := range rules {
		ruleStartTime := time.Now()

		triggered, err := e.evaluateRule(ctx, rule, event)
		if err != nil {
			e.logger.Warn("rule evaluation error",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"error", err,
			)
			continue
		}

		if !triggered {
			e.logger.Debug("rule not triggered",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
			)
			continue
		}

		e.logger.Info("rule triggered",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
		)

		// Check caps
		passed, err := e.checkCaps(ctx, rule, event)
		if err != nil {
			e.logger.Warn("cap check error",
				"rule_id", rule.ID,
				"error", err,
			)
			continue
		}

		if !passed {
			e.logger.Info("rule caps exceeded",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
			)
			continue
		}

		// Issue reward
		issuance, err := e.issueReward(ctx, rule, event)
		if err != nil {
			e.logger.Error("reward issuance error",
				"rule_id", rule.ID,
				"error", err,
			)
			continue
		}

		e.logger.Info("reward issued",
			"rule_id", rule.ID,
			"issuance_id", issuance.ID,
			"customer_id", event.CustomerID,
			"duration_ms", time.Since(ruleStartTime).Milliseconds(),
		)

		issuances = append(issuances, *issuance)
	}

	e.logger.Info("event processing completed",
		"event_id", event.ID,
		"issuances_count", len(issuances),
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return issuances, nil
}

// getMatchingRules retrieves active rules for an event type (with caching)
func (e *Engine) getMatchingRules(ctx context.Context, event db.Event) ([]db.Rule, error) {
	// Generate cache key
	cacheKey := fmt.Sprintf("%s:%s", uuidToString(event.TenantID), event.EventType)

	// Check cache first
	if cachedRules, found := e.cache.Get(cacheKey); found {
		e.logger.Debug("cache hit for rules", "cache_key", cacheKey)
		return cachedRules, nil
	}

	// Query database
	rules, err := e.queries.GetActiveRulesForEvent(ctx, db.GetActiveRulesForEventParams{
		TenantID:  event.TenantID,
		EventType: event.EventType,
	})
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// Cache the results
	e.cache.Set(cacheKey, rules)
	e.logger.Debug("cached rules", "cache_key", cacheKey, "count", len(rules))

	return rules, nil
}

// evaluateRule evaluates a single rule against an event
func (e *Engine) evaluateRule(ctx context.Context, rule db.Rule, event db.Event) (bool, error) {
	// Build data context for JsonLogic evaluation
	data := make(map[string]interface{})

	// Add event fields
	data["event_type"] = event.EventType
	data["tenant_id"] = uuidToString(event.TenantID)
	data["customer_id"] = uuidToString(event.CustomerID)

	// Parse occurred_at
	if event.OccurredAt.Valid {
		data["occurred_at"] = event.OccurredAt.Time
	}

	// Parse and add properties as a nested object
	if len(event.Properties) > 0 {
		var properties map[string]interface{}
		if err := json.Unmarshal(event.Properties, &properties); err != nil {
			return false, fmt.Errorf("failed to parse event properties: %w", err)
		}
		data["properties"] = properties

		// Also add properties at root level for easier access
		for k, v := range properties {
			data[k] = v
		}
	}

	// Evaluate the rule conditions
	result, err := e.evaluator.Evaluate(ctx, rule.Conditions, data)
	if err != nil {
		return false, fmt.Errorf("evaluation failed: %w", err)
	}

	return result, nil
}

// InvalidateCache clears the entire rule cache
func (e *Engine) InvalidateCache() {
	e.cache.Clear()
	e.logger.Info("rule cache invalidated")
}

// InvalidateCacheForTenant clears cache entries for a specific tenant
func (e *Engine) InvalidateCacheForTenant(tenantID string) {
	// Since we don't have a way to iterate cache keys, we'll just clear everything
	// In a production system, you might want to use a more sophisticated caching strategy
	e.cache.Clear()
	e.logger.Info("cache invalidated for tenant", "tenant_id", tenantID)
}

// Helper functions

// uuidToString converts pgtype.UUID to string
func uuidToString(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}
	// Convert [16]byte to UUID string format
	b := uuid.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// EngineStats contains statistics about the rules engine
type EngineStats struct {
	CacheSize       int
	CacheHits       int64
	CacheMisses     int64
	RulesEvaluated  int64
	IssuancesIssued int64
	Errors          int64
}

// GetStats returns current engine statistics
func (e *Engine) GetStats() EngineStats {
	return EngineStats{
		CacheSize: e.cache.Size(),
		// Note: We would need to add counters to track hits/misses/etc
		// This is a placeholder implementation
	}
}
