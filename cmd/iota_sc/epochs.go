package iota_sc

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/teleconsys/DCS/internal/epochs"
)

func epochCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "epoch",
		Short: "Epoch utilities",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		epochTransitionCmd(),
	)
	return cmd
}

func epochTransitionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next",
		Short: "Transition to the next epoch",
		RunE: func(cmd *cobra.Command, args []string) error {
			
			if len(args) == 0 {
				cmd.PrintErrf("Error: provide a CID as argument\n")
				return cmd.Usage()
			}

			if len(args) > 1 {
				cmd.PrintErrf("Error: provide only one CID as argument\n")
				return cmd.Usage()
			}

			// Load parameters using the new wrapper
			params, err := epochs.LoadTransitionParams(cmd, args)
			if err != nil {
				cmd.PrintErrf("Failed to load parameters: %v\n", err)
				return err
			}

			result, err := epochs.TransitionEpoch(cmd.Context(), params, args[0])

			if err != nil {
				cmd.PrintErrf("Failed to transition epoch: %v\n", err)
				return err
			}

			fmt.Printf("Epoch transition result: %v\n", result)
			return nil
		},
	}
	
	return cmd
}