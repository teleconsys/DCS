package rebased

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
	suiaccount "github.com/coming-chat/go-sui/v2/account"
	"github.com/coming-chat/go-sui/v2/lib"
	suitypes "github.com/coming-chat/go-sui/v2/sui_types"
)

// Bech32ToKeystoreB64 converts iotaprivkey1... / suiprivkey1... to base64(flag||privkey).
func Bech32ToKeystoreB64(bech string) (string, error) {
	hrp, fiveBit, err := bech32.Decode(strings.TrimSpace(bech))
	if err != nil {
		return "", fmt.Errorf("bech32 decode: %w", err)
	}
	if !strings.EqualFold(hrp, "iotaprivkey") && !strings.EqualFold(hrp, "suiprivkey") {
		return "", fmt.Errorf("unexpected hrp %q (want iotaprivkey/suiprivkey)", hrp)
	}
	raw, err := bech32.ConvertBits(fiveBit, 5, 8, false) // 5-bit words -> bytes
	if err != nil {
		return "", fmt.Errorf("bech32 5->8 convert: %w", err)
	}
	// Expected: 33 bytes total: 0x00 (ed25519 flag) + 32B secret
	if len(raw) != 33 || raw[0] != 0x00 {
		return "", fmt.Errorf("invalid bech32 key payload: want 33B [0x00|32B], got %d (flag=%#02x)", len(raw), raw[0])
	}
	return base64.StdEncoding.EncodeToString(raw), nil // base64(flag||priv)
}

// HexToKeystoreB64 converts hex seed/private key into base64 keystore [0x00|seed32].
// Accepts:
// - 32B seed hex (64 chars)
// - 64B ed25519 private key hex (128 chars) where first 32 bytes are the seed
func HexToKeystoreB64(h string) (string, error) {
	s := strings.TrimSpace(h)
	s = strings.Trim(s, `"'`) // allow quoted
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "0x")

	if !looksLikeHexKey(s) {
		return "", fmt.Errorf("not a supported hex key")
	}

	raw, err := hex.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("hex decode: %w", err)
	}

	switch len(raw) {
	case 32:
		// seed
		buf := append([]byte{0x00}, raw...)
		return base64.StdEncoding.EncodeToString(buf), nil
	case 64:
		// ed25519.PrivateKey = seed(32) || pub(32)
		seed := raw[:32]
		buf := append([]byte{0x00}, seed...)
		return base64.StdEncoding.EncodeToString(buf), nil
	default:
		return "", fmt.Errorf("hex key must be 32B seed or 64B ed25519 private key (got %d bytes)", len(raw))
	}
}

func looksLikeHexKey(s string) bool {
	if len(s) != 64 && len(s) != 128 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

// KeyStringToKeystoreB64 normalizes key strings into base64 keystore format.
// Supports:
// - iotaprivkey1... / suiprivkey1...
// - base64 keystore (33B [0x00|seed])
// - hex seed/private key (32B/64B)
func KeyStringToKeystoreB64(key string) (string, error) {
	s := strings.TrimSpace(key)
	if s == "" {
		return "", errors.New("empty key string")
	}
	s = strings.Trim(s, `"'`)

	low := strings.ToLower(s)

	// bech32
	if strings.HasPrefix(low, "iotaprivkey1") || strings.HasPrefix(low, "suiprivkey1") {
		return Bech32ToKeystoreB64(s)
	}

	// hex
	if ks, err := HexToKeystoreB64(s); err == nil {
		return ks, nil
	}

	// base64 keystore
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("key is not bech32/hex/base64 keystore: %w", err)
	}
	if len(raw) != 33 || raw[0] != 0x00 {
		return "", fmt.Errorf("keystore base64 must be 33B [0x00|32B], got %d (flag=%#02x)", len(raw), raw[0])
	}
	return s, nil
}

// SignTxBytes signs raw tx bytes using a key string.
// Supported key types for conversion are: bech32, base64 keystore, or hex.
func SignTxBytes(ctx context.Context, txBytes []byte, key any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	var ksB64 string
	switch v := key.(type) {
	case string:
		k, err := KeyStringToKeystoreB64(v)
		if err != nil {
			return "", err
		}
		ksB64 = k
	default:
		return "", fmt.Errorf("unsupported key type %T (need string)", key)
	}

	acc, err := suiaccount.NewAccountWithKeystore(ksB64)
	if err != nil {
		return "", fmt.Errorf("load account from keystore: %w", err)
	}

	intent := suitypes.Intent{
		Scope: suitypes.IntentScope{
			TransactionData: &lib.EmptyEnum{},
		},
		Version: suitypes.IntentVersion{
			V0: &lib.EmptyEnum{},
		},
		AppId: suitypes.AppId{
			Sui: &lib.EmptyEnum{},
		},
	}

	sig, err := acc.SignSecureWithoutEncode(txBytes, intent)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	b, _ := json.Marshal(sig)
	var out string
	if err := json.Unmarshal(b, &out); err == nil && out != "" {
		return out, nil
	}

	return "", errors.New("empty signature after marshal")
}
