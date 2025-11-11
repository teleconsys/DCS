package iota_sc

import (
	"context"
	"encoding/json"
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
