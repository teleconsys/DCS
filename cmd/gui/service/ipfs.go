package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/client/rpc"
	iface "github.com/ipfs/kubo/core/coreiface"

	"github.com/teleconsys/DCS/internal/ipfsutil"
)

// LoadFile uploads a local file to the running IPFS node and returns the
// resulting CID (without the "/ipfs/" prefix).
func LoadFile(_ context.Context, filePath string, out io.Writer) (cidStr string, err error) {
	fp := strings.TrimSpace(filePath)
	if fp == "" {
		return "", fmt.Errorf("file path is empty")
	}
	info, err := os.Stat(fp)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", fp, err)
	}
	fmt.Fprintf(out, "Loading %s (%d bytes) into IPFS…\n", fp, info.Size())

	ipfsPath, err := ipfsutil.LoadFileToIPFS(fp)
	if err != nil {
		return "", fmt.Errorf("load file: %w", err)
	}
	cidStr = strings.TrimPrefix(ipfsPath, "/ipfs/")
	fmt.Fprintf(out, "✅ uploaded: %s\n", ipfsPath)
	return cidStr, nil
}

// CheckCid reports whether a CID is pinned on the local IPFS node.
func CheckCid(ctx context.Context, cidStr string, out io.Writer) (bool, error) {
	cidStr = strings.TrimSpace(cidStr)
	if cidStr == "" {
		return false, fmt.Errorf("CID is empty")
	}
	if !strings.HasPrefix(cidStr, "/ipfs/") {
		cidStr = "/ipfs/" + cidStr
	}
	fmt.Fprintf(out, "Checking %s…\n", cidStr)

	api, err := rpc.NewLocalApi()
	if err != nil {
		return false, fmt.Errorf("connect to local IPFS: %w", err)
	}
	p, err := path.NewPath(cidStr)
	if err != nil {
		return false, fmt.Errorf("parse path: %w", err)
	}
	_, pinned, err := api.Pin().IsPinned(ctx, p)
	if err != nil {
		return false, fmt.Errorf("is-pinned: %w", err)
	}
	if pinned {
		fmt.Fprintf(out, "📌 CID is pinned\n")
	} else {
		fmt.Fprintf(out, "📌 CID is NOT pinned\n")
	}
	return pinned, nil
}

// PinSummary aggregates the results of CheckPins.
type PinSummary struct {
	Total  int
	Intact int
	Broken int
}

// CheckPins iterates every pin on the local node and reports per-pin
// status. The summary is also printed to `out` for convenience.
func CheckPins(ctx context.Context, out io.Writer) (PinSummary, error) {
	var sum PinSummary

	api, err := rpc.NewLocalApi()
	if err != nil {
		return sum, fmt.Errorf("connect to local IPFS: %w", err)
	}

	pinsChan := make(chan iface.Pin)
	errChan := make(chan error, 1)
	go func() { errChan <- api.Pin().Ls(ctx, pinsChan) }()

	for pin := range pinsChan {
		sum.Total++
		cidStr := pin.Path().String()
		p, err := path.NewPath(cidStr)
		short := strings.TrimPrefix(cidStr, "/ipfs/")
		if err != nil {
			fmt.Fprintf(out, "✗ invalid CID %s: %v\n", short, err)
			sum.Broken++
			continue
		}
		_, pinned, err := api.Pin().IsPinned(ctx, p)
		if err != nil {
			fmt.Fprintf(out, "✗ error checking %s: %v\n", short, err)
			sum.Broken++
			continue
		}
		if pinned {
			sum.Intact++
			fmt.Fprintf(out, "✓ %s\n", short)
		} else {
			sum.Broken++
			fmt.Fprintf(out, "✗ %s (not pinned)\n", short)
		}
	}

	if err := <-errChan; err != nil {
		return sum, fmt.Errorf("list pins: %w", err)
	}

	fmt.Fprintf(out, "\n— summary —\ntotal: %d  intact: %d  broken: %d\n",
		sum.Total, sum.Intact, sum.Broken)
	if sum.Broken > 0 {
		return sum, fmt.Errorf("%d broken pins", sum.Broken)
	}
	return sum, nil
}
