package iota_sc

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/teleconsys/DCS/cmd/ipfs"
)

func cidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cid",
		Short: "CID utilities",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		createCidCmd(),
		removeCidCmd(),
		isInListCidCmd(),
	)
	return cmd
}

/*
	Create a CID
*/
func createCidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [CID]",
		Short: "Create a new CID object in the smart contract. Use --file flag to upload a file to IPFS first.",
		Long: `Create a new CID object in the smart contract. You can either:

1. Provide a CID directly as an argument
2. Use the --file flag to upload a local file to IPFS first

The command will create a CID object with the specified epoch parameters and initial funding.

Examples:
  # Create CID object with existing CID
  iota_sc cid create QmWtM9FSHL8pvXVGT9dGumMSFLoGZJqRNsikSB9mWq5KgZ --coins 1000000 --epoch-start 1000 --epoch-end 2000

  # Upload a file to IPFS and create CID object
  iota_sc cid create --file /path/to/your/file.txt --coins 1000000 --epoch-start 1000 --epoch-end 2000`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var cid string
			var err error

			// Check if -file flag is used
			filePath, _ := cmd.Flags().GetString("file")
			if filePath != "" {
				// Upload file to IPFS
				cmd.Printf("Uploading file %s to IPFS...\n", filePath)
				ipfsPath, err := ipfs.LoadFileToIPFS(filePath)
				if err != nil {
					cmd.PrintErrf("Failed to load file to IPFS: %v\n", err)
					return err
				}

				// Extract CID from IPFS path (remove /ipfs/ prefix if present)
				cid = strings.TrimPrefix(ipfsPath, "/ipfs/")
				cmd.Printf("✅ File uploaded to IPFS with CID: %s\n", cid)
			} else {
				// Use CID from argument
				if len(args) == 0 {
					cmd.PrintErr("Error: either provide a CID as argument or use -file flag\n")
					return cmd.Usage()
				}
				cid = args[0]
				cmd.Printf("Using provided CID: %s\n", cid)
			}

			// Get required parameters from flags
			coins, _ := cmd.Flags().GetString("coins")
			epochStart, _ := cmd.Flags().GetUint64("epoch-start")
			epochEnd, _ := cmd.Flags().GetUint64("epoch-end")

			// Validate required parameters
			if coins == "" {
				cmd.PrintErr("Error: --coins parameter is required\n")
				return cmd.Usage()
			}
			if epochStart == 0 {
				cmd.PrintErr("Error: --epoch-start parameter is required\n")
				return cmd.Usage()
			}
			if epochEnd == 0 {
				cmd.PrintErr("Error: --epoch-end parameter is required\n")
				return cmd.Usage()
			}

			// Get whitelist ID
			whID, err := GetWhitelistID()
			if err != nil {
				return err
			}

			// Execute IOTA command to create CID object
			config := IOTACommandConfig{
				Module:   "dcs",
				Function: "create_cid",
				Args:     []string{cid, coins, fmt.Sprintf("%d", epochStart), fmt.Sprintf("%d", epochEnd), whID},
			}

			// Execute create_cid and capture output to get the CID object ID
			output, err := executeScFunctionWithOutput(config)
			if err != nil {
				cmd.PrintErrf("Failed to create CID object: %v\n", err)
				return err
			}

			// Print the create_cid output
			os.Stdout.Write(output)

			// Extract CID object ID from the output
			cidObjectID, err := extractCIDObjectID(output)
			if err != nil {
				cmd.PrintErrf("Failed to extract CID object ID: %v\n", err)
				return err
			}

			cmd.Printf("✅ CID object created with ID: %s\n", cidObjectID)

			// Get CID list ID
			cidListID, err := GetCIDListID()
			if err != nil {
				cmd.PrintErrf("Failed to get CID list ID: %v\n", err)
				return err
			}

			// Execute IOTA command to add CID to CID list
			addToListConfig := IOTACommandConfig{
				Module:   "dcs",
				Function: "add_to_cidlist",
				Args:     []string{cidObjectID, cidListID},
			}

			err = executeScFunction(addToListConfig)
			if err != nil {
				cmd.PrintErrf("Failed to add CID to CID list: %v\n", err)
				return err
			}
			
			cmd.Printf("✅ CID object %s successfully added to CID list\n", cidObjectID)
			return nil
		},
	}

	// Add flags
	cmd.Flags().String("file", "", "Path to file to upload to IPFS")
	cmd.Flags().String("coins", "", "Initial coins to fund the CID object (required)")
	cmd.Flags().Uint64("epoch-start", 0, "Next epoch start timestamp (required)")
	cmd.Flags().Uint64("epoch-end", 0, "Next epoch end timestamp (required)")
	
	// Mark required flags
	cmd.MarkFlagRequired("coins")
	cmd.MarkFlagRequired("epoch-start")
	cmd.MarkFlagRequired("epoch-end")
	
	return cmd
}

/*
	Remove a CID
*/
func removeCidCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <CID>",
		Short: "Remove a CID listed in the smart contract",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Remove the CID
			return nil
		},
	}
}

/*
	Check if a CID is listed in the smart contract
*/
func isInListCidCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "is-in-list <CID>",
		Short: "Check if a CID is listed in the smart contract",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if the CID is in the list
			return nil
		},
	}
}
