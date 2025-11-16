package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Evaluator evaluates JsonLogic expressions
type Evaluator struct {
	customOps *CustomOperators
}

// NewEvaluator creates a new JsonLogic evaluator
func NewEvaluator(customOps *CustomOperators) *Evaluator {
	return &Evaluator{
		customOps: customOps,
	}
}

// Evaluate evaluates a JsonLogic expression against data
func (e *Evaluator) Evaluate(ctx context.Context, logic json.RawMessage, data map[string]interface{}) (bool, error) {
	if len(logic) == 0 {
		return true, nil // Empty logic always passes
	}

	var expr interface{}
	if err := json.Unmarshal(logic, &expr); err != nil {
		return false, fmt.Errorf("failed to parse logic: %w", err)
	}

	result, err := e.evaluate(ctx, expr, data)
	if err != nil {
		return false, err
	}

	// Convert result to boolean
	return toBool(result), nil
}

// evaluate recursively evaluates an expression
func (e *Evaluator) evaluate(ctx context.Context, expr interface{}, data map[string]interface{}) (interface{}, error) {
	// Handle literals (strings, numbers, booleans, nil)
	switch v := expr.(type) {
	case bool, float64, string, nil:
		return v, nil
	}

	// Handle objects (operators)
	exprMap, ok := expr.(map[string]interface{})
	if !ok {
		// If it's an array, evaluate each element
		if arr, ok := expr.([]interface{}); ok {
			results := make([]interface{}, len(arr))
			for i, item := range arr {
				res, err := e.evaluate(ctx, item, data)
				if err != nil {
					return nil, err
				}
				results[i] = res
			}
			return results, nil
		}
		return expr, nil
	}

	// Process the operator
	for op, args := range exprMap {
		return e.applyOperator(ctx, op, args, data)
	}

	return nil, fmt.Errorf("empty operator object")
}

// applyOperator applies an operator to its arguments
func (e *Evaluator) applyOperator(ctx context.Context, op string, args interface{}, data map[string]interface{}) (interface{}, error) {
	switch op {
	case "var":
		return e.opVar(args, data)
	case "==":
		return e.opEqual(ctx, args, data)
	case "!=":
		return e.opNotEqual(ctx, args, data)
	case ">":
		return e.opGreaterThan(ctx, args, data)
	case ">=":
		return e.opGreaterThanOrEqual(ctx, args, data)
	case "<":
		return e.opLessThan(ctx, args, data)
	case "<=":
		return e.opLessThanOrEqual(ctx, args, data)
	case "all":
		return e.opAll(ctx, args, data)
	case "any":
		return e.opAny(ctx, args, data)
	case "none":
		return e.opNone(ctx, args, data)
	case "in":
		return e.opIn(ctx, args, data)
	case "!":
		return e.opNot(ctx, args, data)
	case "and":
		return e.opAnd(ctx, args, data)
	case "or":
		return e.opOr(ctx, args, data)
	// Custom operators
	case "within_days":
		return e.opWithinDays(ctx, args, data)
	case "nth_event_in_period":
		return e.opNthEventInPeriod(ctx, args, data)
	case "distinct_visit_days":
		return e.opDistinctVisitDays(ctx, args, data)
	default:
		return nil, fmt.Errorf("unknown operator: %s", op)
	}
}

// opVar retrieves a variable from data
func (e *Evaluator) opVar(args interface{}, data map[string]interface{}) (interface{}, error) {
	// args can be a string (variable path) or array [path, default]
	var path string
	var defaultVal interface{}

	switch v := args.(type) {
	case string:
		path = v
	case []interface{}:
		if len(v) > 0 {
			path, _ = v[0].(string)
		}
		if len(v) > 1 {
			defaultVal = v[1]
		}
	default:
		return nil, fmt.Errorf("invalid var arguments")
	}

	// Navigate the data structure
	val := getPath(data, path)
	if val == nil {
		return defaultVal, nil
	}
	return val, nil
}

// opEqual implements ==
func (e *Evaluator) opEqual(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	operands, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	if len(operands) < 2 {
		return nil, fmt.Errorf("== requires 2 operands")
	}
	return compare(operands[0], operands[1]) == 0, nil
}

// opNotEqual implements !=
func (e *Evaluator) opNotEqual(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	operands, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	if len(operands) < 2 {
		return nil, fmt.Errorf("!= requires 2 operands")
	}
	return compare(operands[0], operands[1]) != 0, nil
}

// opGreaterThan implements >
func (e *Evaluator) opGreaterThan(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	operands, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	if len(operands) < 2 {
		return nil, fmt.Errorf("> requires 2 operands")
	}
	return compare(operands[0], operands[1]) > 0, nil
}

// opGreaterThanOrEqual implements >=
func (e *Evaluator) opGreaterThanOrEqual(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	operands, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	if len(operands) < 2 {
		return nil, fmt.Errorf(">= requires 2 operands")
	}
	return compare(operands[0], operands[1]) >= 0, nil
}

// opLessThan implements <
func (e *Evaluator) opLessThan(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	operands, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	if len(operands) < 2 {
		return nil, fmt.Errorf("< requires 2 operands")
	}
	return compare(operands[0], operands[1]) < 0, nil
}

// opLessThanOrEqual implements <=
func (e *Evaluator) opLessThanOrEqual(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	operands, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	if len(operands) < 2 {
		return nil, fmt.Errorf("<= requires 2 operands")
	}
	return compare(operands[0], operands[1]) <= 0, nil
}

// opAll implements "all" (logical AND)
func (e *Evaluator) opAll(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	conditions, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	for _, cond := range conditions {
		if !toBool(cond) {
			return false, nil
		}
	}
	return true, nil
}

// opAny implements "any" (logical OR)
func (e *Evaluator) opAny(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	conditions, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	for _, cond := range conditions {
		if toBool(cond) {
			return true, nil
		}
	}
	return false, nil
}

// opNone implements "none" (logical NOR)
func (e *Evaluator) opNone(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	conditions, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	for _, cond := range conditions {
		if toBool(cond) {
			return false, nil
		}
	}
	return true, nil
}

// opIn implements "in" (array membership)
func (e *Evaluator) opIn(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	operands, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	if len(operands) < 2 {
		return nil, fmt.Errorf("in requires 2 operands")
	}

	needle := operands[0]
	haystack := operands[1]

	// Check if haystack is an array
	haystackArr, ok := toArray(haystack)
	if !ok {
		return false, nil
	}

	for _, item := range haystackArr {
		if compare(needle, item) == 0 {
			return true, nil
		}
	}
	return false, nil
}

// opNot implements "!" (logical NOT)
func (e *Evaluator) opNot(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	operands, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	if len(operands) < 1 {
		return nil, fmt.Errorf("! requires 1 operand")
	}
	return !toBool(operands[0]), nil
}

// opAnd implements "and" (logical AND)
func (e *Evaluator) opAnd(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	return e.opAll(ctx, args, data)
}

// opOr implements "or" (logical OR)
func (e *Evaluator) opOr(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	return e.opAny(ctx, args, data)
}

// evaluateArgs evaluates an array of arguments
func (e *Evaluator) evaluateArgs(ctx context.Context, args interface{}, data map[string]interface{}) ([]interface{}, error) {
	argsArr, ok := args.([]interface{})
	if !ok {
		// Single argument
		val, err := e.evaluate(ctx, args, data)
		if err != nil {
			return nil, err
		}
		return []interface{}{val}, nil
	}

	results := make([]interface{}, len(argsArr))
	for i, arg := range argsArr {
		val, err := e.evaluate(ctx, arg, data)
		if err != nil {
			return nil, err
		}
		results[i] = val
	}
	return results, nil
}

// Helper functions

// getPath retrieves a value from a nested map using dot notation
func getPath(data map[string]interface{}, path string) interface{} {
	if path == "" {
		return data
	}

	// Simple implementation - no dot notation for now
	val, ok := data[path]
	if !ok {
		// Try to get nested properties
		if props, ok := data["properties"].(map[string]interface{}); ok {
			if v, ok := props[path]; ok {
				return v
			}
		}
		return nil
	}
	return val
}

// compare compares two values, returning -1, 0, or 1
func compare(a, b interface{}) int {
	// Handle nil cases
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Convert to comparable types
	aNum, aIsNum := toNumber(a)
	bNum, bIsNum := toNumber(b)

	if aIsNum && bIsNum {
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
		return 0
	}

	// String comparison
	aStr := toString(a)
	bStr := toString(b)
	if aStr < bStr {
		return -1
	}
	if aStr > bStr {
		return 1
	}
	return 0
}

// toNumber converts a value to float64
func toNumber(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		var f float64
		_, err := fmt.Sscanf(val, "%f", &f)
		return f, err == nil
	default:
		return 0, false
	}
}

// toString converts a value to string
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// toBool converts a value to boolean
func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case int:
		return val != 0
	case string:
		return val != ""
	case []interface{}:
		return len(val) > 0
	case map[string]interface{}:
		return len(val) > 0
	default:
		return false
	}
}

// toArray converts a value to array
func toArray(v interface{}) ([]interface{}, bool) {
	arr, ok := v.([]interface{})
	return arr, ok
}

// Custom operator implementations

// opWithinDays checks if occurred_at is within N days
func (e *Evaluator) opWithinDays(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	operands, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	if len(operands) < 2 {
		return nil, fmt.Errorf("within_days requires 2 operands")
	}

	// First arg is the date (occurred_at)
	// Second arg is the number of days
	days, ok := toNumber(operands[1])
	if !ok {
		return nil, fmt.Errorf("within_days: days must be a number")
	}

	// Get occurred_at from data
	occurredAt, ok := data["occurred_at"]
	if !ok {
		return false, nil
	}

	var t time.Time
	switch v := occurredAt.(type) {
	case time.Time:
		t = v
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return false, fmt.Errorf("within_days: invalid date format")
		}
		t = parsed
	default:
		return false, fmt.Errorf("within_days: occurred_at must be a time or string")
	}

	cutoff := time.Now().AddDate(0, 0, -int(days))
	return t.After(cutoff), nil
}

// opNthEventInPeriod checks if this is the Nth event in a period
func (e *Evaluator) opNthEventInPeriod(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	if e.customOps == nil {
		return nil, fmt.Errorf("nth_event_in_period requires custom operators")
	}

	operands, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	if len(operands) < 3 {
		return nil, fmt.Errorf("nth_event_in_period requires 3 operands: event_type, n, period_days")
	}

	eventType := toString(operands[0])
	n, ok := toNumber(operands[1])
	if !ok {
		return nil, fmt.Errorf("nth_event_in_period: n must be a number")
	}
	periodDays, ok := toNumber(operands[2])
	if !ok {
		return nil, fmt.Errorf("nth_event_in_period: period_days must be a number")
	}

	// Get tenant_id and customer_id from data
	tenantID, _ := data["tenant_id"].(string)
	customerID, _ := data["customer_id"].(string)

	return e.customOps.NthEventInPeriod(ctx, tenantID, customerID, eventType, int(n), int(periodDays))
}

// opDistinctVisitDays counts distinct visit days
func (e *Evaluator) opDistinctVisitDays(ctx context.Context, args interface{}, data map[string]interface{}) (interface{}, error) {
	if e.customOps == nil {
		return nil, fmt.Errorf("distinct_visit_days requires custom operators")
	}

	operands, err := e.evaluateArgs(ctx, args, data)
	if err != nil {
		return nil, err
	}
	if len(operands) < 1 {
		return nil, fmt.Errorf("distinct_visit_days requires 1 operand: period_days")
	}

	periodDays, ok := toNumber(operands[0])
	if !ok {
		return nil, fmt.Errorf("distinct_visit_days: period_days must be a number")
	}

	// Get tenant_id and customer_id from data
	tenantID, _ := data["tenant_id"].(string)
	customerID, _ := data["customer_id"].(string)

	count, err := e.customOps.DistinctVisitDays(ctx, tenantID, customerID, int(periodDays))
	if err != nil {
		return nil, err
	}
	return float64(count), nil
}
