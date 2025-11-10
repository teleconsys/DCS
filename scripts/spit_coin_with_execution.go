package main

import (
	"context"
	"fmt"

	"github.com/coming-chat/go-sui/v2/account"
	"github.com/coming-chat/go-sui/v2/client"
	"github.com/coming-chat/go-sui/v2/sui_types"
	"github.com/coming-chat/go-sui/v2/types"
)

// SplitCoinConfig holds the configuration for splitting a coin
type SplitCoinConfig struct {
	RPCEndpoint  string   // SUI RPC endpoint (e.g., "https://fullnode.testnet.sui.io:443")
	Account      *account.Account // SUI account to use for signing
	CoinObjectID string   // Object ID of the coin to split
	Amounts      []uint64 // Amounts to split into (in MIST)
	GasObjectID  *string  // Optional: separate gas coin object ID (nil to use same coin)
	GasBudget    uint64   // Gas budget in MIST (e.g., 10000000 = 0.01 SUI)
}

// SplitCoinResult holds the result of a split coin operation
type SplitCoinResult struct {
	TransactionDigest string
	Success           bool
	CreatedCoins      []string
	BalanceChanges    []types.BalanceChange
	Error             error
}

// SplitCoin splits a SUI coin into multiple smaller coins
func SplitCoin(ctx context.Context, config SplitCoinConfig) *SplitCoinResult {
	result := &SplitCoinResult{}

	// 1. Connect to SUI network
	suiClient, err := client.Dial(config.RPCEndpoint)
	if err != nil {
		result.Error = fmt.Errorf("failed to connect to SUI: %w", err)
		return result
	}

	// 2. Convert sender address
	addrPtr, err := sui_types.NewAddressFromHex(config.Account.Address)
	if err != nil {
		result.Error = fmt.Errorf("invalid sender address: %w", err)
		return result
	}
	senderAddress := *addrPtr

	// 3. Convert coin object ID
	coinObjIDPtr, err := sui_types.NewObjectIdFromHex(config.CoinObjectID)
	if err != nil {
		result.Error = fmt.Errorf("invalid coin object ID: %w", err)
		return result
	}
	coinObjID := *coinObjIDPtr

	// 4. Prepare gas object ID
	var gasObjID *sui_types.ObjectID
	if config.GasObjectID != nil {
		gasObjIDPtr, err := sui_types.NewObjectIdFromHex(*config.GasObjectID)
		if err != nil {
			result.Error = fmt.Errorf("invalid gas object ID: %w", err)
			return result
		}
		gasObjID = gasObjIDPtr
	}

	// 5. Convert amounts to SafeSuiBigInt
	splitAmounts := make([]types.SafeSuiBigInt[uint64], 0, len(config.Amounts))
	for _, a := range config.Amounts {
		splitAmounts = append(splitAmounts, types.NewSafeSuiBigInt(a))
	}

	// 6. Build the split coin transaction
	txBytes, err := suiClient.SplitCoin(
		ctx,
		senderAddress,
		coinObjID,
		splitAmounts,
		gasObjID,
		types.NewSafeSuiBigInt(config.GasBudget),
	)
	if err != nil {
		result.Error = fmt.Errorf("failed to build transaction: %w", err)
		return result
	}

	// 7. Sign the transaction
	rawTxBytes := []byte(txBytes.TxBytes)
	signature, err := config.Account.SignSecureWithoutEncode(rawTxBytes, sui_types.DefaultIntent())
	if err != nil {
		result.Error = fmt.Errorf("failed to sign transaction: %w", err)
		return result
	}

	// 8. Execute the transaction
	options := types.SuiTransactionBlockResponseOptions{
		ShowInput:          true,
		ShowEffects:        true,
		ShowEvents:         true,
		ShowObjectChanges:  true,
		ShowBalanceChanges: true,
	}

	resp, err := suiClient.ExecuteTransactionBlock(
		ctx,
		txBytes.TxBytes,
		[]any{signature},
		&options,
		types.TxnRequestTypeWaitForLocalExecution,
	)
	if err != nil {
		result.Error = fmt.Errorf("failed to execute transaction: %w", err)
		return result
	}

	fmt.Println(resp)

	// 9. Parse results
	// result.TransactionDigest = resp.Digest.String()
	// result.Success = resp.Effects.Status.Status == "success"
	// result.BalanceChanges = resp.BalanceChanges

	// // Extract created coin object IDs
	// for _, change := range resp.ObjectChanges {
	// 	if change.Type == "created" {
	// 		if created, ok := change.Data.(map[string]interface{}); ok {
	// 			if objectId, ok := created["objectId"].(string); ok {
	// 				result.CreatedCoins = append(result.CreatedCoins, objectId)
	// 			}
	// 		}
	// 	}
	// }

	return result
}

// Example usage
// func main() {
// 	ctx := context.Background()

// 	// Assume you already have an account (created elsewhere in your application)
// 	// Example: acc, _ := account.NewAccountWithKeystore(privateKeyBytes)
// 	// Or: acc, _ := account.NewAccount(account.Ed25519Flag)
// 	var acc *account.Account // Your existing account

// 	// Configure the split operation
// 	config := SplitCoinConfig{
// 		RPCEndpoint:  "https://fullnode.testnet.sui.io:443",
// 		Account:      acc,
// 		CoinObjectID: "0xYOUR_COIN_OBJECT_ID", // Replace with your coin object ID
// 		Amounts: []uint64{
// 			100000000, // 0.1 SUI
// 			200000000, // 0.2 SUI
// 			300000000, // 0.3 SUI
// 		},
// 		GasObjectID: nil,      // Use same coin for gas (or provide separate gas coin ID)
// 		GasBudget:   10000000, // 0.01 SUI
// 	}

// 	// Execute the split
// 	result := SplitCoin(ctx, config)

// 	// Handle the result
// 	if result.Error != nil {
// 		log.Fatalf("Split coin failed: %v", result.Error)
// 	}

// 	if result.Success {
// 		fmt.Println("✅ Split coin successful!")
// 		fmt.Printf("Transaction Digest: %s\n", result.TransactionDigest)
// 		fmt.Printf("Created %d new coins:\n", len(result.CreatedCoins))
// 		for i, coinId := range result.CreatedCoins {
// 			fmt.Printf("  %d. %s\n", i+1, coinId)
// 		}
// 		fmt.Println("\nBalance Changes:")
// 		for _, balance := range result.BalanceChanges {
// 			fmt.Printf("  - Amount: %s (CoinType: %s)\n", balance.Amount, balance.CoinType)
// 		}
// 	} else {
// 		fmt.Println("❌ Transaction failed")
// 	}
// }

// Helper function: Get coins for an address
func GetCoinsForAddress(ctx context.Context, rpcEndpoint string, acc *account.Account) ([]types.Coin, error) {
	suiClient, err := client.Dial(rpcEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	addrStr := acc.Address
	addrPtr, err := sui_types.NewAddressFromHex(addrStr)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}
	coins, err := suiClient.GetCoins(ctx, *addrPtr, nil, nil, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get coins: %w", err)
	}

	return coins.Data, nil
}