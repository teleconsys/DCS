package rebased_test

import (
	"context"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/teleconsys/DCS/internal/rebased"
)

const rpcURL = "https://api.testnet.iota.cafe:443"

func TestPingIotaRebased(t *testing.T) {
	if os.Getenv("DCS_LIVE") == "" {
		t.Skip("set DCS_LIVE=1 to run this live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	cli, err := rebased.Dial(rpcURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	// 1️⃣  Endpoint identity
	ok, err := cli.IsIota(ctx)
	if err != nil {
		t.Fatalf("IsIota: %v", err)
	}
	if !ok {
		t.Fatal("endpoint responded but is NOT IOTA Rebased")
	}

	// 2️⃣  Chain identifier should be an 8-char hex string
	chain, err := cli.ChainIdentifier(ctx)
	if err != nil {
		t.Fatalf("ChainIdentifier: %v", err)
	}
	if len(chain) != 8 {
		t.Fatalf("unexpected chain identifier format: %q", chain)
	}
	if _, err := hex.DecodeString(chain); err != nil {
		t.Fatalf("unexpected chain identifier format: %q", chain)
	}

	// 3️⃣  Basic liveness
	cp, err := cli.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if cp == 0 {
		t.Fatal("checkpoint number is zero – node unhealthy?")
	}

	t.Logf("PASS – IOTA chain=%s, checkpoint=%d", chain, cp)
}
