package whitelist

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type HasParams struct {
	WhitelistID string
	Member      string
	IotaBin     string // default "iota"
}

func LoadHasAddressParams(cmd *cobra.Command, args []string) (HasParams, error) {
	var p HasParams

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

	// iota binary (flag optional)
	if iotaBin, _ := cmd.Flags().GetString("iota-bin"); iotaBin != "" {
		p.IotaBin = iotaBin
	} else {
		p.IotaBin = "iota"
	}
	return p, nil
}

// Has returns true if the address is present in the whitelist object.
func HasAddress(ctx context.Context, p HasParams) (bool, error) {

	out, err := exec.CommandContext(ctx, p.IotaBin, "client", "object", p.WhitelistID, "--json").Output()
	if err != nil && len(out) == 0 {
		return false, fmt.Errorf("iota client object failed: %w", err)
	}

	jsonBytes := keepJSON(out)
	if len(jsonBytes) == 0 || (jsonBytes[0] != '{' && jsonBytes[0] != '[') {

		preview := string(out)
		if len(preview) > 120 {
			preview = preview[:120] + "..."
		}
		return false, fmt.Errorf("iota CLI did not return JSON on stdout (got: %q)", preview)
	}

	fields, err := extractObjectFields(jsonBytes)
	if err != nil {
		return false, err
	}
	rawWL, ok := fields["whitelist"]
	if !ok {
		return false, nil
	}
	return whitelistContainsAddress(rawWL, p.Member), nil
}
