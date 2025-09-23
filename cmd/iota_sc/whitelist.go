package iota_sc

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/teleconsys/DCS/internal/whitelist"
)

func newWhitelistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "whitelist",
		Short:        "Whitelist utilities",
		SilenceUsage: true,
	}
	// Allow passing the whitelist object id via flag or env (DCS_WHITELIST_ID)
	cmd.PersistentFlags().String("id", "", "Whitelist object ID (0x...)")
	_ = viper.BindPFlag("dcs.whitelist_id", cmd.PersistentFlags().Lookup("id"))

	cmd.AddCommand(
		newWhitelistHasAddressCmd(),
		newWhitelistAddCmd(),
		newWhitelistRemoveCmd(),
	)
	return cmd
}

func newWhitelistHasAddressCmd() *cobra.Command {
	var member string
	var printAddr bool
	var iotaBin string

	c := &cobra.Command{
		Use:   "has [ADDRESS]",
		Short: "Return true if ADDRESS/ID is in the whitelist",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1) read inputs
			p, err := whitelist.LoadHasAddressParams(cmd, args)
			if err != nil {
				return err
			}

			// 2) call helper
			found, err := whitelist.HasAddress(cmd.Context(), p)
			if err != nil {
				return err
			}

			// 3) print according to flag
			if printAddr {
				if found {
					fmt.Println(p.Member)
				}
				return nil
			}
			if found {
				fmt.Println("true")
			} else {
				fmt.Println("false")
			}
			return nil
		},
	}

	c.Flags().StringVarP(&member, "member", "m", "", "Address/ID to check (0x...)")
	c.Flags().BoolVar(&printAddr, "print-addr", false, "Print the address if present (instead of true/false)")
	c.Flags().StringVar(&iotaBin, "iota-bin", "", "Path to iota binary (default: iota)")
	return c
}

// add <ADDRESS> to whitelist by calling the on-chain function (GC must be active signer).
func newWhitelistAddCmd() *cobra.Command {
	var member string
	var pkgID string
	var gasID string
	var gasBudget uint64
	var iotaBin string

	c := &cobra.Command{
		Use:   "add [ADDRESS]",
		Short: "Add ADDRESS/ID to the whitelist (requires GroundControl signer)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1) load params
			p, err := whitelist.LoadAddParams(cmd, args)
			if err != nil {
				return err
			}

			// 2) perform the operation
			out, already, err := whitelist.AddToWhitelist(cmd.Context(), p)
			if already {
				cmd.Println("Address is already in the whitelist.")
				return nil
			}

			cmd.Print(string(out)) // TODO: Clean raw output
			if err != nil {
				return fmt.Errorf("iota client call failed: %w", err)
			}
			return nil
		},
	}

	c.Flags().StringVarP(&member, "member", "m", "", "Address/ID to add (0x...)")
	c.Flags().StringVar(&pkgID, "package-id", "", "DCS package ID (0x...)")
	c.Flags().StringVar(&gasID, "gas", "", "Gas coin object ID (0x...)")
	c.Flags().Uint64Var(&gasBudget, "gas-budget", 0, "Gas budget (nanos)")
	c.Flags().StringVar(&iotaBin, "iota-bin", "", "Path to iota binary (default: iota)")
	return c
}

func newWhitelistRemoveCmd() *cobra.Command {
	var member string
	var pkgID string
	var gasID string
	var gasBudget uint64
	var iotaBin string

	c := &cobra.Command{
		Use:   "remove [ADDRESS]",
		Short: "Remove ADDRESS/ID from the whitelist (requires GroundControl signer)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1) load inputs
			p, err := whitelist.LoadRemoveParams(cmd, args)
			if err != nil {
				return err
			}

			// 2) do the operation
			out, notPresent, err := whitelist.RemoveFromWhitelist(cmd.Context(), p)
			if notPresent {
				cmd.Println("Address is not in the whitelist.")
				return nil
			}

			cmd.Print(string(out)) // TODO: Clean raw output
			if err != nil {
				return fmt.Errorf("iota client call failed: %w", err)
			}
			return nil
		},
	}

	// Flags (optional if env is set)
	c.Flags().StringVarP(&member, "member", "m", "", "Address/ID to remove (0x...)")
	c.Flags().StringVar(&pkgID, "package-id", "", "DCS package ID (0x...)")
	c.Flags().StringVar(&gasID, "gas", "", "Gas coin object ID (0x...)")
	c.Flags().Uint64Var(&gasBudget, "gas-budget", 0, "Gas budget (nanos)")
	c.Flags().StringVar(&iotaBin, "iota-bin", "", "Path to iota binary (default: iota)")
	return c
}
