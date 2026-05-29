package offers

import (
	"encoding/json"
	"strconv"
)

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if m, ok := v.(map[string]any); ok {
		// sometimes encoded as {"bytes":"0x..."} or {"id":"0x..."}
		if s, _ := m["bytes"].(string); s != "" {
			return s
		}
		if s, _ := m["id"].(string); s != "" {
			return s
		}
	}
	return ""
}

func asI64(v any) int64 {
	switch t := v.(type) {
	case string:
		u, _ := strconv.ParseUint(t, 10, 64)
		return int64(u)
	case float64:
		return int64(t)
	default:
		return 0
	}
}

func decodeBytesString(v any) string {
	arr, ok := v.([]any)
	if !ok {
		b, _ := json.Marshal(v)
		return string(b)
	}
	buf := make([]byte, 0, len(arr))
	for _, x := range arr {
		switch n := x.(type) {
		case float64:
			buf = append(buf, byte(n))
		case int:
			buf = append(buf, byte(n))
		}
	}
	return string(buf)
}

func lenArray(v any) int {
	if a, ok := v.([]any); ok {
		return len(a)
	}
	return 0
}

func max0(x int64) int64 {
	if x < 0 {
		return 0
	}
	return x
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case float64:
		return int64(t)
	case string:
		i, _ := strconv.ParseInt(t, 10, 64)
		return i
	default:
		return 0
	}
}

func asSlice(v any) []any {
	if v == nil {
		return nil
	}
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}