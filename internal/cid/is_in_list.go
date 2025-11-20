package cid

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type IsInListParams struct {
	CIDId     string
	CIDListID string
	RPCURL    string
}

func LoadIsInListParams(cmd *cobra.Command, args []string) (IsInListParams, error) {
	var p IsInListParams

	if len(args) == 0 {
		return p, fmt.Errorf("provide CID ID or CID string as argument")
	}

	cidType, err := cmd.Flags().GetString("cid-type")
	if err != nil {
		return p, err
	}

	// Validate cidType
	if cidType != "id" && cidType != "cid" {
		return p, fmt.Errorf("cid-type must be either 'id' or 'cid', got: %s", cidType)
	}

	// Set the cidId based on type
	if cidType == "id" {
		p.CIDId = args[0]
	} else {
		cidStr := args[0]

		// Get RPC URL first (needed for GetCIDIdFromList)
		p.RPCURL = viper.GetString("rpc")
		if p.RPCURL == "" {
			if u := os.Getenv("REBASE_RPC"); u != "" {
				p.RPCURL = u
			} else if u := os.Getenv("DCS_RPC"); u != "" {
				p.RPCURL = u
			} else {
				p.RPCURL = "https://api.testnet.iota.cafe:443"
			}
		}

		// Get the CID object ID from the CIDlist
		cidId, err := GetCIDIdFromList(cmd.Context(), cidStr, p.RPCURL)
		if err != nil {
			return p, fmt.Errorf("failed to find CID object: %w", err)
		}
		p.CIDId = cidId
	}

	// Get CID list ID
	p.CIDListID = viper.GetString("dcs.cidlist_id")
	if p.CIDListID == "" {
		p.CIDListID = os.Getenv("DCS_CIDLIST_ID")
	}
	if p.CIDListID == "" {
		return p, fmt.Errorf("set DCS_CIDLIST_ID env var or pass --cidlist-id (0x...)")
	}

	// Get RPC URL if not already set
	if p.RPCURL == "" {
		p.RPCURL = viper.GetString("rpc")
		if p.RPCURL == "" {
			if u := os.Getenv("REBASE_RPC"); u != "" {
				p.RPCURL = u
			} else if u := os.Getenv("DCS_RPC"); u != "" {
				p.RPCURL = u
			} else {
				p.RPCURL = "https://api.testnet.iota.cafe:443"
			}
		}
	}

	return p, nil
}

func IsInList(ctx context.Context, p IsInListParams) (bool, error) {
	// Get the CID list
	cidList, err := GetCIDList(ctx, p.RPCURL)
	if err != nil {
		return false, fmt.Errorf("failed to get CID list: %w", err)
	}

	// Check if the CID ID is in the list
	for _, cid := range cidList {
		if cid == p.CIDId {
			return true, nil
		}
	}

	return false, nil
}
