package whitelist

import (
	"strings"
)

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
