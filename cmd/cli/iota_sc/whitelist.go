package iota_sc

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/teleconsys/DCS/internal/gcclient"
	"github.com/teleconsys/DCS/internal/whitelist"
)

func newWhitelistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "whitelist",
		Short:        "Whitelist utilities",
		SilenceUsage: true,
	}

	// Allow passing whitelist object id via flag or env (DCS_WHITELIST_ID)
	cmd.PersistentFlags().String("id", "", "Whitelist object ID (0x...)")
	_ = viper.BindPFlag("dcs.whitelist_id", cmd.PersistentFlags().Lookup("id"))

	cmd.AddCommand(
		newWhitelistHasCmd(),
		newWhitelistAddCmd(),
		newWhitelistRemoveCmd(),
	)
	return cmd
}

func newWhitelistHasCmd() *cobra.Command {
	var member string
	var printAddr bool
	var iotaBin string // kept only to preserve flag; unused in RPC path

	c := &cobra.Command{
		Use:   "has [ADDRESS]",
		Short: "Return true if ADDRESS/ID is in the whitelist (RPC)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load inputs (whitelist id, member, rpc url)
			p, err := whitelist.LoadHasAddressParams(cmd, args)
			if err != nil {
				return err
			}

			found, err := whitelist.HasAddress(cmd.Context(), p)
			if err != nil {
				return err
			}

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
	// Preserve old flag surface even if unused in RPC path:
	c.Flags().StringVar(&iotaBin, "iota-bin", "", "Path to iota binary (ignored; RPC is used)")
	return c
}

func newWhitelistAddCmd() *cobra.Command {
	var member string

	var gcEndpoint string
	var gcToken string

	c := &cobra.Command{
		Use:   "add [ADDRESS]",
		Short: "Add ADDRESS/ID to the whitelist (signed by GroundControl API)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := strings.TrimSpace(member)
			if m == "" && len(args) > 0 {
				m = strings.TrimSpace(args[0])
			}
			if m == "" {
				return fmt.Errorf("provide address as positional arg or --member (0x...)")
			}

			cli, err := gcclient.NewFromEnv(gcclient.Options{
				EndpointOverride: gcEndpoint,
				TokenOverride:    gcToken,
			})
			if err != nil {
				return err
			}

			resp, err := cli.WhitelistAdd(cmd.Context(), gcclient.WhitelistMutateRequest{
				Member:      m,
				WhitelistID: viper.GetString("dcs.whitelist_id"), // opzionale
			})
			if err != nil {
				return err
			}

			if resp.Already {
				cmd.Println("Address is already in the whitelist.")
				return nil
			}

			cmd.Printf("Address %s successfully added to the whitelist.\n", strings.ToLower(m))
			return nil
		},
	}

	c.Flags().StringVarP(&member, "member", "m", "", "Address/ID to add (0x...)")
	c.Flags().StringVar(&gcEndpoint, "gc-endpoint", "", "GroundControl API base URL (overrides GC_ENDPOINT env)")
	c.Flags().StringVar(&gcToken, "gc-token", "", "GroundControl API token (overrides GC_API_TOKEN env)")

	return c
}

func newWhitelistRemoveCmd() *cobra.Command {
	var member string

	var gcEndpoint string
	var gcToken string

	c := &cobra.Command{
		Use:   "remove [ADDRESS]",
		Short: "Remove ADDRESS/ID from the whitelist (signed by GroundControl API)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := strings.TrimSpace(member)
			if m == "" && len(args) > 0 {
				m = strings.TrimSpace(args[0])
			}
			if m == "" {
				return fmt.Errorf("provide address as positional arg or --member (0x...)")
			}

			cli, err := gcclient.NewFromEnv(gcclient.Options{
				EndpointOverride: gcEndpoint,
				TokenOverride:    gcToken,
			})
			if err != nil {
				return err
			}

			resp, err := cli.WhitelistRemove(cmd.Context(), gcclient.WhitelistMutateRequest{
				Member:      m,
				WhitelistID: viper.GetString("dcs.whitelist_id"),
			})
			if err != nil {
				return err
			}

			if resp.NotPresent {
				cmd.Println("Address is not in the whitelist.")
				return nil
			}

			cmd.Printf("Address %s successfully removed from the whitelist.\n", strings.ToLower(m))
			return nil
		},
	}

	c.Flags().StringVarP(&member, "member", "m", "", "Address/ID to remove (0x...)")
	c.Flags().StringVar(&gcEndpoint, "gc-endpoint", "", "GroundControl API base URL (overrides GC_ENDPOINT env)")
	c.Flags().StringVar(&gcToken, "gc-token", "", "GroundControl API token (overrides GC_API_TOKEN env)")

	return c
}
