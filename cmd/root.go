package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	// sub-trees
	"github.com/teleconsys/DCS/cmd/app"
	"github.com/teleconsys/DCS/cmd/ipfs"
)

var cfgFile string

// rootCmd is the CLI entry-point.
var rootCmd = &cobra.Command{
	Use:   "DCS",
	Short: "Decentralised Content Security CLI",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// Execute is called by main().
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default $HOME/.DCS.yaml)")

	// register sub-command trees
	rootCmd.AddCommand(
		app.NewCmd(),
		ipfs.NewCmd(),
	)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".DCS")
	}

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
