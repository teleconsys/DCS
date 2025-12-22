package iota_sc

import (
	"context"
	"fmt"
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

			// Get checkpoint details to retrieve timestamp
			checkpoint, err := cli.GetCheckpoint(ctx, cp)
			if err != nil {
				return err
			}

			// Parse timestampMs (milliseconds) to time
			var timestampMs int64
			if _, err := fmt.Sscanf(checkpoint.TimestampMs, "%d", &timestampMs); err != nil {
				return fmt.Errorf("failed to parse timestamp: %w", err)
			}

			checkpointTime := time.UnixMilli(timestampMs)
			dateStr := checkpointTime.Format("2006-01-02 15:04:05")

			cmd.Printf("✅ node is alive — latest checkpoint: %d (%s)\n", cp, dateStr)
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Second, "RPC timeout")
	return cmd
}
