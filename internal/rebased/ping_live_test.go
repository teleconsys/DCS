package rebased_test

import (
	"context"
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

	// Endpoint identity
	ok, err := cli.IsIota(ctx)
	if err != nil {
		t.Fatalf("IsIota: %v", err)
	}
	if !ok {
		t.Fatal("endpoint responded but is NOT IOTA Rebased")
	}

	// Basic liveness
	cp, err := cli.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if cp == 0 {
		t.Fatal("checkpoint number is zero – node unhealthy?")
	}

	t.Logf("PASS – IOTA Rebased testnet checkpoint=%d", cp)
}
