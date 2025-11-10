// package main

// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"github.com/coming-chat/go-sui/v2/client"
// 	"github.com/coming-chat/go-sui/v2/lib"
// 	"github.com/coming-chat/go-sui/v2/sui_types"
// 	"github.com/coming-chat/go-sui/v2/types"
// )

// // Example: execute a SplitCoinUnsignedRPC transaction manually
// func main() {
// 	ctx := context.Background()

// 	// ---- 1) Connect to node ----
// 	rpcURL := "https://fullnode.testnet.iota.cafe:443" // or your IOTA Rebased endpoint
// 	rpc, err := client.Dial(rpcURL)
// 	if err != nil {
// 		panic(fmt.Errorf("failed to connect: %w", err))
// 	}

// 	// ---- 2) Prepare transaction bytes (from SplitCoinUnsignedRPC) ----
// 	txBytesBase64 := "AAACAQBV7UXr1Hp8MVhWhxsZDjWxRXhShi+kKusAL6834b6pCgLVgCEAAAAAIBkD0u0zrQWeAka8X7DRx6MOTl9GkKQx1UkQVxMBlhsuAAkBAMqaOwAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACA3BheQlzcGxpdF92ZWMBBwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACBGlvdGEESU9UQQACAQAAAQEA6oGAIF1MQvFlCD0qQto0symshm+MDsy5tFi7akEAkpYBVe1F69R6fDFYVocbGQ41sUV4UoYvpCrrAC+vN+G+qQoC1YAhAAAAACAZA9LtM60FngJGvF+w0cejDk5fRpCkMdVJEFcTAZYbLuqBgCBdTELxZQg9KkLaNLMprIZvjA7MubRYu2pBAJKW6AMAAAAAAAAAypo7AAAAAAA="

// 	// ---- 3) Prepare your private key ----
// 	// Example for an ed25519 key (base64 or hex)
// 	// NOTE: replace with your actual private key
// 	keyHex := os.Getenv("USER_PRIVATE_KEY")
// 	privKey, err := sui_types.NewPrivateKeyFromHex(keyHex)
// 	if err != nil {
// 		panic(fmt.Errorf("invalid private key: %w", err))
// 	}

// 	// ---- 4) Sign the transaction ----
// 	txBytes := lib.Base64Data(txBytesBase64)
// 	signature, err := privKey.SignSecure([]byte(txBytes))
// 	if err != nil {
// 		panic(fmt.Errorf("failed to sign tx: %w", err))
// 	}

// 	// ---- 5) Encode the signature in base64 for RPC ----
// 	sigB64 := lib.Base64Data(signature)

// 	// ---- 6) Create execution options ----
// 	opts := &types.SuiTransactionBlockResponseOptions{
// 		ShowInput:          true,
// 		ShowEffects:        true,
// 		ShowEvents:         true,
// 		ShowBalanceChanges: true,
// 		ShowObjectChanges:  true,
// 	}

// 	// ---- 7) Execute transaction ----
// 	reqType := types.ExecuteTransactionRequestTypeWaitForLocalExecution
// 	var rsp types.SuiTransactionBlockResponse

// 	err = rpc.CallContext(ctx, &rsp, "iota_executeTransactionBlock",
// 		txBytesBase64, []any{sigB64}, opts, reqType,
// 	)
// 	if err != nil {
// 		panic(fmt.Errorf("execution failed: %w", err))
// 	}

// 	// ---- 8) Print result ----
// 	fmt.Printf("Transaction digest: %s\n", rsp.Digest)
// 	fmt.Printf("Effects: %+v\n", rsp.Effects)
// }
