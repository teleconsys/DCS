package ipfs

import "github.com/spf13/cobra"

func newLoadFileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "load-file <path>",
		Short: "Add a local file to IPFS and pin it",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Loading %s into IPFS …\n", args[0])
			// TODO: real implementation
		},
	}
}
