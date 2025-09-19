package iota_sc

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/spf13/viper"
)

// IOTACommandConfig represents the configuration for executing an IOTA command
type IOTACommandConfig struct {
	Module   string   // The module name (e.g., "dcs")
	Function string   // The function name (e.g., "add_id_to_whitelist")
	Args     []string // Additional arguments for the function
	IOTABin  string   // Path to IOTA binary (optional, defaults to "iota")
}

// executeScFunction executes an IOTA client call command with the given configuration
func executeScFunction(config IOTACommandConfig) error {
	// Get package ID
	pkgID, err := GetPackageID()
	if err != nil {
		return err
	}

	// Get gas configuration
	gasConfig, err := GetGasConfig()
	if err != nil {
		return err
	}

	// Set default IOTA binary path
	iotaBin := config.IOTABin
	if iotaBin == "" {
		iotaBin = "iota"
	}

	// Build command arguments
	argsv := []string{
		"client", "call",
		"--package", pkgID,
		"--module", config.Module,
		"--function", config.Function,
		"--args",
	}
	
	// Add function arguments
	argsv = append(argsv, config.Args...)
	
	// Add gas configuration
	argsv = append(argsv, "--gas", gasConfig.GasID, "--gas-budget", strconv.FormatUint(gasConfig.GasBudget, 10))

	// Execute the command
	out, err := exec.Command(iotaBin, argsv...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute IOTA command: %w\nOutput: %s", err, string(out))
	}

	// Print the output
	os.Stdout.Write(out)
	return nil
}

// executeScFunctionWithOutput executes an IOTA client call command and returns the output
func executeScFunctionWithOutput(config IOTACommandConfig) ([]byte, error) {
	// Get package ID
	pkgID, err := GetPackageID()
	if err != nil {
		return nil, err
	}

	// Get gas configuration
	gasConfig, err := GetGasConfig()
	if err != nil {
		return nil, err
	}

	// Set default IOTA binary path
	iotaBin := config.IOTABin
	if iotaBin == "" {
		iotaBin = "iota"
	}

	// Build command arguments
	argsv := []string{
		"client", "call",
		"--package", pkgID,
		"--module", config.Module,
		"--function", config.Function,
		"--args",
	}
	
	// Add function arguments
	argsv = append(argsv, config.Args...)
	
	// Add gas configuration
	argsv = append(argsv, "--gas", gasConfig.GasID, "--gas-budget", strconv.FormatUint(gasConfig.GasBudget, 10))

	// Execute the command
	out, err := exec.Command(iotaBin, argsv...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute IOTA command: %w\nOutput: %s", err, string(out))
	}

	return out, nil
}


// GasConfig represents gas configuration for IOTA commands
type GasConfig struct {
	GasID     string
	GasBudget uint64
}

// GetPackageID retrieves the package ID from config or environment
func GetPackageID() (string, error) {
	pkgID := viper.GetString("dcs.package_id")
	if pkgID == "" {
		pkgID = os.Getenv("DCS_PACKAGE_ID")
	}
	if pkgID == "" {
		return "", fmt.Errorf("set DCS_PACKAGE_ID env var or pass --package-id (0x...)")
	}
	return pkgID, nil
}

// GetGasConfig retrieves gas configuration from environment variables
func GetGasConfig() (GasConfig, error) {
	gasID := os.Getenv("WALLET_GAS_ID")
	if gasID == "" {
		return GasConfig{}, fmt.Errorf("set WALLET_GAS_ID env var or pass --gas (0x...)")
	}

	gasBudget := uint64(0)
	if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			gasBudget = v
		}
	}
	if gasBudget == 0 {
		gasBudget = 10_000_000 // default
	}

	return GasConfig{
		GasID:     gasID,
		GasBudget: gasBudget,
	}, nil
}

// GetWhitelistID retrieves the whitelist ID from config or environment
func GetWhitelistID() (string, error) {
	whID := viper.GetString("dcs.whitelist_id")
	if whID == "" {
		whID = os.Getenv("DCS_WHITELIST_ID")
	}
	if whID == "" {
		return "", fmt.Errorf("set DCS_WHITELIST_ID env var or pass --id (0x...)")
	}
	return whID, nil
}

// GetCIDListID retrieves the CID list ID from config or environment
func GetCIDListID() (string, error) {
	cidListID := viper.GetString("dcs.cidlist_id")
	if cidListID == "" {
		cidListID = os.Getenv("DCS_CIDLIST_ID")
	}
	if cidListID == "" {
		return "", fmt.Errorf("set DCS_CIDLIST_ID env var or pass --cidlist-id (0x...)")
	}
	return cidListID, nil
}

// extractCIDObjectID extracts the CID object ID from transaction output
func extractCIDObjectID(output []byte) (string, error) {
	outputStr := string(output)
	
	// Look for object ID pattern in the output
	// The output typically contains something like "Created Objects: [0x...]"
	re := regexp.MustCompile(`0x[a-fA-F0-9]{64}`)
	matches := re.FindAllString(outputStr, -1)
	
	if len(matches) == 0 {
		return "", fmt.Errorf("no object ID found in transaction output")
	}
	
	// Return the first (and typically only) object ID
	return matches[0], nil
}