package cid

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/spf13/viper"

	"github.com/teleconsys/DCS/internal/rebased"
)

// GetCIDList fetches the CIDlist and returns only the list of CID object IDs using RPC wrapper
func GetCIDList(ctx context.Context, rpcURL string) ([]string, error) {
	// Get CID list ID
	cidListID, err := GetCIDListID()
	if err != nil {
		return nil, err
	}

	// Dial RPC
	w, err := rebased.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial failed: %w", err)
	}

	// Get the CIDlist object from the blockchain
	obj, err := w.GetObject(ctx, cidListID, suitypes.SuiObjectDataOptions{ShowContent: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get CIDlist object: %w", err)
	}
	if obj == nil || obj.Data == nil {
		return nil, fmt.Errorf("CIDlist object not found")
	}
	if obj.Error != nil {
		return nil, fmt.Errorf("rpc getObject error: %+v", obj.Error)
	}
	if obj.Data.Content == nil {
		return nil, fmt.Errorf("CIDlist object has no content")
	}

	// Parse the CIDlist JSON
	cb, err := json.Marshal(obj.Data.Content)
	if err != nil {
		return nil, fmt.Errorf("marshal content: %w", err)
	}

	fields, _, ok := extractFieldsFromContent(cb)
	if !ok {
		return nil, fmt.Errorf("could not extract fields from CIDlist content")
	}

	rawCIDList, ok := fields["cidlist"]
	if !ok {
		return nil, fmt.Errorf("cidlist field not found in CIDlist object")
	}

	// Convert to string slice
	cidList, ok := rawCIDList.([]interface{})
	if !ok {
		return nil, fmt.Errorf("cidlist field is not an array")
	}

	result := make([]string, len(cidList))
	for i, item := range cidList {
		if str, ok := item.(string); ok {
			result[i] = str
		} else {
			return nil, fmt.Errorf("cidlist item %d is not a string", i)
		}
	}

	return result, nil
}

// GetCIDFields fetches a CID object and returns its fields map using RPC wrapper
func GetCIDFields(ctx context.Context, w *rebased.Wrapper, cidObjectID string) (map[string]any, error) {
	// Get the CID object from the blockchain
	obj, err := w.GetObject(ctx, cidObjectID, suitypes.SuiObjectDataOptions{ShowContent: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get CID object %s: %w", cidObjectID, err)
	}
	if obj == nil || obj.Data == nil {
		return nil, fmt.Errorf("CID object %s not found", cidObjectID)
	}
	if obj.Error != nil {
		return nil, fmt.Errorf("rpc getObject error: %+v", obj.Error)
	}
	if obj.Data.Content == nil {
		return nil, fmt.Errorf("CID object %s has no content", cidObjectID)
	}

	// Parse the CID object JSON
	cb, err := json.Marshal(obj.Data.Content)
	if err != nil {
		return nil, fmt.Errorf("marshal content: %w", err)
	}

	fields, _, ok := extractFieldsFromContent(cb)
	if !ok {
		return nil, fmt.Errorf("could not extract fields from CID object content")
	}

	return fields, nil
}

// GetCidStringFromObject fetches a CID object and returns only the cid_str using RPC wrapper
func GetCIDStringFromObject(ctx context.Context, cidObjectID, rpcURL string) (string, error) {
	// Dial RPC
	w, err := rebased.Dial(rpcURL)
	if err != nil {
		return "", fmt.Errorf("rpc dial failed: %w", err)
	}

	fields, err := GetCIDFields(ctx, w, cidObjectID)
	if err != nil {
		return "", err
	}

	cidStr, ok := fields["cid_str"]
	if !ok {
		return "", fmt.Errorf("cid_str field not found in CID object")
	}

	return convertToString(cidStr)
}

// GetCIDIdFromList searches for a particular CID (not its ID) in CIDlist and returns the object ID
func GetCIDIdFromList(ctx context.Context, cidStr, rpcURL string) (string, error) {
	// Get the CID list

	cidList, err := GetCIDList(ctx, rpcURL)
	if err != nil {
		return "", err
	}
	
	// Search through each CID object ID in the list
	for _, cidObjectID := range cidList {
		// Get the CID string from the CID object
		storedCidStr, err := GetCIDStringFromObject(ctx, cidObjectID, rpcURL)
		if err != nil {
			continue // Skip if we can't fetch this CID object
		}

		// Check if the cid_str matches our target
		if storedCidStr == cidStr {
			return cidObjectID, nil
		}
	}

	return "", fmt.Errorf("CID %s not found in CIDlist", cidStr)
}

// GetCIDListID retrieves the CID list ID from config or environment
func GetCIDListID() (string, error) {
	cidListID := viper.GetString("dcs.cidlist_id")
	if cidListID == "" {
		cidListID = os.Getenv("DCS_CIDLIST_ID")
	}
	if cidListID == "" {
		return "", fmt.Errorf("set DCS_CIDLIST_ID env var or pass --cidlist-id (0x...)")
	}
	return cidListID, nil
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

// convertToString converts a value that may be a string or byte array (vector<u8>) to a string.
// It handles string, []byte, and []interface{} (JSON array of numbers) types.
func convertToString(value interface{}) (string, error) {
	// Handle string case
	if str, ok := value.(string); ok {
		return str, nil
	}

	// Handle vector<u8> case (byte array from JSON)
	if byteSlice, ok := value.([]interface{}); ok {
		bytes := make([]byte, len(byteSlice))
		for i, v := range byteSlice {
			// JSON numbers are typically float64, but we need uint8
			switch val := v.(type) {
			case float64:
				bytes[i] = byte(val)
			case uint8:
				bytes[i] = val
			case int:
				bytes[i] = byte(val)
			default:
				return "", fmt.Errorf("byte array contains non-numeric value at index %d: %T", i, v)
			}
		}
		return string(bytes), nil
	}

	// Handle []byte case directly
	if bytes, ok := value.([]byte); ok {
		return string(bytes), nil
	}

	return "", fmt.Errorf("value is neither a string nor a byte array (type: %T)", value)
}
