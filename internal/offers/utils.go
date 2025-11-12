package offers

import (
	"encoding/json"
	"strconv"

	suitypes "github.com/coming-chat/go-sui/v2/types"
)

// txStatusOK inspects effects status across SDK JSON shapes.
// Returns (ok, reason). ok==true when status == "success".
func txStatusOK(resp *suitypes.SuiTransactionBlockResponse) (bool, string) {
	if resp == nil || resp.Effects == nil {
		return false, "no effects in response"
	}
	b, _ := json.Marshal(resp.Effects)

	// Shape A: effects.Data.v1.status
	var a struct {
		Data struct {
			V1 struct {
				Status struct {
					Status string `json:"status"`
					Error  string `json:"error"`
				} `json:"status"`
			} `json:"v1"`
		} `json:"Data"`
	}
	if json.Unmarshal(b, &a) == nil && a.Data.V1.Status.Status != "" {
		return a.Data.V1.Status.Status == "success", a.Data.V1.Status.Error
	}

	// Shape B: effects.data.status
	var bshape struct {
		Data struct {
			Status struct {
				Status string `json:"status"`
				Error  string `json:"error"`
			} `json:"status"`
		} `json:"data"`
	}
	if json.Unmarshal(b, &bshape) == nil && bshape.Data.Status.Status != "" {
		return bshape.Data.Status.Status == "success", bshape.Data.Status.Error
	}

	// Shape C: effects.status
	var c struct {
		Status struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"status"`
	}
	if json.Unmarshal(b, &c) == nil && c.Status.Status != "" {
		return c.Status.Status == "success", c.Status.Error
	}

	// Unknown shape: assume success; caller should still verify state if needed.
	return true, ""
}

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
