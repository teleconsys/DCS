package rebased

import (
	"context"
	"encoding/base64"
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

// SignTxBytes signs raw tx bytes using either a bech32 key string (iotaprivkey1.../suiprivkey1...)
// or a keystore base64 string (flag||priv). Returns the base64 SerializedSignature.
func SignTxBytes(ctx context.Context, txBytes []byte, key any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	var ksB64 string
	switch v := key.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return "", errors.New("empty key string")
		}
		if strings.HasPrefix(strings.ToLower(s), "iotaprivkey1") || strings.HasPrefix(strings.ToLower(s), "suiprivkey1") {
			k, err := Bech32ToKeystoreB64(s)
			if err != nil {
				return "", err
			}
			ksB64 = k
		} else {
			// assume keystore base64; quick sanity: must decode to 33 bytes
			raw, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return "", fmt.Errorf("key is not bech32 nor base64 keystore: %w", err)
			}
			if len(raw) != 33 || raw[0] != 0x00 {
				return "", fmt.Errorf("keystore base64 must be 33B [0x00|32B], got %d (flag=%#02x)", len(raw), raw[0])
			}
			ksB64 = s
		}
	default:
		return "", fmt.Errorf("unsupported key type %T (need string bech32 or base64 keystore)", key)
	}

	// Load account from keystore
	acc, err := suiaccount.NewAccountWithKeystore(ksB64)
	if err != nil {
		return "", fmt.Errorf("load account from keystore: %w", err)
	}

	// Intent: TransactionData scope
	intent := suitypes.Intent{
		Scope: suitypes.IntentScope{
			TransactionData: &lib.EmptyEnum{}, // user signature over tx data
		},
		Version: suitypes.IntentVersion{
			V0: &lib.EmptyEnum{}, // intent version = v0
		},
		AppId: suitypes.AppId{
			Sui: &lib.EmptyEnum{}, // app-id domain = Sui
		},
	}

	sig, err := acc.SignSecureWithoutEncode(txBytes, intent)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	// Extract base64 serialized signature string:
	// Signature JSON is a one-key object like {"Ed25519SuiSignature":"<b64>"}.
	b, _ := json.Marshal(sig)
	var s string
	if err := json.Unmarshal(b, &s); err == nil && s != "" {
		return s, nil
	}

	// var m map[string]string
	// if err := json.Unmarshal(b, &m); err != nil {
	// 	return "", fmt.Errorf("encode signature: %w", err)
	// }
	// for _, v := range m {
	// 	if v != "" {
	// 		return v, nil
	// 	}
	// }
	return "", errors.New("empty signature after marshal")
}
