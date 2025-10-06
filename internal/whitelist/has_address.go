package whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/teleconsys/DCS/internal/rebased"
)

type HasParams struct {
	WhitelistID string
	Member      string
	RPCURL      string // JSON-RPC endpoint (e.g., https://api.testnet.iota.cafe:443)
}

// LoadHasAddressParams reads flags/env/positional args and resolves the RPC URL.
func LoadHasAddressParams(cmd *cobra.Command, args []string) (HasParams, error) {
	var p HasParams

	// whitelist id
	p.WhitelistID = viper.GetString("dcs.whitelist_id")
	if p.WhitelistID == "" {
		p.WhitelistID = os.Getenv("DCS_WHITELIST_ID")
	}
	if p.WhitelistID == "" {
		return p, fmt.Errorf("set DCS_WHITELIST_ID env var or pass --id (0x...)")
	}

	// member: flag takes priority, otherwise positional
	if mFlag, _ := cmd.Flags().GetString("member"); mFlag != "" {
		p.Member = mFlag
	} else if len(args) > 0 {
		p.Member = args[0]
	}
	if p.Member == "" {
		return p, fmt.Errorf("provide address as positional arg or --member (0x...)")
	}
	p.Member = strings.ToLower(p.Member)

	// RPC URL: prefer root flag (--rpc via viper), else envs, else testnet default
	p.RPCURL = viper.GetString("rpc")
	if p.RPCURL == "" {
		if u := os.Getenv("REBASE_RPC"); u != "" {
			p.RPCURL = u
		} else if u := os.Getenv("DCS_RPC"); u != "" {
			p.RPCURL = u
		} else {
			p.RPCURL = "https://api.testnet.iota.cafe:443"
		}
	}

	return p, nil
}

// HasAddress checks presence using JSON-RPC only (no iota CLI).
func HasAddress(ctx context.Context, p HasParams) (bool, error) {
	// 1) Dial RPC wrapper
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return false, fmt.Errorf("rpc dial failed: %w", err)
	}

	// 2) Fetch object with content
	obj, err := w.GetObject(ctx, p.WhitelistID, suitypes.SuiObjectDataOptions{ShowContent: true})
	if err != nil {
		return false, err
	}
	if obj == nil || obj.Data == nil {
		return false, nil
	}
	if obj.Error != nil {
		return false, fmt.Errorf("rpc getObject error: %+v", obj.Error)
	}
	if obj.Data.Content == nil {
		return false, nil
	}

	// 3) Extract fields from Content and check presence
	cb, err := json.Marshal(obj.Data.Content)
	if err != nil {
		return false, fmt.Errorf("marshal content: %w", err)
	}
	fields, _, ok := extractFieldsFromContent(cb)
	if !ok {
		return false, nil
	}
	rawWL, ok := fields["whitelist"]
	if !ok {
		return false, nil
	}
	return whitelistContainsAddress(rawWL, p.Member), nil
}

// extractFieldsFromContent tries common Content shapes and reports which matched.
// It returns (fields, path, true) if found; otherwise (nil, "", false).
func extractFieldsFromContent(jsonBytes []byte) (map[string]any, string, bool) {
	var root map[string]any
	if err := json.Unmarshal(jsonBytes, &root); err != nil {
		return nil, "", false
	}

	// A) Content = { "Data": { "moveObject": { "fields": {...} } } }
	if data, ok := root["Data"].(map[string]any); ok {
		if mo, ok := data["moveObject"].(map[string]any); ok {
			if fields, ok := mo["fields"].(map[string]any); ok {
				return fields, "Content.Data.moveObject.fields", true
			}
		}
	}

	// B) Content = { "data": { "moveObject": { "fields": {...} } } }
	if data, ok := root["data"].(map[string]any); ok {
		if mo, ok := data["moveObject"].(map[string]any); ok {
			if fields, ok := mo["fields"].(map[string]any); ok {
				return fields, "Content.data.moveObject.fields", true
			}
		}
	}

	// C) Content = { "moveObject": { "fields": {...} } }
	if mo, ok := root["moveObject"].(map[string]any); ok {
		if fields, ok := mo["fields"].(map[string]any); ok {
			return fields, "Content.moveObject.fields", true
		}
	}

	// D) Content = { "fields": {...} }
	if fields, ok := root["fields"].(map[string]any); ok {
		return fields, "Content.fields", true
	}

	// E) Less common: { "bcs": { "fields": {...} } }
	if bcs, ok := root["bcs"].(map[string]any); ok {
		if fields, ok := bcs["fields"].(map[string]any); ok {
			return fields, "Content.bcs.fields", true
		}
	}

	return nil, "", false
}
