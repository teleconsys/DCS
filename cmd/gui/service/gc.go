package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/internal/gcclient"
)

// gcOptions adapts a GC profile to gcclient.Options.
func gcOptions(p state.ActorProfile) gcclient.Options {
	return gcclient.Options{
		EndpointOverride: strings.TrimSpace(p.GCEndpoint),
		TokenOverride:    strings.TrimSpace(p.GCToken),
	}
}

// GCWhitelistAdd asks the GC HTTP API to add `member` to the whitelist.
// Requires the Ground Control profile (signing happens server-side).
// On success, returns a short message suitable for a UI dialog.
func GCWhitelistAdd(ctx context.Context, p state.ActorProfile, member string, out io.Writer) (string, error) {
	if p.Actor != state.ActorGC {
		return "", fmt.Errorf("whitelist add requires the Ground Control actor")
	}
	m := strings.TrimSpace(member)
	if m == "" {
		return "", fmt.Errorf("member address is empty")
	}
	cli, err := gcclient.NewFromEnv(gcOptions(p))
	if err != nil {
		return "", err
	}
	resp, err := cli.WhitelistAdd(ctx, gcclient.WhitelistMutateRequest{
		Member:      m,
		WhitelistID: p.WhitelistID,
	})
	if err != nil {
		return "", err
	}
	if resp.Already {
		fmt.Fprintf(out, "ℹ %s already in whitelist\n", strings.ToLower(m))
		return fmt.Sprintf("%s was already on the whitelist.", m), nil
	}
	fmt.Fprintf(out, "✅ %s added to whitelist\n", strings.ToLower(m))
	return fmt.Sprintf("%s added to the whitelist.", m), nil
}

// GCWhitelistRemove asks the GC HTTP API to remove `member` from the
// whitelist. Requires the Ground Control profile.
func GCWhitelistRemove(ctx context.Context, p state.ActorProfile, member string, out io.Writer) (string, error) {
	if p.Actor != state.ActorGC {
		return "", fmt.Errorf("whitelist remove requires the Ground Control actor")
	}
	m := strings.TrimSpace(member)
	if m == "" {
		return "", fmt.Errorf("member address is empty")
	}
	cli, err := gcclient.NewFromEnv(gcOptions(p))
	if err != nil {
		return "", err
	}
	resp, err := cli.WhitelistRemove(ctx, gcclient.WhitelistMutateRequest{
		Member:      m,
		WhitelistID: p.WhitelistID,
	})
	if err != nil {
		return "", err
	}
	if resp.NotPresent {
		fmt.Fprintf(out, "ℹ %s not in whitelist\n", strings.ToLower(m))
		return fmt.Sprintf("%s was not on the whitelist.", m), nil
	}
	fmt.Fprintf(out, "✅ %s removed from whitelist\n", strings.ToLower(m))
	return fmt.Sprintf("%s removed from the whitelist.", m), nil
}
