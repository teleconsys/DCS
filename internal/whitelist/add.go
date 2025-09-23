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

type AddParams struct {
	WhitelistID string
	Member      string // 0x..
	PackageID   string
	GasID       string
	GasBudget   uint64
	IotaBin     string // "iota" by default
}

// LoadAddParams reads flags/env/positional args and returns a filled AddParams.
func LoadAddParams(cmd *cobra.Command, args []string) (AddParams, error) {
	var p AddParams

	// whitelist id
	p.WhitelistID = viper.GetString("dcs.whitelist_id")
	if p.WhitelistID == "" {
		p.WhitelistID = os.Getenv("DCS_WHITELIST_ID")
	}
	if p.WhitelistID == "" {
		return p, fmt.Errorf("set DCS_WHITELIST_ID env var or pass --id (0x...)")
	}

	// member: flag takes priority, otherwise positional
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

	// gas budget (env or default)
	if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			p.GasBudget = v
		}
	}
	if p.GasBudget == 0 {
		p.GasBudget = 10_000_000
	}

	// iota binary (flag optional)
	if iotaBin, _ := cmd.Flags().GetString("iota-bin"); iotaBin != "" {
		p.IotaBin = iotaBin
	} else {
		p.IotaBin = "iota"
	}

	return p, nil
}

// AddToWhitelist checks presence of the target address in whitelist; if already present, returns (nil, true, nil).
// Otherwise it calls the smart contract and returns its RAW output.
func AddToWhitelist(ctx context.Context, p AddParams) (out []byte, already bool, err error) {

	found, err := HasAddress(ctx, HasParams{WhitelistID: p.WhitelistID, Member: p.Member, IotaBin: p.IotaBin})
	if err == nil && found {
		return nil, true, nil
	}

	// Build args and call the function (unchanged)
	argsv := []string{
		"client", "call",
		"--package", p.PackageID,
		"--module", "dcs",
		"--function", "add_id_to_whitelist",
		"--args", p.Member, p.WhitelistID,
		"--gas", p.GasID,
		"--gas-budget", strconv.FormatUint(p.GasBudget, 10),
	}
	callCmd := exec.CommandContext(ctx, p.IotaBin, argsv...)
	out, err = callCmd.CombinedOutput()
	return out, false, err
}
