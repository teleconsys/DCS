package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	// env loader
	"github.com/teleconsys/DCS/internal/config"

	// sub-trees
	"github.com/teleconsys/DCS/cmd/app"
	"github.com/teleconsys/DCS/cmd/iota_sc"
	"github.com/teleconsys/DCS/cmd/ipfs"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "DCS",
	Short: "Decentralised Content Security CLI",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default $HOME/.DCS.yaml)")

	rootCmd.AddCommand(
		app.NewCmd(),
		ipfs.NewCmd(),
		iota_sc.NewCmd(),
	)
}

func initConfig() {
	config.LoadEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".DCS")
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
