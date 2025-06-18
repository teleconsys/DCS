package app

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	appCmd := &cobra.Command{
		Use:   "app",
		Short: "Application-level operations",
	}

	// leaf commands
	appCmd.AddCommand(newStartCmd())
	return appCmd
}
