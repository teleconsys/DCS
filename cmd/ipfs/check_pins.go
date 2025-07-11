package ipfs

import (
	"context"
	"os"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/client/rpc"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/spf13/cobra"
)

func newCheckPinsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-pins",
		Short: "Verify that pins are still intact on the cluster",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println("Checking IPFS pins …")

			// Connect to local IPFS node
			api, err := rpc.NewLocalApi()
			if err != nil {
				cmd.PrintErrf("Failed to connect to IPFS node: %v\n", err)
				os.Exit(1)
			}

			ctx := context.Background()
			cmd.Println("After context")

			// Get list of pinned items and store it in a channel
			pinsChan := make(chan iface.Pin)
			errChan := make(chan error, 1)

			go func() {
				errChan <- api.Pin().Ls(ctx, pinsChan)
				// Remove the explicit close - the channel will be closed by the Ls method
			}()

			cmd.Println("After list pins")

			var totalPins int
			var intactPins int
			var brokenPins int

			// Check each pin
			for pin := range pinsChan {
				cmd.Println("Inside for loop")
				totalPins++
				// Extract pin information
				cidStr := pin.Path().String()
				// Create path from CID
				cid, err := path.NewPath(cidStr)
				if err != nil {
					cmd.PrintErrf("Invalid CID: %s - %v\n", cidStr, err)
					brokenPins++
					continue
				}
				// Check if the pin is still intact by verifying it exists
				_, pinned, err := api.Pin().IsPinned(ctx, cid)
				if err != nil {
					cmd.PrintErrf("Error checking pin %s: %v\n", cidStr, err)
					brokenPins++
					continue
				}
				if pinned {
					intactPins++
					cmd.Printf("✓ Pin intact: %s\n", cidStr)
				} else {
					brokenPins++
					cmd.Printf("✗ Pin broken: %s\n", cidStr)
				}
			}

			// Check for errors after processing pins
			if err := <-errChan; err != nil {
				cmd.PrintErrf("Failed to list pins: %v\n", err)
				os.Exit(1)
			}

			// Print summary
			cmd.Println("\n--- Pin Check Summary ---")
			cmd.Printf("Total pins: %d\n", totalPins)
			cmd.Printf("Intact pins: %d\n", intactPins)
			cmd.Printf("Broken pins: %d\n", brokenPins)

			if brokenPins > 0 {
				cmd.PrintErrf("\n⚠️  Found %d broken pins!\n", brokenPins)
				os.Exit(1)
			} else {
				cmd.Printf("\n✅ All %d pins are intact!\n", totalPins)
			}
		},
	}
}
