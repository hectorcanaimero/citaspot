package engine

import (
	"fmt"
	"strings"

	"github.com/citaspot/api/internal/domain"
)

func EvaluateConditions(conditions []domain.RuleCondition, ctx map[string]any) bool {
	if len(conditions) == 0 {
		return true
	}
	for _, cond := range conditions {
		if !evaluateCondition(cond, ctx) {
			return false
		}
	}
	return true
}

func evaluateCondition(cond domain.RuleCondition, ctx map[string]any) bool {
	actual, ok := resolveField(cond.Field, ctx)
	if !ok {
		return false
	}
	switch cond.Op {
	case "eq":
		return compareEq(actual, cond.Value)
	case "neq":
		return !compareEq(actual, cond.Value)
	case "gt":
		return compareNumeric(actual, cond.Value) > 0
	case "lt":
		return compareNumeric(actual, cond.Value) < 0
	case "gte":
		return compareNumeric(actual, cond.Value) >= 0
	case "lte":
		return compareNumeric(actual, cond.Value) <= 0
	case "contains":
		return compareContains(actual, cond.Value)
	case "in":
		return compareIn(actual, cond.Value)
	default:
		return false
	}
}

func resolveField(field string, ctx map[string]any) (any, bool) {
	parts := strings.Split(field, ".")
	var current any = ctx
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func compareEq(actual, expected any) bool {
	a := normalizeValue(actual)
	e := normalizeValue(expected)
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", e)
}

func compareNumeric(actual, expected any) int {
	a := toFloat64(actual)
	b := toFloat64(expected)
	if a == nil || b == nil {
		return -2
	}
	switch {
	case *a < *b:
		return -1
	case *a > *b:
		return 1
	default:
		return 0
	}
}

func compareContains(actual, expected any) bool {
	aStr := fmt.Sprintf("%v", actual)
	eStr := fmt.Sprintf("%v", expected)
	return strings.Contains(strings.ToLower(aStr), strings.ToLower(eStr))
}

func compareIn(actual, expected any) bool {
	list, ok := expected.([]any)
	if !ok {
		strList, ok := expected.([]string)
		if !ok {
			return false
		}
		for _, item := range strList {
			if compareEq(actual, item) {
				return true
			}
		}
		return false
	}
	for _, item := range list {
		if compareEq(actual, item) {
			return true
		}
	}
	return false
}

func normalizeValue(v any) any {
	switch val := v.(type) {
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case bool:
		return val
	default:
		return v
	}
}

func toFloat64(v any) *float64 {
	var f float64
	switch val := v.(type) {
	case float64:
		f = val
	case float32:
		f = float64(val)
	case int:
		f = float64(val)
	case int32:
		f = float64(val)
	case int64:
		f = float64(val)
	case string:
		n, err := fmt.Sscanf(val, "%f", &f)
		if err != nil || n != 1 {
			return nil
		}
	default:
		return nil
	}
	return &f
}
