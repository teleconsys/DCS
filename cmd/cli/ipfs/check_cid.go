package ipfs

import (
	"context"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/client/rpc"
	"github.com/spf13/cobra"
)

func newCheckCidCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-cid <cid>",
		Short: "Check if a specific CID exists in IPFS",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Add /ipfs prefix to the CID
			cidPath := "/ipfs/" + args[0]
			cmd.Printf("Checking if CID %s exists in IPFS...\n", cidPath)

			// Connect to local IPFS node
			api, err := rpc.NewLocalApi()
			if err != nil {
				cmd.PrintErrf("Failed to connect to IPFS node: %v\n", err)
				return err
			}

			// Create a context without timeout
			ctx := context.Background()

			// Parse the CID
			cid, err := path.NewPath(cidPath)
			if err != nil {
				return err
			}

			// Check if the CID is pinned using Pin().Ls() - more efficient than IsPinned
			_, pinned, err := api.Pin().IsPinned(ctx, cid)	
			if err != nil {
				cmd.PrintErrf("Error while checking pin %v: %v\n", cid, err)
				return err
			}

			if pinned {
				cmd.Printf("📌 CID is pinned\n")
			} else {
				cmd.Printf("📌 CID is NOT pinned\n")
			}
			
			return nil
		},
	}
}


