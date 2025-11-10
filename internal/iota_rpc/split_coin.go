package iota_rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SplitCoinParams contains all parameters for the unsafe_splitCoin RPC method
type SplitCoinParams struct {
	Signer        string   // Transaction signer's IOTA address (required)
	CoinObjectID  string   // Coin object to split (required)
	SplitAmounts  []string // Amounts to split out (required) - as strings to match BigInt format
	Gas           *string  // Gas object ID (optional, nil means omit)
	GasBudget     string   // Gas budget (required) - as string to match BigInt format
}

// SplitCoinResponse represents the JSON-RPC response structure
type splitCoinResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		TxBytes      string `json:"txBytes"`
		Gas          []any  `json:"gas"`
		InputObjects []any  `json:"inputObjects"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data"`
	} `json:"error,omitempty"`
}

// UnsafeSplitCoin calls the unsafe_splitCoin RPC method and returns the txBytes field
// endpoint: RPC endpoint URL (e.g., "https://api.testnet.iota.cafe:443")
// params: Parameters for the splitCoin operation
func UnsafeSplitCoin(ctx context.Context, endpoint string, params SplitCoinParams) (string, error) {
	// Build the JSON-RPC request parameters
	rpcParams := make([]any, 0, 5)
	rpcParams = append(rpcParams, params.Signer)
	rpcParams = append(rpcParams, params.CoinObjectID)
	rpcParams = append(rpcParams, params.SplitAmounts)
	
	// Handle optional gas parameter - use nil if not provided
	if params.Gas != nil {
		rpcParams = append(rpcParams, *params.Gas)
	} else {
		rpcParams = append(rpcParams, nil)
	}
	
	rpcParams = append(rpcParams, params.GasBudget)

	// Build the JSON-RPC request
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "unsafe_splitCoin",
		"params":  rpcParams,
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var rpcResp splitCoinResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	// Check for RPC error
	if rpcResp.Error != nil {
		return "", fmt.Errorf("RPC error [%d]: %s (data: %v)",
			rpcResp.Error.Code, rpcResp.Error.Message, rpcResp.Error.Data)
	}

	// Return txBytes
	if rpcResp.Result.TxBytes == "" {
		return "", fmt.Errorf("txBytes field is empty in response")
	}

	return rpcResp.Result.TxBytes, nil
}


