package whitelist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// keepJSON drops anything before the first '{' or '['.
// Some CLIs print banners or warnings before the JSON.
func keepJSON(b []byte) []byte {
	if i := bytes.IndexAny(b, "{["); i >= 0 {
		return b[i:]
	}
	return nil
}

// Extract fields map from either shape:
// 1) root.content.fields (current CLI)
// 2) root.data.content.fields (legacy)
func extractObjectFields(jsonBytes []byte) (map[string]any, error) {
	var root map[string]any
	if err := json.Unmarshal(jsonBytes, &root); err != nil {
		return nil, fmt.Errorf("decode object json: %w", err)
	}
	if content, ok := root["content"].(map[string]any); ok {
		if fields, ok := content["fields"].(map[string]any); ok {
			return fields, nil
		}
	}
	if data, ok := root["data"].(map[string]any); ok {
		if content, ok := data["content"].(map[string]any); ok {
			if fields, ok := content["fields"].(map[string]any); ok {
				return fields, nil
			}
		}
	}
	return nil, fmt.Errorf("fields not found in object JSON")
}

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
