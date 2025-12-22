package app

import "github.com/spf13/cobra"

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the DCS application locally",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println("DCS app starting …")
			// TODO: real start-up logic
		},
	}
}
