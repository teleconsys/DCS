package wallet

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"golang.org/x/crypto/blake2b"

	"github.com/teleconsys/DCS/internal/rebased"
)

// DeriveAddress derives an address from an ed25519 public key.
// Ed25519 address = 0x + blake2b-256(pubkey)
func DeriveAddress(pub ed25519.PublicKey) string {
	sum := blake2b.Sum256(pub)
	return "0x" + hex.EncodeToString(sum[:])
}

// PrivateKeyToAddress converts a private key string to its corresponding address.
func PrivateKeyToAddress(privKey string) (string, error) {
	privKey = strings.TrimSpace(privKey)
	if privKey == "" {
		return "", errors.New("private key is empty")
	}

	ksB64, err := rebased.KeyStringToKeystoreB64(privKey)
	if err != nil {
		return "", fmt.Errorf("private key not in accepted format: %w", err)
	}

	raw, err := base64.StdEncoding.DecodeString(ksB64)
	if err != nil {
		return "", fmt.Errorf("decode keystore base64: %w", err)
	}
	if len(raw) != 33 || raw[0] != 0x00 {
		return "", fmt.Errorf("invalid keystore format: want 33B [0x00|32B], got %d bytes (flag=%#02x)", len(raw), raw[0])
	}

	seed := raw[1:33]
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	return DeriveAddress(pub), nil
}

// ResolvePrivateKey resolves the private key to use for signing.
func ResolvePrivateKey(flagValue string) (string, error) {
	// 1) flag
	if privKey := strings.TrimSpace(flagValue); privKey != "" {
		if err := validatePrivateKeyFormat(privKey); err != nil {
			return "", err
		}
		return privKey, nil
	}

	// 2) env fallbacks
	for _, k := range []string{"ACTIVE_PRIVATE_KEY", "USER_PRIVATE_KEY", "PROVIDER_PRIVATE_KEY"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			if err := validatePrivateKeyFormat(v); err == nil {
				return v, nil
			}
		}
	}

	// 3) prompt
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr, "Enter private key (bech32/base64/hex): ")
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading private key from stdin: %w", err)
	}
	key := strings.TrimSpace(line)
	if err := validatePrivateKeyFormat(key); err != nil {
		return "", err
	}
	return key, nil
}

// validatePrivateKeyFormat validates the key using rebased conversion
func validatePrivateKeyFormat(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return errors.New("private key not in accepted format (empty string)")
	}

	ksB64, err := rebased.KeyStringToKeystoreB64(s)
	if err != nil {
		return fmt.Errorf("private key not in accepted format: %w", err)
	}

	raw, err := base64.StdEncoding.DecodeString(ksB64)
	if err != nil {
		return fmt.Errorf("private key not in accepted format (invalid keystore base64): %w", err)
	}
	if len(raw) != 33 || raw[0] != 0x00 {
		return fmt.Errorf("private key not in accepted format (expected 33B [0x00|32B], got %d bytes)", len(raw))
	}

	return nil
}

// ResolveSignerAddress resolves the signer address to use for a transaction
func ResolveSignerAddress(privKey string, flagSignerAddress string, envVars ...string) (string, error) {
	derivedAddr, err := PrivateKeyToAddress(privKey)
	if err != nil {
		return "", fmt.Errorf("error deriving address from private key: %w", err)
	}

	var envValues []string
	for _, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			envValues = append(envValues, val)
		}
	}
	flagOrEnvAddr := FirstNonEmpty(append([]string{flagSignerAddress}, envValues...)...)

	if flagOrEnvAddr == "" {
		fmt.Fprintf(os.Stderr, "🔑 Using address: %s\n", derivedAddr)
		return derivedAddr, nil
	}

	if strings.EqualFold(derivedAddr, flagOrEnvAddr) {
		fmt.Fprintf(os.Stderr, "🔑 Using address: %s\n", derivedAddr)
		return derivedAddr, nil
	}

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
	if response == "y" || response == "yes" {
		fmt.Fprintf(os.Stderr, "🔑 Using address: %s\n", derivedAddr)
		return derivedAddr, nil
	}

	return "", fmt.Errorf(
		"address mismatch: private key address (%s) does not match flag/env address (%s)",
		derivedAddr, flagOrEnvAddr,
	)
}

// FirstNonEmpty returns the first non-empty string from the provided values.
func FirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// ResolveGasCoinId resolves the gas coin ID to use for a transaction and verifies ownership.
func ResolveGasCoinId(ctx context.Context, flagValue string, signerAddress string, rpcURL string, envVars ...string) (string, error) {
	var envValues []string
	for _, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			envValues = append(envValues, val)
		}
	}
	gasID := FirstNonEmpty(append([]string{flagValue}, envValues...)...)
	if gasID == "" {
		return "", fmt.Errorf("missing gas coin id (set --signer-gas-id or env vars: %v)", envVars)
	}

	// Get RPC URL if not already set
	if rpcURL == "" {
		rpcURL = os.Getenv("REBASE_RPC")
		if rpcURL == "" {
			rpcURL = os.Getenv("DCS_RPC")
		}
		if rpcURL == "" {
			rpcURL = "https://api.testnet.iota.cafe:443"
		}
	}

	w, err := rebased.Dial(rpcURL)
	if err != nil {
		return "", fmt.Errorf("rpc dial failed: %w", err)
	}

	obj, err := w.GetObject(ctx, gasID, suitypes.SuiObjectDataOptions{ShowOwner: true})
	if err != nil {
		return "", fmt.Errorf("failed to get gas coin object: %w", err)
	}
	if obj == nil || obj.Data == nil {
		return "", fmt.Errorf("gas coin object %s not found", gasID)
	}
	if obj.Error != nil {
		return "", fmt.Errorf("rpc getObject error: %+v", obj.Error)
	}

	ownerAddr, err := extractOwnerAddress(obj.Data)
	if err != nil {
		return "", fmt.Errorf("failed to extract owner address from gas coin: %w", err)
	}

	if !strings.EqualFold(ownerAddr, signerAddress) {
		return "", fmt.Errorf(
			"gas coin %s belongs to address %s, but signer address is %s",
			gasID, ownerAddr, signerAddress,
		)
	}

	return gasID, nil
}

// extractOwnerAddress extracts the owner address from SuiObjectData.
func extractOwnerAddress(data *suitypes.SuiObjectData) (string, error) {
	if data == nil {
		return "", errors.New("object data is nil")
	}
	if data.Owner == nil {
		return "", errors.New("owner is nil")
	}

	ownerJSON, err := json.Marshal(data.Owner)
	if err != nil {
		return "", fmt.Errorf("marshal owner: %w", err)
	}

	var ownerMap map[string]interface{}
	if err := json.Unmarshal(ownerJSON, &ownerMap); err != nil {
		return "", fmt.Errorf("unmarshal owner: %w", err)
	}

	if addrOwner, ok := ownerMap["AddressOwner"]; ok {
		if addr, ok := addrOwner.(string); ok {
			return addr, nil
		}
	}

	if objOwner, ok := ownerMap["ObjectOwner"]; ok {
		if addr, ok := objOwner.(string); ok {
			return addr, nil
		}
	}

	if _, ok := ownerMap["Shared"]; ok {
		return "", errors.New("gas coin cannot be a shared object")
	}
	if _, ok := ownerMap["Immutable"]; ok {
		return "", errors.New("gas coin cannot be an immutable object")
	}

	return "", errors.New("unknown owner type or owner structure")
}
