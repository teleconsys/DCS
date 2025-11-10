package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/teleconsys/DCS/internal/cid"
	"github.com/teleconsys/DCS/internal/config"
)

func main() {
	// Load .env first
	config.LoadEnv()
	fmt.Println("📄 File .env caricato\n")

	// Get RPC URL
	rpcURL := getEnvOrDefault("REBASE_RPC", "https://api.testnet.iota.cafe:443")

	// Required envs
	coinID := os.Getenv("USER_GAS_COIN_ID")
	if coinID == "" {
		fmt.Println("❌ Missing USER_GAS_COIN_ID in .env")
		os.Exit(1)
	}

	userAddr := os.Getenv("USER_ADDRESS")
	if userAddr == "" {
		fmt.Println("❌ Missing USER_ADDRESS in .env")
		os.Exit(1)
	}

	privKey := os.Getenv("USER_PRIVATE_KEY")
	if privKey == "" {
		fmt.Println("❌ Missing USER_PRIVATE_KEY in .env")
		os.Exit(1)
	}

	// Amount to split out; default 1_000_000
	amount := getAmountFromEnv("SPLIT_AMOUNT", 1_000_000)
	gasBudget := getAmountFromEnv("WALLET_GAS_BUDGET", 10_000_000)

	fmt.Println("🔧 SplitCoin standalone test")
	fmt.Println("===========================")
	fmt.Printf("RPC URL: %s\n", rpcURL)
	fmt.Printf("User Address: %s\n", userAddr)
	fmt.Printf("Coin to split: %s\n", coinID)
	fmt.Printf("Gas Budget: %d\n", gasBudget)
	fmt.Printf("Amount: %d nanoIOTA\n\n", amount)

	ctx := context.Background()
    fmt.Println("⏳ Executing split coin...")

    // Prepare params for SplitCoin from internal/cid/create.go
    params := cid.CreateParams{
        RPCURL:            rpcURL,
        UserSignerAddress: userAddr,
        UserPrivateKey:    privKey,
        GasID:             coinID,   // use this coin as gas too
        GasBudget:         gasBudget,
    }

    newCoinID, err := cid.SplitCoin(ctx, params, coinID, int64(amount))
    if err != nil {
        fmt.Printf("❌ SplitCoin failed: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("✅ Split coin executed successfully!\n")
    fmt.Printf("New coin ID: %s\n", newCoinID)
    fmt.Println("\n✅ Split coin test completed!")
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getAmountFromEnv(key string, defaultVal uint64) uint64 {
	if s := os.Getenv(key); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			return v
		}
	}
	return defaultVal
}