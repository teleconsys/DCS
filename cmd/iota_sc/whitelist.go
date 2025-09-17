package iota_sc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newWhitelistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "whitelist",
		Short:        "Whitelist utilities",
		SilenceUsage: true,
	}
	// Allow passing the whitelist object id via flag or env (DCS_WHITELIST_ID)
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

	c := &cobra.Command{
		Use:   "has [ADDRESS]",
		Short: "Return true if ADDRESS/ID is in the whitelist",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1) Resolve whitelist ID: flag -> viper -> env
			id := viper.GetString("dcs.whitelist_id")
			if id == "" {
				id = os.Getenv("DCS_WHITELIST_ID")
			}
			if id == "" {
				return fmt.Errorf("set DCS_WHITELIST_ID env var or pass --id (0x...)")
			}

			// 2) Resolve member: flag (priority) -> positional arg
			if mFlag, _ := cmd.Flags().GetString("member"); mFlag != "" {
				member = mFlag
			} else if len(args) > 0 {
				member = args[0]
			}
			if member == "" {
				return fmt.Errorf("provide address as positional arg or --member (0x...)")
			}
			member = strings.ToLower(member)

			// 3) Call IOTA CLI and get stdout JSON
			out, err := exec.Command("iota", "client", "object", id, "--json").Output()
			if err != nil && len(out) == 0 {
				return fmt.Errorf("iota client object failed: %w", err)
			}

			// 4) Keep only from the first '{'/'[' onward (tolerate banners on stdout)
			if i := bytes.IndexAny(out, "{["); i >= 0 {
				out = out[i:]
			}

			// 5) Extract fields.whitelist
			fields, err := extractObjectFields(out)
			if err != nil {
				return err
			}
			rawWL, ok := fields["whitelist"]
			if !ok {
				if printAddr {
					return nil
				}
				fmt.Println("false")
				return nil
			}

			found := whitelistContainsAddress(rawWL, member)
			if printAddr {
				if found {
					fmt.Println(member)
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
	return c
}

// add <ADDRESS> to whitelist by calling the on-chain function (GC must be active signer).
func newWhitelistAddCmd() *cobra.Command {
	var member string
	var pkgID string
	var gasID string
	var gasBudget uint64
	var iotaBin string

	c := &cobra.Command{
		Use:   "add [ADDRESS]",
		Short: "Add ADDRESS/ID to the whitelist (requires GroundControl signer)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --- Resolve whitelist object ID ---
			whID := viper.GetString("dcs.whitelist_id")
			if whID == "" {
				whID = os.Getenv("DCS_WHITELIST_ID")
			}
			if whID == "" {
				return fmt.Errorf("set DCS_WHITELIST_ID env var or pass --id (0x...)")
			}

			// --- Resolve member address ---
			if mFlag, _ := cmd.Flags().GetString("member"); mFlag != "" {
				member = mFlag
			} else if len(args) > 0 {
				member = args[0]
			}
			if member == "" {
				return fmt.Errorf("provide address as positional arg or --member (0x...)")
			}
			member = strings.ToLower(member)

			// --- Check if member is already in whitelist ---
			out, err := exec.Command("iota", "client", "object", whID, "--json").Output()
			if err == nil {
				// Keep only JSON part
				if i := bytes.IndexAny(out, "{["); i >= 0 {
					out = out[i:]
				}
				fields, err := extractObjectFields(out)
				if err == nil {
					if rawWL, ok := fields["whitelist"]; ok {
						if whitelistContainsAddress(rawWL, member) {
							fmt.Println("Address is already in the whitelist.")
							return nil // skip transaction
						}
					}
				}
			}

			// --- Resolve package ID ---
			if pkgID == "" {
				pkgID = viper.GetString("dcs.package_id")
			}
			if pkgID == "" {
				pkgID = os.Getenv("DCS_PACKAGE_ID")
			}
			if pkgID == "" {
				return fmt.Errorf("set DCS_PACKAGE_ID env var or --package-id (0x...)")
			}

			// --- Resolve gas ID ---
			if gasID == "" {
				gasID = os.Getenv("WALLET_GAS_ID")
			}
			if gasID == "" {
				return fmt.Errorf("set WALLET_GAS_ID env var or --gas (0x...)")
			}

			// --- Resolve gas budget ---
			if gasBudget == 0 {
				if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
					if v, err := strconv.ParseUint(s, 10, 64); err == nil {
						gasBudget = v
					}
				}
				if gasBudget == 0 {
					gasBudget = 10_000_000
				}
			}

			if iotaBin == "" {
				iotaBin = "iota"
			}

			// --- Build CLI args and call smart contract ---
			argsv := []string{
				"client", "call",
				"--package", pkgID,
				"--module", "dcs",
				"--function", "add_id_to_whitelist",
				"--args", member, whID,
				"--gas", gasID,
				"--gas-budget", strconv.FormatUint(gasBudget, 10),
			}
			out, err = exec.Command(iotaBin, argsv...).CombinedOutput()
			os.Stdout.Write(out)
			if err != nil {
				return fmt.Errorf("iota client call failed: %w", err)
			}
			return nil
		},
	}

	c.Flags().StringVarP(&member, "member", "m", "", "Address/ID to add (0x...)")
	c.Flags().StringVar(&pkgID, "package-id", "", "DCS package ID (0x...)")
	c.Flags().StringVar(&gasID, "gas", "", "Gas coin object ID (0x...)")
	c.Flags().Uint64Var(&gasBudget, "gas-budget", 0, "Gas budget (nanos)")
	c.Flags().StringVar(&iotaBin, "iota-bin", "", "Path to iota binary (default: iota)")
	return c
}

func newWhitelistRemoveCmd() *cobra.Command {
	var member string
	var pkgID string
	var gasID string
	var gasBudget uint64
	var iotaBin string

	c := &cobra.Command{
		Use:   "remove [ADDRESS]",
		Short: "Remove ADDRESS/ID from the whitelist (requires GroundControl signer)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1) Whitelist object ID
			whID := viper.GetString("dcs.whitelist_id")
			if whID == "" {
				whID = os.Getenv("DCS_WHITELIST_ID")
			}
			if whID == "" {
				return fmt.Errorf("set DCS_WHITELIST_ID env var or pass --id (0x...)")
			}

			// 2) Member: flag (priority) -> positional
			if mFlag, _ := cmd.Flags().GetString("member"); mFlag != "" {
				member = mFlag
			} else if len(args) > 0 {
				member = args[0]
			}
			if member == "" {
				return fmt.Errorf("provide address as positional arg or --member (0x...)")
			}
			member = strings.ToLower(member)

			// 3) Package ID (DCS)
			if pkgID == "" {
				pkgID = viper.GetString("dcs.package_id")
			}
			if pkgID == "" {
				pkgID = os.Getenv("DCS_PACKAGE_ID")
			}
			if pkgID == "" {
				return fmt.Errorf("set DCS_PACKAGE_ID env var or --package-id (0x...)")
			}

			// 4) Gas coin + budget
			if gasID == "" {
				gasID = os.Getenv("WALLET_GAS_ID")
			}
			if gasID == "" {
				return fmt.Errorf("set WALLET_GAS_ID env var or --gas (0x...)")
			}
			if gasBudget == 0 {
				if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
					if v, err := strconv.ParseUint(s, 10, 64); err == nil {
						gasBudget = v
					}
				}
				if gasBudget == 0 {
					gasBudget = 10_000_000
				}
			}

			// 5) IOTA client command (remove from whitelist)
			if iotaBin == "" {
				iotaBin = "iota"
			}

			argsv := []string{
				"client", "call",
				"--package", pkgID,
				"--module", "dcs",
				"--function", "remove_id_from_whitelist",
				"--args", member, whID,
				"--gas", gasID,
				"--gas-budget", strconv.FormatUint(gasBudget, 10),
			}
			out, err := exec.Command(iotaBin, argsv...).CombinedOutput()

			// Print raw result
			//TODO: Clean output if necessary
			os.Stdout.Write(out)
			if err != nil {
				return fmt.Errorf("iota client call failed: %w", err)
			}
			return nil
		},
	}

	// Flags (optional if env is set)
	c.Flags().StringVarP(&member, "member", "m", "", "Address/ID to remove (0x...)")
	c.Flags().StringVar(&pkgID, "package-id", "", "DCS package ID (0x...)")
	c.Flags().StringVar(&gasID, "gas", "", "Gas coin object ID (0x...)")
	c.Flags().Uint64Var(&gasBudget, "gas-budget", 0, "Gas budget (nanos)")
	c.Flags().StringVar(&iotaBin, "iota-bin", "", "Path to iota binary (default: iota)")
	return c
}

// Extract fields map from either shape:
// 1) root.content.fields (current CLI)
// 2) root.data.content.fields (legacy)
func extractObjectFields(jsonBytes []byte) (map[string]any, error) {
	var root map[string]any
	if err := json.Unmarshal(jsonBytes, &root); err != nil {
		return nil, fmt.Errorf("decode object json: %w", err)
	}
	if content, ok := root["content"].(map[string]any); ok {
		if fields, ok := content["fields"].(map[string]any); ok {
			return fields, nil
		}
	}
	if data, ok := root["data"].(map[string]any); ok {
		if content, ok := data["content"].(map[string]any); ok {
			if fields, ok := content["fields"].(map[string]any); ok {
				return fields, nil
			}
		}
	}
	return nil, fmt.Errorf("fields not found in object JSON")
}

// Check presence of needle in fields.whitelist (strings or maps {"id":..., "bytes":...})
func whitelistContainsAddress(raw any, needle string) bool {
	switch arr := raw.(type) {
	case []any:
		for _, item := range arr {
			switch t := item.(type) {
			case string:
				if strings.EqualFold(t, needle) {
					return true
				}
			case map[string]any:
				if s, _ := t["id"].(string); s != "" && strings.EqualFold(s, needle) {
					return true
				}
				if s, _ := t["bytes"].(string); s != "" && strings.EqualFold(s, needle) {
					return true
				}
			}
		}
	}
	return false
}
