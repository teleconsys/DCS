package iota_sc

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmd() *cobra.Command {
	iota_scCmd := &cobra.Command{
		Use:   "iota_sc",
		Short: "IOTA Smart Contract-related utilities",
	}

	iota_scCmd.PersistentFlags().
		String("rpc", "https://api.testnet.iota.cafe:443",
			"JSON-RPC endpoint (testnet default)")
	_ = viper.BindPFlag("rebase.rpc", iota_scCmd.PersistentFlags().Lookup("rpc"))

	// leaf commands
	iota_scCmd.AddCommand(
		newPingCmd(),
		newAccountCmd(),
		newWhitelistCmd(),
		cidCmd(),
		epochCmd(),
	)
	return iota_scCmd
}
