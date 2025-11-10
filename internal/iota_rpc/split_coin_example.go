package iota_rpc

import (
	"context"
	"fmt"
)

// Example usage of UnsafeSplitCoin
func ExampleUnsafeSplitCoin() {
	ctx := context.Background()
	endpoint := "https://api.testnet.iota.cafe:443"

	params := SplitCoinParams{
		Signer:       "0xea8180205d4c42f165083d2a42da34b329ac866f8c0eccb9b458bb6a41009296",
		CoinObjectID: "0x55ed45ebd47a7c315856871b190e35b1457852862fa42aeb002faf37e1bea90a",
		SplitAmounts: []string{"1000000000"},
		Gas:          nil, // nil means let the node pick a gas object
		GasBudget:    "1000000000",
	}

	txBytes, err := UnsafeSplitCoin(ctx, endpoint, params)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Transaction bytes: %s\n", txBytes)
}

// Example with explicit gas object
func ExampleUnsafeSplitCoinWithGas() {
	ctx := context.Background()
	endpoint := "https://api.testnet.iota.cafe:443"

	gasObjectID := "0x55ed45ebd47a7c315856871b190e35b1457852862fa42aeb002faf37e1bea90a"
	params := SplitCoinParams{
		Signer:       "0xea8180205d4c42f165083d2a42da34b329ac866f8c0eccb9b458bb6a41009296",
		CoinObjectID: "0x55ed45ebd47a7c315856871b190e35b1457852862fa42aeb002faf37e1bea90a",
		SplitAmounts: []string{"1000000000"},
		Gas:          &gasObjectID, // Explicitly specify gas object
		GasBudget:    "1000000000",
	}

	txBytes, err := UnsafeSplitCoin(ctx, endpoint, params)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Transaction bytes: %s\n", txBytes)
}


