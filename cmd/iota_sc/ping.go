package iota_sc

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/teleconsys/DCS/internal/rebased"
)

// NewCmd wires the whole `iota-sc` subtree.
func NewCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "iota-sc",
		Short: "Interact with IOTA Rebased smart contracts",
	}
	root.PersistentFlags().
		String("rpc", "https://api.testnet.iota.cafe", "JSON-RPC endpoint")
	_ = viper.BindPFlag("rebase.rpc", root.PersistentFlags().Lookup("rpc"))

	root.AddCommand(newPingCmd())
	// root.AddCommand(newCallCmd())   ← add more as you grow
	return root
}

// ---------------------------------------------------------------------

func newPingCmd() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Quick connectivity check against the IOTA node",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := rebased.Dial(viper.GetString("rebase.rpc"))
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			cp, err := cli.Ping(ctx)
			if err != nil {
				return err
			}
			cmd.Printf("✅ node is alive — latest checkpoint: %d\n", cp)
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Second, "RPC timeout")
	return cmd
}
