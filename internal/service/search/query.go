package search

import (
	"encoding/json"
	"fmt"
	"time"
)

// SearchQuery is the top-level structured query.
type SearchQuery struct {
	Logic      string            `json:"logic"`       // "and" / "or"
	Conditions []SearchCondition `json:"conditions"`
}

// SearchCondition is either a leaf condition or a nested group.
// If Conditions is non-nil, it's a group. Otherwise it's a leaf.
type SearchCondition struct {
	// Leaf fields
	Type  string `json:"type,omitempty"`  // filename, tag, tag_value, size, ext, date
	Op    string `json:"op,omitempty"`    // contains, has, gt, etc.
	Value any    `json:"value,omitempty"` // string, float64, []string
	Tag   string `json:"tag,omitempty"`   // tag_value type only

	// Group fields
	Logic      string            `json:"logic,omitempty"`
	Conditions []SearchCondition `json:"conditions,omitempty"`
}

func (c *SearchCondition) IsGroup() bool {
	return len(c.Conditions) > 0
}

// validateQuery checks a SearchQuery for structural validity.
func validateQuery(q *SearchQuery) error {
	if q.Logic != "and" && q.Logic != "or" {
		return fmt.Errorf("logic 必须是 and 或 or")
	}
	if len(q.Conditions) == 0 {
		return fmt.Errorf("至少需要一个条件")
	}
	return validateConditions(q.Conditions)
}

func validateConditions(conditions []SearchCondition) error {
	validTypes := map[string]bool{
		"filename": true, "tag": true, "tag_value": true,
		"size": true, "ext": true, "date": true,
	}
	validOps := map[string]bool{
		"contains": true, "not_contains": true, "equals": true,
		"has": true, "not_has": true,
		"gt": true, "gte": true, "lt": true, "lte": true,
		"in": true, "not_in": true,
	}

	for i, c := range conditions {
		if c.IsGroup() {
			if c.Logic != "and" && c.Logic != "or" {
				return fmt.Errorf("条件组[%d]: logic 必须是 and 或 or", i)
			}
			if len(c.Conditions) == 0 {
				return fmt.Errorf("条件组[%d]: 不能为空", i)
			}
			if err := validateConditions(c.Conditions); err != nil {
				return err
			}
			continue
		}
		if !validTypes[c.Type] {
			return fmt.Errorf("条件[%d]: 无效的 type '%s'", i, c.Type)
		}
		if !validOps[c.Op] {
			return fmt.Errorf("条件[%d]: 无效的 op '%s'", i, c.Op)
		}
		if c.Value == nil {
			return fmt.Errorf("条件[%d]: value 不能为空", i)
		}
		if c.Type == "tag_value" && c.Tag == "" {
			return fmt.Errorf("条件[%d]: tag_value 类型必须指定 tag 字段", i)
		}
	}
	return nil
}

// toInt64 converts a JSON number (float64) to int64 for size comparisons.
func toInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}

// toStringSlice converts a []any (from JSON array) to []string.
func toStringSlice(v any) []string {
	switch arr := v.(type) {
	case []any:
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return arr
	default:
		return nil
	}
}

// parseDate parses an ISO date string to time.Time.
func parseDate(s string) (time.Time, error) {
	layouts := []string{"2006-01-02", "2006-01-02T15:04:05", time.RFC3339}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无效的日期格式: %s", s)
}
