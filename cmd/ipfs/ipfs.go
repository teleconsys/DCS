package ipfs

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	ipfsCmd := &cobra.Command{
		Use:   "ipfs",
		Short: "IPFS-related utilities",
	}

	// leaf commands
	ipfsCmd.AddCommand(
		newCheckPinsCmd(),
		newLoadFileCmd(),
	)
	return ipfsCmd
}
