package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	suiclient "github.com/coming-chat/go-sui/v2/client"
	suitypes "github.com/coming-chat/go-sui/v2/types"

	"github.com/teleconsys/DCS/internal/config"
	"github.com/teleconsys/DCS/internal/rebased"
)

func main() {
	// Load the .env file
	config.LoadEnv()

	// Read the environment variables from the .env file
	userAddress := os.Getenv("USER_ADDRESS")
	userPrivateKey := os.Getenv("USER_PRIVATE_KEY")
	userGasCoinID := os.Getenv("USER_GAS_COIN_ID")

	// Verify that all the environment variables are present
	if userAddress == "" {
		fmt.Fprintf(os.Stderr, "Error: USER_ADDRESS not found in the .env file\n")
		os.Exit(1)
	}
	if userPrivateKey == "" {
		fmt.Fprintf(os.Stderr, "Error: USER_PRIVATE_KEY not found in the .env file\n")
		os.Exit(1)
	}
	if userGasCoinID == "" {
		fmt.Fprintf(os.Stderr, "Error: USER_GAS_COIN_ID not found in the .env file\n")
		os.Exit(1)
	}

	// Read the RPC URL (default: testnet Iota)
	rpcURL := os.Getenv("REBASE_RPC")
	if rpcURL == "" {
		rpcURL = "https://api.testnet.iota.cafe:443"
	}

	gasBudget := uint64(10_000_000)

	amount := int64(4000) // 4000 nanos = 0.000004 IOTA
	if s := os.Getenv("SPLIT_COIN_AMOUNT"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			amount = v
		}
	}

	// Print the information
	fmt.Println("=== Test Split Coin on the Testnet of Iota ===")
	fmt.Printf("User Address: %s\n", userAddress)
	fmt.Printf("Gas Coin ID: %s\n", userGasCoinID)
	fmt.Printf("RPC URL: %s\n", rpcURL)
	fmt.Printf("Gas Budget: %d\n", gasBudget)
	fmt.Printf("Amount to split: %d nanos\n", amount)
	fmt.Println()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to the RPC client
	client, err := suiclient.Dial(rpcURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to the RPC: %v\n", err)
		os.Exit(1)
	}

	// Call unsafe_splitCoin to build the transaction
	fmt.Println("Building transaction split coin...")
	var txb suitypes.TransactionBytes

	payIotaMethod := unsafeMethod("payIota")
	inputCoins := []string{userGasCoinID}
	recipients := []string{userAddress}
	amounts := []string{fmt.Sprintf("%d", amount)}
	gasBudgetStr := fmt.Sprintf("%d", gasBudget)

	err = client.CallContext(ctx, &txb, payIotaMethod,
		userAddress,  // signer
		inputCoins,   // input_coins
		recipients,   // recipients (pay to yourself)
		amounts,      // amounts
		gasBudgetStr, // gas_budget
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building the transaction: %v\n", err)
		os.Exit(1)
	}

	// Sign the transaction
	fmt.Println("Signing the transaction...")
	rawTx := []byte(txb.TxBytes)
	base64Tx := base64.StdEncoding.EncodeToString(rawTx)
	sigB64, err := rebased.SignTxBytes(ctx, rawTx, userPrivateKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error signing the transaction: %v\n", err)
		os.Exit(1)
	}

	// Execute the transaction
	fmt.Println("Executing the transaction...")
	opts := &suitypes.SuiTransactionBlockResponseOptions{
		ShowEffects:       true,
		ShowEvents:        true,
		ShowObjectChanges: true,
	}
	reqType := suitypes.ExecuteTransactionRequestType("WaitForLocalExecution")

	var rsp suitypes.SuiTransactionBlockResponse
	executeMethod := iotaMethod("executeTransactionBlock")
	err = client.CallContext(ctx, &rsp, executeMethod,
		base64Tx, []any{sigB64}, opts, reqType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing the transaction: %v\n", err)
		os.Exit(1)
	}

	// Extract the new Coin ID from the response
	newCoinID, err := extractNewCoinIdFromResponse(&rsp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting the new Coin ID: %v\n", err)
		os.Exit(1)
	}

	// Print the result
	fmt.Println("✅ Split coin successful!")
	fmt.Printf("New Coin ID: %s\n", newCoinID)
}

func extractNewCoinIdFromResponse(rsp *suitypes.SuiTransactionBlockResponse) (string, error) {
	if len(rsp.ObjectChanges) == 0 {
		return "", fmt.Errorf("no object changes found in transaction response")
	}

	// Search in the ObjectChanges array for objects "created" with type Coin
	for _, change := range rsp.ObjectChanges {
		// Marshal to access the fields
		changeJSON, err := json.Marshal(change)
		if err != nil {
			continue
		}

		var changeMap map[string]interface{}
		if err := json.Unmarshal(changeJSON, &changeMap); err != nil {
			continue
		}

		// Search for the "Data": {"created": {...}} structure
		if data, ok := changeMap["Data"].(map[string]interface{}); ok {
			if created, ok := data["created"].(map[string]interface{}); ok {
				// Verify that it is a Coin
				if objectType, ok := created["objectType"].(string); ok {
					if strings.Contains(objectType, "Coin") {
						if objectId, ok := created["objectId"].(string); ok {
							return objectId, nil
						}
					}
				}
			}
		}
	}

	// Fallback: search with regex in the marshaled JSON
	b, err := json.Marshal(rsp.ObjectChanges)
	if err != nil {
		return "", fmt.Errorf("failed to marshal object changes: %w", err)
	}

	// Pattern to search for "created" with objectId
	re := regexp.MustCompile(`"created"\s*:\s*\{[^}]*"objectType"\s*:\s*"[^"]*Coin[^"]*"[^}]*"objectId"\s*:\s*"(0x[a-fA-F0-9]{64})"`)
	matches := re.FindStringSubmatch(string(b))
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("no new coin ID found in transaction response")
}

// unsafeMethod implements the Method interface for the unsafe methods
type unsafeMethod string

func (u unsafeMethod) String() string {
	return "unsafe_" + string(u)
}

// iotaMethod implements the Method interface for the iota methods
type iotaMethod string

func (i iotaMethod) String() string {
	return "iota_" + string(i)
}
