package ipfsutil

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/client/rpc"
	iface "github.com/ipfs/kubo/core/coreiface"
)

const DefaultUnpinnedCID = "QmWtM9FSHL8pvXVGT9dGumMSFLoGZJqRNsikSB9mWq5KgZ"

func GetPinnedCID() (string, error) {
	api, err := rpc.NewLocalApi()
	if err != nil {
		return "", fmt.Errorf("connect to ipfs: %w", err)
	}

	ctx := context.Background()
	pinsChan := make(chan iface.Pin)
	errChan := make(chan error, 1)

	go func() {
		errChan <- api.Pin().Ls(ctx, pinsChan)
	}()

	var pinnedCIDs []string
	for pin := range pinsChan {
		cidStr := pin.Path().String()

		cid, err := path.NewPath(cidStr)
		if err != nil {
			continue
		}

		_, pinned, err := api.Pin().IsPinned(ctx, cid)
		if err != nil || !pinned {
			continue
		}

		cidStr = strings.TrimPrefix(cidStr, "/ipfs/")
		pinnedCIDs = append(pinnedCIDs, cidStr)
	}

	if err := <-errChan; err != nil {
		return "", fmt.Errorf("list pins: %w", err)
	}
	if len(pinnedCIDs) == 0 {
		return "", fmt.Errorf("no pinned cids found")
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return pinnedCIDs[r.Intn(len(pinnedCIDs))], nil
}

// GetUnpinnedCID returns a CID that is not pinned
func GetUnpinnedCID() (string, error) {
	validCid, err := GetPinnedCID()
	if err != nil {
		return DefaultUnpinnedCID, nil
	}

	api, err := rpc.NewLocalApi()
	if err != nil {
		return "", fmt.Errorf("connect to ipfs: %w", err)
	}

	ctx := context.Background()
	candidate := validCid

	for {
		if len(candidate) == 0 {
			return DefaultUnpinnedCID, nil
		}

		last := candidate[len(candidate)-1]
		candidate = candidate[:len(candidate)-1] + string(last+1)

		p, err := path.NewPath("/ipfs/" + candidate)
		if err != nil {
			continue
		}

		_, pinned, err := api.Pin().IsPinned(ctx, p)
		if err != nil {
			continue
		}
		if !pinned {
			return candidate, nil
		}
	}
}

// LoadFileToIPFS adds a file to IPFS and pins it on the local node.
func LoadFileToIPFS(filePath string) (string, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	api, err := rpc.NewLocalApi()
	if err != nil {
		return "", fmt.Errorf("connect to ipfs: %w", err)
	}

	ctx := context.Background()
	node := files.NewReaderFile(bytes.NewReader(b))

	ipfsPath, err := api.Unixfs().Add(ctx, node)
	if err != nil {
		return "", fmt.Errorf("ipfs add: %w", err)
	}

	if err := api.Pin().Add(ctx, ipfsPath); err != nil {
		return "", fmt.Errorf("ipfs pin: %w", err)
	}

	return ipfsPath.String(), nil
}

func ValidateCID(cid string) error {
	cid = strings.TrimSpace(cid)
	if cid == "" {
		return fmt.Errorf("cid is empty")
	}
	if !strings.HasPrefix(cid, "Qm") {
		return fmt.Errorf("expected CIDv0 (Qm...), got: %s", cid)
	}
	return nil
}
