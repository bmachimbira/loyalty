package rules

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// BenchmarkSimpleRule benchmarks a simple comparison rule
// Target: < 25ms per evaluation
func BenchmarkSimpleRule(b *testing.B) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	logic := json.RawMessage(`{">=": [{"var": "amount"}, 20]}`)
	data := map[string]interface{}{
		"amount": 25.0,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = e.Evaluate(ctx, logic, data)
	}
}

// BenchmarkComplexRule benchmarks a complex multi-condition rule
func BenchmarkComplexRule(b *testing.B) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	logic := json.RawMessage(`{
		"all": [
			{">=": [{"var": "amount"}, 20]},
			{"==": [{"var": "event_type"}, "purchase"]},
			{"==": [{"var": "currency"}, "USD"]},
			{"in": [{"var": "location"}, ["store_1", "store_2", "store_3", "store_4", "store_5"]]}
		]
	}`)
	data := map[string]interface{}{
		"amount":     25.0,
		"event_type": "purchase",
		"currency":   "USD",
		"location":   "store_3",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = e.Evaluate(ctx, logic, data)
	}
}

// BenchmarkNestedConditions benchmarks deeply nested conditions
func BenchmarkNestedConditions(b *testing.B) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	logic := json.RawMessage(`{
		"any": [
			{
				"all": [
					{">=": [{"var": "amount"}, 50]},
					{"==": [{"var": "currency"}, "USD"]},
					{"in": [{"var": "category"}, ["electronics", "appliances"]]}
				]
			},
			{
				"all": [
					{">=": [{"var": "amount"}, 100]},
					{"==": [{"var": "currency"}, "ZWL"]},
					{"in": [{"var": "category"}, ["groceries", "food"]]}
				]
			}
		]
	}`)
	data := map[string]interface{}{
		"amount":   60.0,
		"currency": "USD",
		"category": "electronics",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = e.Evaluate(ctx, logic, data)
	}
}

// BenchmarkCacheHit benchmarks cache performance
func BenchmarkCacheHit(b *testing.B) {
	cache := NewRuleCache(5 * time.Minute)
	rules := make([]db.Rule, 10)

	cache.Set("test-key", rules)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("test-key")
	}
}

// BenchmarkCacheMiss benchmarks cache miss performance
func BenchmarkCacheMiss(b *testing.B) {
	cache := NewRuleCache(5 * time.Minute)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("non-existent-key")
	}
}

// BenchmarkVarOperator benchmarks variable access performance
func BenchmarkVarOperator(b *testing.B) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	logic := json.RawMessage(`{"var": "deeply.nested.property.value"}`)
	data := map[string]interface{}{
		"properties": map[string]interface{}{
			"amount": 25.0,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = e.Evaluate(ctx, logic, data)
	}
}

// BenchmarkMultipleRulesEvaluation simulates evaluating multiple rules against one event
func BenchmarkMultipleRulesEvaluation(b *testing.B) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	rules := []json.RawMessage{
		json.RawMessage(`{">=": [{"var": "amount"}, 20]}`),
		json.RawMessage(`{"==": [{"var": "event_type"}, "purchase"]}`),
		json.RawMessage(`{"in": [{"var": "location"}, ["store_1", "store_2"]]}`),
		json.RawMessage(`{"all": [{">=": [{"var": "amount"}, 10]}, {"<": [{"var": "amount"}, 100]}]}`),
		json.RawMessage(`{"any": [{"==": [{"var": "currency"}, "USD"]}, {"==": [{"var": "currency"}, "ZWL"]}]}`),
	}

	data := map[string]interface{}{
		"amount":     25.0,
		"event_type": "purchase",
		"location":   "store_1",
		"currency":   "USD",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, logic := range rules {
			_, _ = e.Evaluate(ctx, logic, data)
		}
	}
}
