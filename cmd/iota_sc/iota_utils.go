package iota_sc

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/spf13/viper"
)

// IOTA cli wrapper

// IOTACommandConfig represents the configuration for executing an IOTA command
type IOTACommandConfig struct {
	Module   string   // The module name (e.g., "dcs")
	Function string   // The function name (e.g., "add_id_to_whitelist")
	Args     []string // Additional arguments for the function
	IOTABin  string   // Path to IOTA binary (optional, defaults to "iota")
	Account  string   // Account to use for signing the transaction
}


// executeScFunction executes an IOTA client call command and returns the output
func executeScFunction(config IOTACommandConfig) ([]byte, error) {
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

	argsv = append(argsv, "--sender", config.Account)

	fmt.Println("Executing IOTA command:", iotaBin, argsv)
	// Execute the command
	out, err := exec.Command(iotaBin, argsv...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute IOTA command: %w\nOutput: %s", err, string(out))
	}

	return out, nil
}

// fetchSharedObject fetches a shared object from the blockchain using iota client object
func fetchSharedObject(objectID string) ([]byte, error) {
	iotaBin := "iota"
	args := []string{"client", "object", objectID, "--json"}
	out, err := exec.Command(iotaBin, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch object %s: %w\nOutput: %s", objectID, err, string(out))
	}
	return out, nil
}


// GasConfig represents gas configuration for IOTA commands
type GasConfig struct {
	GasID     string
	GasBudget uint64
}


// Env variables retriever

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

func GetGasCoinID() (string, error) {
	cidListID := viper.GetString("dcs.cidcoin_id")
	if cidListID == "" {
		cidListID = os.Getenv("DCS_CIDCOIN_ID")
	}
	if cidListID == "" {
		return "", fmt.Errorf("set DCS_CIDCOIN_ID env var or pass --cidcoin-id (0x...)")
	}
	return cidListID, nil
}


// Cli output extractors

// extractCdId extracts the CID object ID from transaction output
func extractCidId(output []byte) (string, error) {
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



// Utils to retrieve shared objects properties

// getCidObject fetches a CID object and returns only the cid_str
func getCidObject(cidObjectID string) (string, error) {
	// Fetch the CID object
	objectData, err := fetchSharedObject(cidObjectID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch CID object %s: %w", cidObjectID, err)
	}

	// Parse CID object JSON
	var cidObject struct {
		Content struct {
			Fields struct {
				CIDStr string `json:"cid_str"`
			} `json:"fields"`
		} `json:"content"`
	}

	if err := json.Unmarshal(objectData, &cidObject); err != nil {
		return "", fmt.Errorf("failed to parse CID object JSON: %w", err)
	}

	return cidObject.Content.Fields.CIDStr, nil
}

// getCidList fetches the CIDlist and returns only the list of CID object IDs
func getCidList() ([]string, error) {
	// Get CID list ID
	cidListID, err := GetCIDListID()
	if err != nil {
		return nil, err
	}

	// Get the CIDlist object from the blockchain
	out, err := fetchSharedObject(cidListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get CIDlist object: %w", err)
	}

	// Parse the CIDlist JSON
	var cidListData struct {
		Content struct {
			Fields struct {
				CIDList []string `json:"cidlist"`
			} `json:"fields"`
		} `json:"content"`
	}

	if err := json.Unmarshal(out, &cidListData); err != nil {
		return nil, fmt.Errorf("failed to parse CIDlist JSON: %w", err)
	}

	return cidListData.Content.Fields.CIDList, nil
}

// Search for a particular CID (not its ID) in CIDlist
func getCidIdFromList(cidStr string) (string, error) {
	// Get the CID list
	cidList, err := getCidList()
	if err != nil {
		return "", err
	}

	// Search through each CID object ID in the list
	for _, cidObjectID := range cidList {
		// Get the CID string from the CID object
		storedCidStr, err := getCidObject(cidObjectID)
		if err != nil {
			continue // Skip if we can't fetch this CID object
		}

		// Check if the cid_str matches our target
		if storedCidStr == cidStr {
			return cidObjectID, nil
		}
	}

	return "", fmt.Errorf("CID %s not found in CIDlist", cidStr)
}