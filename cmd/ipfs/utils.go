package ipfs

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/client/rpc"
	iface "github.com/ipfs/kubo/core/coreiface"
)
const (
	DEFAULT_UNPINNED_CID = "QmWtM9FSHL8pvXVGT9dGumMSFLoGZJqRNsikSB9mWq5KgZ"
)
// GetPinnedCID returns a randomly selected pinned CID from the IPFS node
func GetPinnedCID() (string, error) {
	// Connect to local IPFS node
	api, err := rpc.NewLocalApi()
	if err != nil {
		return "", fmt.Errorf("failed to connect to IPFS node: %v", err)
	}

	ctx := context.Background()

	// Get list of pinned items
	pinsChan := make(chan iface.Pin)
	errChan := make(chan error, 1)

	go func() {
		errChan <- api.Pin().Ls(ctx, pinsChan)
	}()

	var pinnedCIDs []string

	// Collect all pinned CIDs
	for pin := range pinsChan {
		cidStr := pin.Path().String()
		
		// Verify the pin is still intact
		cid, err := path.NewPath(cidStr)
		if err != nil {
			continue // Skip invalid CIDs
		}
		
		_, pinned, err := api.Pin().IsPinned(ctx, cid)
		if err != nil {
			continue // Skip broken pins
		}
		if pinned {
			// Extract only the CID part, removing /ipfs/ prefix if present
			cidStr = strings.TrimPrefix(cidStr, "/ipfs/")
			pinnedCIDs = append(pinnedCIDs, cidStr)
		}
	}

	// Check for errors after processing pins
	if err := <-errChan; err != nil {
		return "", fmt.Errorf("failed to list pins: %v", err)
	}

	if len(pinnedCIDs) == 0 {
		return "", fmt.Errorf("no pinned CIDs found")
	}

	// Return a random pinned CID
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(pinnedCIDs))
	return pinnedCIDs[randomIndex], nil
}

func GetUnpinnedCID() (string, error) {
	validCid, err := GetPinnedCID()
	
	if err != nil {
		return DEFAULT_UNPINNED_CID, nil
	}

	api, err := rpc.NewLocalApi()
	if err != nil {
		return "", fmt.Errorf("failed to connect to IPFS node: %v", err)
	}

	ctx := context.Background()
	unvalidCid := validCid
	for {
		unvalidCid = unvalidCid[:len(unvalidCid)-1] + string(unvalidCid[len(unvalidCid)-1]+1)

		cid, err := path.NewPath("/ipfs/"+unvalidCid)
		if err != nil {
			continue // Skip invalid CIDs
		}
		
		_, pinned, err := api.Pin().IsPinned(ctx, cid)
		if err != nil {
			continue // Skip broken pins
		}
		
		if !pinned {
			break
		}
	}

	return unvalidCid, nil
}

// ValidateCID checks if a CID is valid and has the correct format, used in the tests
func ValidateCID(cid string) error {
	if cid == "" {
		return fmt.Errorf("CID cannot be empty")
	}
	
	// CID should start with Qm (IPFS CIDv0 format)
	if !strings.HasPrefix(cid, "Qm") {
		return fmt.Errorf("CID should start with 'Qm', got: %s", cid)
	}

	// TODO: Add more validation for CIDv1 format, 
	// see https://github.com/multiformats/cid/blob/ef1b2002394b15b1e6c26c30545fd485f2c4c138/README.md#decoding-algorithm
	// see https://github.com/ipfs/go-cid/blob/master/cid.go#L100
	
	return nil
}