package ipfs

import "github.com/spf13/cobra"

func newCheckPinsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-pins",
		Short: "Verify that pins are still intact on the cluster",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println("Checking IPFS pins …")
			// TODO: real implementation
		},
	}
}
