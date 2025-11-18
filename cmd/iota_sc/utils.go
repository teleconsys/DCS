package iota_sc

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/teleconsys/DCS/internal/rebased"
)

func debugTxStatus(resp *suitypes.SuiTransactionBlockResponse) error {
	b, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println("== tx effects/events (debug) ==")
	fmt.Println(string(b))
	return nil
}

func debugCIDWindow(ctx context.Context, rpc, cid string) error {
	w, err := rebased.Dial(rpc)
	if err != nil {
		return err
	}
	o, err := w.GetObject(ctx, cid, suitypes.SuiObjectDataOptions{ShowContent: true, ShowType: true})
	if err != nil {
		return err
	}
	b, _ := json.Marshal(o)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	fields := findFields(m)
	if fields == nil {
		return fmt.Errorf("cannot read CID fields")
	}
	nextStart := toInt64(fields["next_epoch_start"])
	nextEnd := toInt64(fields["next_epoch_end"])
	curEnd := toInt64(fields["current_epoch_end"])

	now := time.Now().UnixMilli()
	windowClose := nextStart + 600_000 // start + 10m

	fmt.Println("== epoch window (debug) ==")
	fmt.Printf("now_ms=%d  current_end=%d  next_start=%d  next_end=%d  window_closes=%d\n",
		now, curEnd, nextStart, nextEnd, windowClose)
	inWindow := (now >= curEnd) && (now < windowClose)
	fmt.Printf("in_window=%v  (now >= current_end && now < next_start+10m)\n", inWindow)
	return nil
}

func debugCIDOfferLens(ctx context.Context, rpc, cid, label string) error {
	w, err := rebased.Dial(rpc)
	if err != nil {
		return err
	}
	o, err := w.GetObject(ctx, cid, suitypes.SuiObjectDataOptions{ShowContent: true})
	if err != nil {
		return err
	}
	b, _ := json.Marshal(o)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	fields := findFields(m)
	if fields == nil {
		return fmt.Errorf("cannot read CID fields")
	}
	nextOffers := asSlice(fields["next_epoch_offers"])
	curOffers := asSlice(fields["current_epoch_offers"])
	prevOffers := asSlice(fields["prev_epoch_offers"])

	fmt.Printf("== offers %s ==\n", label)
	fmt.Printf("next=%d  current=%d  prev=%d\n", len(nextOffers), len(curOffers), len(prevOffers))
	return nil
}

func debugShowOffer(ctx context.Context, rpc, cid string, idx uint64) error {
	w, err := rebased.Dial(rpc)
	if err != nil {
		return err
	}
	o, err := w.GetObject(ctx, cid, suitypes.SuiObjectDataOptions{ShowContent: true})
	if err != nil {
		return err
	}
	b, _ := json.Marshal(o)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	fields := findFields(m)
	if fields == nil {
		return fmt.Errorf("cannot read CID fields")
	}
	nextOffers := asSlice(fields["next_epoch_offers"])
	if int(idx) >= len(nextOffers) {
		return fmt.Errorf("idx out of range: %d >= %d", idx, len(nextOffers))
	}
	ob, _ := json.MarshalIndent(nextOffers[idx], "", "  ")
	fmt.Printf("offer[%d] (next): %s\n", idx, string(ob))
	return nil
}

// ---------- generic JSON helpers ----------

func findFields(m map[string]any) map[string]any {
	c := findByKey(m, "content")
	if c == nil {
		return nil
	}
	if cm, ok := c.(map[string]any); ok {
		if f, ok := cm["fields"].(map[string]any); ok {
			return f
		}
	}
	f := findByKey(c, "fields")
	if fm, ok := f.(map[string]any); ok {
		return fm
	}
	return nil
}

func findByKey(v any, want string) any {
	switch x := v.(type) {
	case map[string]any:
		if val, ok := x[want]; ok {
			return val
		}
		for _, val := range x {
			if res := findByKey(val, want); res != nil {
				return res
			}
		}
	case []any:
		for _, it := range x {
			if res := findByKey(it, want); res != nil {
				return res
			}
		}
	}
	return nil
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case float64:
		return int64(t)
	case string:
		i, _ := strconv.ParseInt(t, 10, 64)
		return i
	default:
		return 0
	}
}

func asSlice(v any) []any {
	if v == nil {
		return nil
	}
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

// ---------- env helpers ----------
func getEnvAsUint64(key string, def uint64) uint64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return def
	}
	return i
}
func must(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing required env: " + k)
	}
	return v
}
func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// ResolvePrivateKey resolves the private key to use for signing.
//
// Precedence:
//  1. If flagValue is non-empty, it is validated and returned.
//  2. Otherwise, the user is prompted on stdin to enter the key.
//
// The key must be either:
//   - a bech32 string starting with iotaprivkey1... or suiprivkey1..., or
//   - a base64-encoded 33-byte [0x00 | 32-byte ed25519 secret] keystore.
func ResolvePrivateKey(flagValue string) (string, error) {
	if s := strings.TrimSpace(flagValue); s != "" {
		if err := validatePrivateKeyFormat(s); err != nil {
			return "", err
		}
		return s, nil
	}

	// Interactive prompt
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
