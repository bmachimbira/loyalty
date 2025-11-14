package rules

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestEvaluator_SimpleComparison(t *testing.T) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	tests := []struct {
		name     string
		logic    string
		data     map[string]interface{}
		expected bool
		wantErr  bool
	}{
		{
			name:     "equal - true",
			logic:    `{"==": [{"var": "amount"}, 20]}`,
			data:     map[string]interface{}{"amount": 20.0},
			expected: true,
		},
		{
			name:     "equal - false",
			logic:    `{"==": [{"var": "amount"}, 20]}`,
			data:     map[string]interface{}{"amount": 25.0},
			expected: false,
		},
		{
			name:     "not equal - true",
			logic:    `{"!=": [{"var": "status"}, "inactive"]}`,
			data:     map[string]interface{}{"status": "active"},
			expected: true,
		},
		{
			name:     "greater than - true",
			logic:    `{">": [{"var": "amount"}, 20]}`,
			data:     map[string]interface{}{"amount": 25.0},
			expected: true,
		},
		{
			name:     "greater than - false",
			logic:    `{">": [{"var": "amount"}, 20]}`,
			data:     map[string]interface{}{"amount": 15.0},
			expected: false,
		},
		{
			name:     "greater than or equal - true (greater)",
			logic:    `{">=": [{"var": "amount"}, 20]}`,
			data:     map[string]interface{}{"amount": 25.0},
			expected: true,
		},
		{
			name:     "greater than or equal - true (equal)",
			logic:    `{">=": [{"var": "amount"}, 20]}`,
			data:     map[string]interface{}{"amount": 20.0},
			expected: true,
		},
		{
			name:     "greater than or equal - false",
			logic:    `{">=": [{"var": "amount"}, 20]}`,
			data:     map[string]interface{}{"amount": 15.0},
			expected: false,
		},
		{
			name:     "less than - true",
			logic:    `{"<": [{"var": "amount"}, 20]}`,
			data:     map[string]interface{}{"amount": 15.0},
			expected: true,
		},
		{
			name:     "less than or equal - true",
			logic:    `{"<=": [{"var": "amount"}, 20]}`,
			data:     map[string]interface{}{"amount": 20.0},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Evaluate(ctx, json.RawMessage(tt.logic), tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_LogicalOperators(t *testing.T) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	tests := []struct {
		name     string
		logic    string
		data     map[string]interface{}
		expected bool
	}{
		{
			name:     "all - true",
			logic:    `{"all": [{">=": [{"var": "amount"}, 20]}, {"==": [{"var": "currency"}, "USD"]}]}`,
			data:     map[string]interface{}{"amount": 25.0, "currency": "USD"},
			expected: true,
		},
		{
			name:     "all - false (first fails)",
			logic:    `{"all": [{">=": [{"var": "amount"}, 20]}, {"==": [{"var": "currency"}, "USD"]}]}`,
			data:     map[string]interface{}{"amount": 15.0, "currency": "USD"},
			expected: false,
		},
		{
			name:     "all - false (second fails)",
			logic:    `{"all": [{">=": [{"var": "amount"}, 20]}, {"==": [{"var": "currency"}, "USD"]}]}`,
			data:     map[string]interface{}{"amount": 25.0, "currency": "EUR"},
			expected: false,
		},
		{
			name:     "any - true (first passes)",
			logic:    `{"any": [{">=": [{"var": "amount"}, 20]}, {"==": [{"var": "currency"}, "USD"]}]}`,
			data:     map[string]interface{}{"amount": 25.0, "currency": "EUR"},
			expected: true,
		},
		{
			name:     "any - true (second passes)",
			logic:    `{"any": [{">=": [{"var": "amount"}, 20]}, {"==": [{"var": "currency"}, "USD"]}]}`,
			data:     map[string]interface{}{"amount": 15.0, "currency": "USD"},
			expected: true,
		},
		{
			name:     "any - false (both fail)",
			logic:    `{"any": [{">=": [{"var": "amount"}, 20]}, {"==": [{"var": "currency"}, "USD"]}]}`,
			data:     map[string]interface{}{"amount": 15.0, "currency": "EUR"},
			expected: false,
		},
		{
			name:     "none - true",
			logic:    `{"none": [{"<": [{"var": "amount"}, 10]}, {"==": [{"var": "currency"}, "EUR"]}]}`,
			data:     map[string]interface{}{"amount": 25.0, "currency": "USD"},
			expected: true,
		},
		{
			name:     "none - false",
			logic:    `{"none": [{"<": [{"var": "amount"}, 10]}, {"==": [{"var": "currency"}, "EUR"]}]}`,
			data:     map[string]interface{}{"amount": 5.0, "currency": "USD"},
			expected: false,
		},
		{
			name:     "not - true",
			logic:    `{"!": [{"<": [{"var": "amount"}, 10]}]}`,
			data:     map[string]interface{}{"amount": 25.0},
			expected: true,
		},
		{
			name:     "not - false",
			logic:    `{"!": [{"<": [{"var": "amount"}, 10]}]}`,
			data:     map[string]interface{}{"amount": 5.0},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Evaluate(ctx, json.RawMessage(tt.logic), tt.data)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_ArrayOperators(t *testing.T) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	tests := []struct {
		name     string
		logic    string
		data     map[string]interface{}
		expected bool
	}{
		{
			name:     "in - true",
			logic:    `{"in": [{"var": "product"}, ["apple", "banana", "orange"]]}`,
			data:     map[string]interface{}{"product": "banana"},
			expected: true,
		},
		{
			name:     "in - false",
			logic:    `{"in": [{"var": "product"}, ["apple", "banana", "orange"]]}`,
			data:     map[string]interface{}{"product": "grape"},
			expected: false,
		},
		{
			name:     "in with var array - true",
			logic:    `{"in": ["USD", {"var": "currencies"}]}`,
			data:     map[string]interface{}{"currencies": []interface{}{"USD", "EUR", "GBP"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Evaluate(ctx, json.RawMessage(tt.logic), tt.data)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_VariableAccess(t *testing.T) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	tests := []struct {
		name     string
		logic    string
		data     map[string]interface{}
		expected bool
	}{
		{
			name:     "var - simple",
			logic:    `{"==": [{"var": "status"}, "active"]}`,
			data:     map[string]interface{}{"status": "active"},
			expected: true,
		},
		{
			name:     "var - from properties",
			logic:    `{">=": [{"var": "amount"}, 20]}`,
			data:     map[string]interface{}{"properties": map[string]interface{}{"amount": 25.0}},
			expected: true,
		},
		{
			name:     "var - with default (exists)",
			logic:    `{"==": [{"var": ["status", "unknown"]}, "active"]}`,
			data:     map[string]interface{}{"status": "active"},
			expected: true,
		},
		{
			name:     "var - with default (missing)",
			logic:    `{"==": [{"var": ["status", "unknown"]}, "unknown"]}`,
			data:     map[string]interface{}{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Evaluate(ctx, json.RawMessage(tt.logic), tt.data)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_ComplexConditions(t *testing.T) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	tests := []struct {
		name     string
		logic    string
		data     map[string]interface{}
		expected bool
	}{
		{
			name: "purchase of $20+ at specific location",
			logic: `{
				"all": [
					{">=": [{"var": "amount"}, 20]},
					{"==": [{"var": "event_type"}, "purchase"]},
					{"in": [{"var": "location"}, ["store_1", "store_2"]]}
				]
			}`,
			data: map[string]interface{}{
				"amount":     25.0,
				"event_type": "purchase",
				"location":   "store_1",
			},
			expected: true,
		},
		{
			name: "nested conditions",
			logic: `{
				"any": [
					{
						"all": [
							{">=": [{"var": "amount"}, 50]},
							{"==": [{"var": "currency"}, "USD"]}
						]
					},
					{
						"all": [
							{">=": [{"var": "amount"}, 100]},
							{"==": [{"var": "currency"}, "ZWL"]}
						]
					}
				]
			}`,
			data: map[string]interface{}{
				"amount":   60.0,
				"currency": "USD",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Evaluate(ctx, json.RawMessage(tt.logic), tt.data)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_WithinDays(t *testing.T) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	lastWeek := now.AddDate(0, 0, -8)

	tests := []struct {
		name     string
		logic    string
		data     map[string]interface{}
		expected bool
	}{
		{
			name:  "within 7 days - true (yesterday)",
			logic: `{"within_days": [{"var": "occurred_at"}, 7]}`,
			data: map[string]interface{}{
				"occurred_at": yesterday,
			},
			expected: true,
		},
		{
			name:  "within 7 days - false (8 days ago)",
			logic: `{"within_days": [{"var": "occurred_at"}, 7]}`,
			data: map[string]interface{}{
				"occurred_at": lastWeek,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Evaluate(ctx, json.RawMessage(tt.logic), tt.data)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_EmptyLogic(t *testing.T) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	result, err := e.Evaluate(ctx, json.RawMessage(""), map[string]interface{}{})
	if err != nil {
		t.Errorf("Evaluate() error = %v", err)
		return
	}
	if result != true {
		t.Errorf("Evaluate() with empty logic should return true, got %v", result)
	}
}

func TestEvaluator_InvalidJSON(t *testing.T) {
	e := NewEvaluator(nil)
	ctx := context.Background()

	_, err := e.Evaluate(ctx, json.RawMessage("{invalid json}"), map[string]interface{}{})
	if err == nil {
		t.Error("Evaluate() should return error for invalid JSON")
	}
}

func BenchmarkEvaluator_SimpleComparison(b *testing.B) {
	e := NewEvaluator(nil)
	ctx := context.Background()
	logic := json.RawMessage(`{">=": [{"var": "amount"}, 20]}`)
	data := map[string]interface{}{"amount": 25.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Evaluate(ctx, logic, data)
	}
}

func BenchmarkEvaluator_ComplexCondition(b *testing.B) {
	e := NewEvaluator(nil)
	ctx := context.Background()
	logic := json.RawMessage(`{
		"all": [
			{">=": [{"var": "amount"}, 20]},
			{"==": [{"var": "event_type"}, "purchase"]},
			{"in": [{"var": "location"}, ["store_1", "store_2", "store_3"]]}
		]
	}`)
	data := map[string]interface{}{
		"amount":     25.0,
		"event_type": "purchase",
		"location":   "store_1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Evaluate(ctx, logic, data)
	}
}
