package wallet

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/btcsuite/btcutil/bech32"
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

