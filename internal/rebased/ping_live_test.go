// internal/rebased/wrapper_live_test.go
package rebased_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/teleconsys/DCS/internal/rebased"
)

const defaultRPC = "https://api.testnet.iota.cafe:443"

func TestWrapper_Live(t *testing.T) {
	if os.Getenv("DCS_LIVE") == "" {
		t.Skip("set DCS_LIVE=1 to run this live test")
	}

	rpcURL := os.Getenv("REBASE_RPC")
	if rpcURL == "" {
		rpcURL = defaultRPC
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1) Dial
	w, err := rebased.Dial(rpcURL)
	if err != nil {
		t.Fatalf("Dial(%s): %v", rpcURL, err)
	}

	// 2) Liveness
	cp, err := w.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if cp == 0 {
		t.Fatalf("Ping checkpoint is zero – node unhealthy?")
	}
	t.Logf("Ping OK – latest checkpoint: %d", cp)

	// 3) Optional read: whitelist object by ID (no signing needed)
	if wl := os.Getenv("DCS_WHITELIST_ID"); wl != "" {
		obj, err := w.GetObject(ctx, wl, suitypes.SuiObjectDataOptions{ShowContent: true})
		if err != nil {
			t.Fatalf("GetObject(%s): %v", wl, err)
		}
		if obj == nil || obj.Data == nil {
			t.Fatalf("GetObject(%s) returned no Data", wl)
		}
		// Just log a tiny preview of the returned object
		b, _ := json.MarshalIndent(obj.Data, "", "  ")
		if len(b) > 512 {
			b = b[:512]
		}
		t.Logf("GetObject OK for %s: %s...", wl, string(b))
	} else {
		t.Log("DCS_WHITELIST_ID not set; skipping GetObject smoke test")
	}
}
