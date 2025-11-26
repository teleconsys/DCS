package wallet

import (
	"bufio"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/blake2b"

	"github.com/teleconsys/DCS/internal/rebased"
)

// DeriveAddress derives an address from an ed25519 public key.
// Ed25519 address = 0x + blake2b-256(pubkey)
func DeriveAddress(pub ed25519.PublicKey) string {
	sum := blake2b.Sum256(pub)
	return "0x" + hex.EncodeToString(sum[:])
}

// PrivateKeyToAddress converts a private key (bech32 or base64 keystore)
// to its corresponding public key and address.
//
// The private key can be in either format:
//   - bech32: iotaprivkey1... or suiprivkey1...
//   - base64: base64-encoded 33-byte [0x00 | 32-byte ed25519 secret] keystore
//
// Returns the ed25519 public key and the derived address.
func PrivateKeyToAddress(privKey string) (string, error) {
	privKey = strings.TrimSpace(privKey)
	if privKey == "" {
		return "", errors.New("private key is empty")
	}

	// Convert to keystore base64 format if needed
	var ksB64 string
	lower := strings.ToLower(privKey)
	if strings.HasPrefix(lower, "iotaprivkey1") || strings.HasPrefix(lower, "suiprivkey1") {
		// Bech32 format - convert to base64 keystore
		var err error
		ksB64, err = rebased.Bech32ToKeystoreB64(privKey)
		if err != nil {
			return "", fmt.Errorf("convert bech32 to keystore: %w", err)
		}
	} else {
		// Assume base64 keystore format
		raw, err := base64.StdEncoding.DecodeString(privKey)
		if err != nil {
			return "", fmt.Errorf("invalid base64 keystore: %w", err)
		}
		if len(raw) != 33 || raw[0] != 0x00 {
			return "", fmt.Errorf("keystore must be 33B [0x00|32B], got %d bytes (flag=%#02x)", len(raw), raw[0])
		}
		ksB64 = privKey
	}

	// Decode keystore to get the seed (32 bytes after the 0x00 flag)
	raw, err := base64.StdEncoding.DecodeString(ksB64)
	if err != nil {
		return "", fmt.Errorf("decode keystore base64: %w", err)
	}
	if len(raw) != 33 || raw[0] != 0x00 {
		return "", fmt.Errorf("invalid keystore format: want 33B [0x00|32B], got %d bytes", len(raw))
	}
	seed := raw[1:33] // Extract 32-byte seed

	// Derive ed25519 keypair from seed
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	// Derive address from public key
	addr := DeriveAddress(pub)

	return addr, nil
}

// ResolvePrivateKey resolves the private key to use for signing and the corresponding address.
//
// Precedence:
//  1. If flagValue is non-empty, it is validated and returned.
//  2. If ACTIVE_PRIVATE_KEY env var is set, it is validated and returned.
//  3. Otherwise, the user is prompted on stdin to enter the key. In this case,
//     the key is stored in the ACTIVE_PRIVATE_KEY env var.
//
// The key must be either:
//   - a bech32 string starting with iotaprivkey1... or suiprivkey1..., or
//   - a base64-encoded 33-byte [0x00 | 32-byte ed25519 secret] keystore.
func ResolvePrivateKey(flagValue string) (string, error) {
	// 1. flag value
	if privKey := strings.TrimSpace(flagValue); privKey != "" {
		if err := validatePrivateKeyFormat(privKey); err != nil {
			return "", err
		}
		return privKey, nil
	}

	// 2. ACTIVE_PRIVATE_KEY env var
	if privKey := os.Getenv("ACTIVE_PRIVATE_KEY"); privKey != "" {
		if err := validatePrivateKeyFormat(privKey); err != nil {
			return "", err
		}
		return privKey, nil
	}

	// 3. interactive prompt
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr,
		"Enter private key (iotaprivkey1... / suiprivkey1... or base64 keystore): ",
	)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading private key from stdin: %w", err)
	}
	key := strings.TrimSpace(line)
	if err := validatePrivateKeyFormat(key); err != nil {
		return "", err
	}

	// TODO set private key in the .env file
	return key, nil
}

// validatePrivateKeyFormat performs format checks against the
// formats supported by rebased.SignTxBytes.
func validatePrivateKeyFormat(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return errors.New("private key not in accepted format (empty string)")
	}

	lower := strings.ToLower(s)

	// 1) Bech32 form: iotaprivkey1... / suiprivkey1...
	if strings.HasPrefix(lower, "iotaprivkey1") || strings.HasPrefix(lower, "suiprivkey1") {
		// Run through the bech32 decoder from rebased to catch obvious mistakes.
		if _, err := rebased.Bech32ToKeystoreB64(s); err != nil {
			return fmt.Errorf(
				"private key not in accepted format (invalid iotaprivkey1.../suiprivkey1... bech32): %w",
				err,
			)
		}
		return nil
	}

	// 2) Base64 form: 33 bytes [0x00 | 32-byte secret]
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return fmt.Errorf(
			"private key not in accepted format (expected iotaprivkey1.../suiprivkey1... or base64 keystore): %w",
			err,
		)
	}
	if len(raw) != 33 || raw[0] != 0x00 {
		return fmt.Errorf(
			"private key not in accepted format (expected base64-encoded [0x00 | 32-byte ed25519 secret], got %d bytes)",
			len(raw),
		)
	}

	return nil
}

// ResolveSignerAddress resolves the signer address to use for a transaction.
//
// It performs the following steps:
//  1. Derives the address from the private key
//  2. Reads the address from flag/env
//  3. Compares the two addresses:
//     3.1. If they match, returns the address
//     3.2. If they don't match, prompts the user for confirmation to use the address
//     derived from the private key
//
// Parameters:
//   - privKey: The private key (already resolved)
//   - flagSignerAddress: The value from the --signer-address flag
//   - envVars: Optional environment variable names to check (e.g., "PROVIDER_ADDRESS", "USER_ADDRESS")
//
// Returns the address to use for signing.
func ResolveSignerAddress(privKey string, flagSignerAddress string, envVars ...string) (string, error) {
	// 1. Derive address from private key
	derivedAddr, err := PrivateKeyToAddress(privKey)
	if err != nil {
		return "", fmt.Errorf("error deriving address from private key: %w", err)
	}

	// 2. Read address from flag/env
	var envValues []string
	for _, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			envValues = append(envValues, val)
		}
	}
	flagOrEnvAddr := FirstNonEmpty(append([]string{flagSignerAddress}, envValues...)...)

	// If no flag/env address is provided, use the derived address
	if flagOrEnvAddr == "" {
		// TODO set address in the .env file
		fmt.Fprintf(os.Stderr, "Using address: %s\n", derivedAddr)
		return derivedAddr, nil
	}

	// 3. Compare addresses
	if derivedAddr == flagOrEnvAddr {
		// 3.1. Addresses match, use it
		// TODO set address in the .env file
		fmt.Fprintf(os.Stderr, "Using address: %s\n", derivedAddr)
		return derivedAddr, nil
	}

	// 3.2. Addresses don't match, ask for confirmation
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stderr,
		"⚠️  Address mismatch detected:\n"+
			"  Address from private key: %s\n"+
			"  Address from flag/env:    %s\n"+
			"Use address from private key (%s)? [y/N]: ",
		derivedAddr, flagOrEnvAddr, derivedAddr,
	)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading confirmation from stdin: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))
	if response == "y" || response == "yes" || response == "Y" {
		// TODO set address in the .env file
		fmt.Fprintf(os.Stderr, "Using address: %s\n", derivedAddr)
		return derivedAddr, nil
	}

	// User declined, return error
	return "", fmt.Errorf(
		"address mismatch: private key address (%s) does not match flag/env address (%s)",
		derivedAddr, flagOrEnvAddr,
	)
}

// FirstNonEmpty returns the first non-empty string from the provided values
func FirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
