package whitelist

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RemoveParams struct {
	WhitelistID string
	Member      string
	PackageID   string
	GasID       string
	GasBudget   uint64
	IotaBin     string // default "iota"
}

func LoadRemoveParams(cmd *cobra.Command, args []string) (RemoveParams, error) {
	var p RemoveParams

	// whitelist id
	p.WhitelistID = viper.GetString("dcs.whitelist_id")
	if p.WhitelistID == "" {
		p.WhitelistID = os.Getenv("DCS_WHITELIST_ID")
	}
	if p.WhitelistID == "" {
		return p, fmt.Errorf("set DCS_WHITELIST_ID env var or pass --id (0x...)")
	}

	// member
	if mFlag, _ := cmd.Flags().GetString("member"); mFlag != "" {
		p.Member = mFlag
	} else if len(args) > 0 {
		p.Member = args[0]
	}
	if p.Member == "" {
		return p, fmt.Errorf("provide address as positional arg or --member (0x...)")
	}
	p.Member = strings.ToLower(p.Member)

	// package id
	p.PackageID = viper.GetString("dcs.package_id")
	if p.PackageID == "" {
		p.PackageID = os.Getenv("DCS_PACKAGE_ID")
	}
	if p.PackageID == "" {
		return p, fmt.Errorf("set DCS_PACKAGE_ID env var or --package-id (0x...)")
	}

	// gas id
	p.GasID = os.Getenv("WALLET_GAS_ID")
	if p.GasID == "" {
		return p, fmt.Errorf("set WALLET_GAS_ID env var or --gas (0x...)")
	}

	// gas budget
	if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			p.GasBudget = v
		}
	}
	if p.GasBudget == 0 {
		p.GasBudget = 10_000_000
	}

	// iota binary
	if iotaBin, _ := cmd.Flags().GetString("iota-bin"); iotaBin != "" {
		p.IotaBin = iotaBin
	} else {
		p.IotaBin = "iota"
	}
	return p, nil
}

// RemoveFromWhitelist uses HasAddress() to skip if address is absent; otherwise calls the SC.
func RemoveFromWhitelist(ctx context.Context, p RemoveParams) (out []byte, notPresent bool, err error) {
	found, err := HasAddress(ctx, HasParams{WhitelistID: p.WhitelistID, Member: p.Member, IotaBin: p.IotaBin})
	if err == nil && !found {
		return nil, true, nil // nothing to do
	}
	argv := []string{
		"client", "call",
		"--package", p.PackageID,
		"--module", "dcs",
		"--function", "remove_id_from_whitelist",
		"--args", p.Member, p.WhitelistID,
		"--gas", p.GasID,
		"--gas-budget", strconv.FormatUint(p.GasBudget, 10),
	}
	out, err = exec.CommandContext(ctx, p.IotaBin, argv...).CombinedOutput()
	return out, false, err
}
