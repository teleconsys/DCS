package offers

import (
	"encoding/json"

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
