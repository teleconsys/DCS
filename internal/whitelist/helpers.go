package whitelist

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// GroundControlInfo holds the ground control information from the API
type GroundControlInfo struct {
	Address     string
	PrivateKey  string
	WalletGasID string
}

// // keepJSON drops anything before the first '{' or '['.
// // Some CLIs print banners or warnings before the JSON.
// func keepJSON(b []byte) []byte {
// 	if i := bytes.IndexAny(b, "{["); i >= 0 {
// 		return b[i:]
// 	}
// 	return nil
// }

// // Extract fields map from either shape:
// // 1) root.content.fields (current CLI)
// // 2) root.data.content.fields (legacy)
// func extractObjectFields(jsonBytes []byte) (map[string]any, error) {
// 	var root map[string]any
// 	if err := json.Unmarshal(jsonBytes, &root); err != nil {
// 		return nil, fmt.Errorf("decode object json: %w", err)
// 	}
// 	if content, ok := root["content"].(map[string]any); ok {
// 		if fields, ok := content["fields"].(map[string]any); ok {
// 			return fields, nil
// 		}
// 	}
// 	if data, ok := root["data"].(map[string]any); ok {
// 		if content, ok := data["content"].(map[string]any); ok {
// 			if fields, ok := content["fields"].(map[string]any); ok {
// 				return fields, nil
// 			}
// 		}
// 	}
// 	return nil, fmt.Errorf("fields not found in object JSON")
// }

// Check presence of address in fields.whitelist
func whitelistContainsAddress(raw any, address string) bool {
	switch arr := raw.(type) {
	case []any:
		for _, item := range arr {
			switch t := item.(type) {
			case string:
				if strings.EqualFold(t, address) {
					return true
				}
			case map[string]any:
				if s, _ := t["id"].(string); s != "" && strings.EqualFold(s, address) {
					return true
				}
				if s, _ := t["bytes"].(string); s != "" && strings.EqualFold(s, address) {
					return true
				}
				// Some whitelists may wrap the address deeper: {"fields":{"id":"0x.."}}
				if f, ok := t["fields"].(map[string]any); ok {
					if s, _ := f["id"].(string); s != "" && strings.EqualFold(s, address) {
						return true
					}
					if s, _ := f["bytes"].(string); s != "" && strings.EqualFold(s, address) {
						return true
					}
				}
			}
		}
	}
	return false
}

// GetGroundControlInfo fetches ground control information from the API server
// configured via GC_ENDPOINT and GC_ENDPOINT_PORT environment variables
func GetGroundControlInfo() (*GroundControlInfo, error) {
	endpoint := os.Getenv("GC_ENDPOINT")
	if endpoint == "" {
		return nil, fmt.Errorf("GC_ENDPOINT environment variable is not set")
	}

	url := fmt.Sprintf("http://%s/get-groundcontrol-info", endpoint)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ground control info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var result struct {
		Address     string `json:"address"`
		PrivateKey  string `json:"privateKey"`
		WalletGasID string `json:"walletGasId"`
		Error       string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("API error: %s", result.Error)
	}

	return &GroundControlInfo{
		Address:     result.Address,
		PrivateKey:  result.PrivateKey,
		WalletGasID: result.WalletGasID,
	}, nil
}
