package wallet

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/joho/godotenv"

	"github.com/teleconsys/DCS/internal/config"
)

// ed25519PrivToIotaPrivKey is a test helper that encodes an ed25519 private key
// into the iotaprivkey1... bech32 format used by IOTA Rebased / Sui keystores.
func ed25519PrivToIotaPrivKey(priv ed25519.PrivateKey) (string, error) {
	// ed25519.PrivateKey.Seed() returns the 32-byte private seed.
	seed := priv.Seed()
	if len(seed) != ed25519.SeedSize {
		return "", fmt.Errorf("unexpected ed25519 seed length: %d", len(seed))
	}

	// 0x00 is the Ed25519 scheme flag (same as used by Sui / IOTA keystore).
	raw := append([]byte{0x00}, seed...)

	// 8-bit -> 5-bit groups, with padding.
	fiveBit, err := bech32.ConvertBits(raw, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("bech32 8->5 convert: %w", err)
	}

	// HRP is iotaprivkey for IOTA Rebased.
	return bech32.Encode("iotaprivkey", fiveBit)
}

func TestPrivateKeyToAddress(t *testing.T) {
	// Generate a test keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	// Get the expected address using DeriveAddress
	expectedAddr := DeriveAddress(pub)

	// Test 1: Base64 keystore format
	seed := priv.Seed()
	keystoreBytes := append([]byte{0x00}, seed...)
	keystoreB64 := base64.StdEncoding.EncodeToString(keystoreBytes)

	addr, err := PrivateKeyToAddress(keystoreB64)
	if err != nil {
		t.Fatalf("PrivateKeyToAddress(base64) failed: %v", err)
	}
	if addr != expectedAddr {
		t.Errorf("Address mismatch for base64 format:\n  got:  %s\n  want: %s", addr, expectedAddr)
	}
	t.Logf("✓ Base64 format test passed: address = %s", addr)

	// Test 2: Bech32 format (iotaprivkey1...)
	iotapriv, err := ed25519PrivToIotaPrivKey(priv)
	if err != nil {
		t.Fatalf("Failed to convert private key to bech32: %v", err)
	}

	addr2, err := PrivateKeyToAddress(iotapriv)
	if err != nil {
		t.Fatalf("PrivateKeyToAddress(bech32) failed: %v", err)
	}
	if addr2 != expectedAddr {
		t.Errorf("Address mismatch for bech32 format:\n  got:  %s\n  want: %s", addr2, expectedAddr)
	}
	if addr2 != addr {
		t.Errorf("Address mismatch between formats:\n  base64: %s\n  bech32: %s", addr, addr2)
	}
	t.Logf("✓ Bech32 format test passed: address = %s", addr2)

	// Test 3: Both formats should produce the same address
	if addr != addr2 {
		t.Errorf("Different addresses from same key:\n  base64: %s\n  bech32: %s", addr, addr2)
	} else {
		t.Logf("✓ Both formats produce the same address: %s", addr)
	}
}

func TestPrivateKeyToAddress_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		privKey string
		wantErr bool
	}{
		{
			name:    "empty string",
			privKey: "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			privKey: "   ",
			wantErr: true,
		},
		{
			name:    "invalid base64",
			privKey: "not-base64!!!",
			wantErr: true,
		},
		{
			name:    "base64 too short",
			privKey: base64.StdEncoding.EncodeToString([]byte{0x00, 0x01, 0x02}),
			wantErr: true,
		},
		{
			name:    "base64 wrong flag",
			privKey: base64.StdEncoding.EncodeToString(make([]byte, 33)), // all zeros, flag should be 0x00 but seed is all zeros
			wantErr: false,                                               // This might not error, but it's a valid format
		},
		{
			name:    "invalid bech32 prefix",
			privKey: "invalid1abc123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := PrivateKeyToAddress(tt.privKey)
			if tt.wantErr {
				if err == nil {
					t.Errorf("PrivateKeyToAddress() expected error but got address: %s", addr)
				} else {
					t.Logf("✓ Expected error occurred: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("PrivateKeyToAddress() unexpected error: %v", err)
				} else {
					t.Logf("✓ No error (expected): address = %s", addr)
				}
			}
		})
	}
}

func TestPrivateKeyToAddress_Consistency(t *testing.T) {
	// Test that the same private key always produces the same address
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	seed := priv.Seed()
	keystoreBytes := append([]byte{0x00}, seed...)
	keystoreB64 := base64.StdEncoding.EncodeToString(keystoreBytes)

	// Call multiple times
	addr1, err1 := PrivateKeyToAddress(keystoreB64)
	if err1 != nil {
		t.Fatalf("First call failed: %v", err1)
	}

	addr2, err2 := PrivateKeyToAddress(keystoreB64)
	if err2 != nil {
		t.Fatalf("Second call failed: %v", err2)
	}

	addr3, err3 := PrivateKeyToAddress(keystoreB64)
	if err3 != nil {
		t.Fatalf("Third call failed: %v", err3)
	}

	if addr1 != addr2 || addr2 != addr3 {
		t.Errorf("Address not consistent across calls:\n  call1: %s\n  call2: %s\n  call3: %s", addr1, addr2, addr3)
	} else {
		t.Logf("✓ Address is consistent: %s", addr1)
	}
}

// TestResolveGasCoinId tests the ResolveGasCoinId function with live blockchain data.
// This test requires DCS_LIVE=1 and valid environment variables:
//   - ACTIVE_GAS_COIN_ID or USER_GAS_COIN_ID or PROVIDER_GAS_COIN_ID: a valid gas coin object ID
//   - ACTIVE_ADDRESS or USER_ADDRESS or PROVIDER_ADDRESS: the address that owns the gas coin
//   - REBASE_RPC: RPC URL (optional, defaults to testnet)
func TestResolveGasCoinId(t *testing.T) {
	if os.Getenv("DCS_LIVE") == "" {
		t.Skip("set DCS_LIVE=1 to run this live test")
	}

	// Load environment variables from .env file
	// Try to find .env file in multiple locations
	wd, _ := os.Getwd()
	// t.Logf("Current working directory: %s", wd) // Debug: uncomment to see working directory

	// Try current directory first
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err == nil {
			// t.Logf("✓ Loaded .env from current directory") // Debug: uncomment to see .env location
		}
	} else {
		// Try project root (go up from internal/wallet)
		projectRoot := filepath.Join(wd, "..", "..")
		envPath := filepath.Join(projectRoot, ".env")
		if _, err := os.Stat(envPath); err == nil {
			if err := godotenv.Load(envPath); err == nil {
				// t.Logf("✓ Loaded .env from project root: %s", envPath) // Debug: uncomment to see .env location
			}
		} else {
			// Try one more level up (in case we're in a subdirectory)
			projectRoot2 := filepath.Join(wd, "..", "..", "..")
			envPath2 := filepath.Join(projectRoot2, ".env")
			if _, err := os.Stat(envPath2); err == nil {
				if err := godotenv.Load(envPath2); err == nil {
					// t.Logf("✓ Loaded .env from: %s", envPath2) // Debug: uncomment to see .env location
				}
			} else {
				// t.Logf("⚠ .env file not found in current dir, project root, or parent") // Debug: uncomment for troubleshooting
				// Fallback to config.LoadEnv() which uses current working dir
				config.LoadEnv()
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get RPC URL
	rpcURL := os.Getenv("REBASE_RPC")
	if rpcURL == "" {
		rpcURL = "https://api.testnet.iota.cafe:443"
	}

	// Get signer address (from ACTIVE_ADDRESS, USER_ADDRESS, or PROVIDER_ADDRESS)
	signerAddress := os.Getenv("ACTIVE_ADDRESS")
	if signerAddress == "" {
		signerAddress = os.Getenv("USER_ADDRESS")
	}
	if signerAddress == "" {
		signerAddress = os.Getenv("PROVIDER_ADDRESS")
	}

	// t.Logf("Selected signerAddress: %s", signerAddress) // Debug: uncomment to see selected address

	if signerAddress == "" {
		t.Fatal("missing signer address: set ACTIVE_ADDRESS, USER_ADDRESS, or PROVIDER_ADDRESS")
	}

	// Test 1: Gas coin ID from flag (highest priority)
	t.Run("from_flag", func(t *testing.T) {
		gasCoinID := os.Getenv("ACTIVE_GAS_COIN_ID")
		if gasCoinID == "" {
			gasCoinID = os.Getenv("USER_GAS_COIN_ID")
		}
		if gasCoinID == "" {
			gasCoinID = os.Getenv("PROVIDER_GAS_COIN_ID")
		}
		if gasCoinID == "" {
			t.Skip("skipping: no gas coin ID available in env vars")
		}

		// Clear env vars to test flag priority
		oldActive := os.Getenv("ACTIVE_GAS_COIN_ID")
		oldUser := os.Getenv("USER_GAS_COIN_ID")
		oldProvider := os.Getenv("PROVIDER_GAS_COIN_ID")
		os.Unsetenv("ACTIVE_GAS_COIN_ID")
		os.Unsetenv("USER_GAS_COIN_ID")
		os.Unsetenv("PROVIDER_GAS_COIN_ID")
		defer func() {
			if oldActive != "" {
				os.Setenv("ACTIVE_GAS_COIN_ID", oldActive)
			}
			if oldUser != "" {
				os.Setenv("USER_GAS_COIN_ID", oldUser)
			}
			if oldProvider != "" {
				os.Setenv("PROVIDER_GAS_COIN_ID", oldProvider)
			}
		}()

		resolved, err := ResolveGasCoinId(ctx, gasCoinID, signerAddress, rpcURL, "ACTIVE_GAS_COIN_ID", "USER_GAS_COIN_ID", "PROVIDER_GAS_COIN_ID")
		if err != nil {
			t.Fatalf("ResolveGasCoinId(from flag) failed: %v", err)
		}
		if resolved != gasCoinID {
			t.Errorf("ResolveGasCoinId() = %s, want %s", resolved, gasCoinID)
		}
		t.Logf("✓ Gas coin ID resolved from flag: %s", resolved)
	})

	// Test 2: Gas coin ID from env var
	t.Run("from_env", func(t *testing.T) {
		gasCoinID := os.Getenv("ACTIVE_GAS_COIN_ID")
		if gasCoinID == "" {
			gasCoinID = os.Getenv("USER_GAS_COIN_ID")
		}
		if gasCoinID == "" {
			gasCoinID = os.Getenv("PROVIDER_GAS_COIN_ID")
		}
		if gasCoinID == "" {
			t.Skip("skipping: no gas coin ID available in env vars")
		}

		// Pass empty flag to test env var priority
		resolved, err := ResolveGasCoinId(ctx, "", signerAddress, rpcURL, "ACTIVE_GAS_COIN_ID", "USER_GAS_COIN_ID", "PROVIDER_GAS_COIN_ID")
		if err != nil {
			t.Fatalf("ResolveGasCoinId(from env) failed: %v", err)
		}
		if resolved != gasCoinID {
			t.Errorf("ResolveGasCoinId() = %s, want %s", resolved, gasCoinID)
		}
		t.Logf("✓ Gas coin ID resolved from env: %s", resolved)
	})

	// Test 3: Missing gas coin ID (should error)
	t.Run("missing_gas_coin", func(t *testing.T) {
		// Temporarily clear env vars
		oldActive := os.Getenv("ACTIVE_GAS_COIN_ID")
		oldUser := os.Getenv("USER_GAS_COIN_ID")
		oldProvider := os.Getenv("PROVIDER_GAS_COIN_ID")
		os.Unsetenv("ACTIVE_GAS_COIN_ID")
		os.Unsetenv("USER_GAS_COIN_ID")
		os.Unsetenv("PROVIDER_GAS_COIN_ID")
		defer func() {
			if oldActive != "" {
				os.Setenv("ACTIVE_GAS_COIN_ID", oldActive)
			}
			if oldUser != "" {
				os.Setenv("USER_GAS_COIN_ID", oldUser)
			}
			if oldProvider != "" {
				os.Setenv("PROVIDER_GAS_COIN_ID", oldProvider)
			}
		}()

		_, err := ResolveGasCoinId(ctx, "", signerAddress, rpcURL, "ACTIVE_GAS_COIN_ID", "USER_GAS_COIN_ID", "PROVIDER_GAS_COIN_ID")
		if err == nil {
			t.Error("ResolveGasCoinId() expected error for missing gas coin ID, got nil")
		} else {
			t.Logf("✓ Expected error for missing gas coin ID: %v", err)
		}
	})

	// Test 4: Valid gas coin that belongs to signer (should succeed)
	t.Run("valid_ownership", func(t *testing.T) {
		gasCoinID := os.Getenv("ACTIVE_GAS_COIN_ID")
		if gasCoinID == "" {
			gasCoinID = os.Getenv("USER_GAS_COIN_ID")
		}
		if gasCoinID == "" {
			gasCoinID = os.Getenv("PROVIDER_GAS_COIN_ID")
		}
		if gasCoinID == "" {
			t.Skip("skipping: no gas coin ID available in env vars")
		}

		resolved, err := ResolveGasCoinId(ctx, gasCoinID, signerAddress, rpcURL)
		if err != nil {
			t.Fatalf("ResolveGasCoinId(valid ownership) failed: %v", err)
		}
		if resolved != gasCoinID {
			t.Errorf("ResolveGasCoinId() = %s, want %s", resolved, gasCoinID)
		}
		t.Logf("✓ Valid ownership verified: %s belongs to %s", resolved, signerAddress)
	})

	// Test 5: Invalid gas coin ID (non-existent object, should error)
	t.Run("invalid_gas_coin_id", func(t *testing.T) {
		invalidID := "0x0000000000000000000000000000000000000000000000000000000000000000"
		_, err := ResolveGasCoinId(ctx, invalidID, signerAddress, rpcURL)
		if err == nil {
			t.Error("ResolveGasCoinId() expected error for invalid gas coin ID, got nil")
		} else {
			t.Logf("✓ Expected error for invalid gas coin ID: %v", err)
		}
	})

	// Test 6: Gas coin owned by different address (should error)
	t.Run("wrong_owner", func(t *testing.T) {
		gasCoinID := os.Getenv("ACTIVE_GAS_COIN_ID")
		if gasCoinID == "" {
			gasCoinID = os.Getenv("USER_GAS_COIN_ID")
		}
		if gasCoinID == "" {
			gasCoinID = os.Getenv("PROVIDER_GAS_COIN_ID")
		}
		if gasCoinID == "" {
			t.Skip("skipping: no gas coin ID available in env vars")
		}

		// Use a different address (just change the last character)
		wrongAddress := signerAddress[:len(signerAddress)-1] + "0"
		if wrongAddress == signerAddress {
			// If they're the same, use a different approach
			wrongAddress = "0x0000000000000000000000000000000000000000000000000000000000000000"
		}

		_, err := ResolveGasCoinId(ctx, gasCoinID, wrongAddress, rpcURL)
		if err == nil {
			t.Error("ResolveGasCoinId() expected error for wrong owner, got nil")
		} else {
			t.Logf("✓ Expected error for wrong owner: %v", err)
		}
	})
}
