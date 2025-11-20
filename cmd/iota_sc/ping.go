package iota_sc

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/teleconsys/DCS/internal/rebased"
)

func newPingCmd() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Quick connectivity check against the IOTA network",
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
