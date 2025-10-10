package iota_sc

import (
	"fmt"
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
		Use:   "create --type <path|cid> [CID]",
		Short: "Create a new CID object in the smart contract. Use --type flag to specify the type of input (path or cid).",
		Long: `Create a new CID object in the smart contract. You can either:

1. Provide a CID directly as an argument
2. Use the --type flag to specify the type of input (path or cid)

The command will create a CID object with the specified epoch parameters and initial funding.

Examples:
  # Create CID object with existing CID
  iota_sc cid create --type cid QmWtM9FSHL8pvXVGT9dGumMSFLoGZJqRNsikSB9mWq5KgZ --coins 1000000 --epoch-start 1000 --epoch-end 2000

  # Upload a file to IPFS and create CID object
  iota_sc cid create --type path /path/to/your/file.txt --coins 1000000 --epoch-start 1000 --epoch-end 2000`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var cid string
			var err error

			// Check if -file flag is used
			inputType, _ := cmd.Flags().GetString("type")

			if inputType != "path" && inputType != "cid" {
				cmd.PrintErrf("Error: type must be either 'path' or 'cid', got: %s\n", inputType)
				return cmd.Usage()
			}

			if inputType == "path" {
				if len(args) == 0 {
					cmd.PrintErr("Error: provide a path as argument\n")
					return cmd.Usage()
				}
				filePath := args[0]
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
				if len(args) == 0 {
					cmd.PrintErr("Error: provide a CID as argument\n")
					return cmd.Usage()
				}
				// Use CID from argument
				cid = args[0]
				cmd.Printf("Using provided CID: %s\n", cid)
				
				// Convert CID to hexadecimal
				// cidHex, err := cidToHex(cid)
				// if err != nil {
				// 	cmd.PrintErrf("Failed to convert CID to hex: %v\n", err)
				// 	return err
				// }
				// cid = cidHex
				// cmd.Printf("Converted CID to hex: %s\n", cid)
			}

			// Get required parameters from flags
			epochStart, _ := cmd.Flags().GetUint64("epoch-start")
			epochEnd, _ := cmd.Flags().GetUint64("epoch-end")

			// Validate required parameters
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

			coinID, err := GetGasCoinID()
			if err != nil {
				return err
			}

			owner, err := cmd.Flags().GetString("owner")
			if err != nil {
				return err
			}

			// Execute IOTA command to create CID object
			config := IOTACommandConfig{
				Module:   "dcs",
				Function: "create_cid",
				Args:     []string{cid, coinID, fmt.Sprintf("%d", epochStart), fmt.Sprintf("%d", epochEnd), whID},
				Account:  owner,
			}

			// Execute create_cid and capture output to get the CID object ID
			output, err := executeScFunction(config)
			if err != nil {
				cmd.PrintErrf("Failed to create CID object: %v\n", err)
				return err
			}

			// Extract CID object ID from the output
			cidId, err := extractCidId(output)
			if err != nil {
				cmd.PrintErrf("Failed to extract CID object ID: %v\n", err)
				return err
			}

			cmd.Printf("✅ CID object created with ID: %s\n", cidId)

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
				Args:     []string{cidId, cidListID},
			}

			_, err = executeScFunction(addToListConfig)
			if err != nil {
				cmd.PrintErrf("Failed to add CID id to CID list: %v\n", err)
				return err
			}

			
			cmd.Printf("✅ CID id %s successfully added to CID list\n", cidId)
			return nil
		},
	}

	// Add flags
	cmd.Flags().String("type", "", "Type of input (path or cid)")
	cmd.Flags().String("owner", "", "IOTA address of the owner of the CID object (required)")
	cmd.Flags().Uint64("epoch-start", 0, "Next epoch start timestamp (required)")
	cmd.Flags().Uint64("epoch-end", 0, "Next epoch end timestamp (required)")
	
	// Mark required flags
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("epoch-start")
	cmd.MarkFlagRequired("epoch-end")
	cmd.MarkFlagRequired("owner")
	return cmd
}

/*
	Remove a CID
*/
func removeCidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove --cid-type <id|cid> [CIDID]",
		Short: "Remove a CID listed in the smart contract",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			cidType, err := cmd.Flags().GetString("cid-type")
			if err != nil {
				return err
			}

			// Validate cidType
			if cidType != "id" && cidType != "cid" {
				cmd.PrintErrf("Error: cid-type must be either 'id' or 'cid', got: %s\n", cidType)
				return cmd.Usage()
			}

			var cidId string

			// Set the cidId
			if cidType == "id" {
				cidId = args[0]
			} else {
				cidStr := args[0]

				// Get the CID object ID from the CIDlist
				cidId, err = getCidIdFromList(cidStr)
				if err != nil {
					cmd.PrintErrf("Failed to find CID object: %v\n", err)
					return err
				}
			}


			// Get the CIDlist ID
			cidListID, err := GetCIDListID()
			if err != nil {
				return err
			}

			// Execute IOTA command to remove CID from CIDlist
			funcExec := IOTACommandConfig{
				Module:   "dcs",
				Function: "remove_from_cidlist",
				Args:     []string{cidId, cidListID},
				Account:  "0xb536e8aad181525e772111eeed764f7c92b8502b9692d705fae643157e5d445d",
			}

			_, err = executeScFunction(funcExec)
			if err != nil {
				cmd.PrintErrf("Failed to remove CID from CID list: %v\n", err)
				return err
			}

			cmd.Printf("✅ Object with CID id %s successfully removed from CID list\n", cidId)
			return nil
		},
	}

	cmd.Flags().String("cid-type", "", "type of cid (id or cid)")
	cmd.MarkFlagRequired("cid-type")

	return cmd


}

/*
	Check if a CID is listed in the smart contract
*/
func isInListCidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "is-in-list --cid-type <id|cid> [CIDID]",
		Short: "Check if a CID ID is listed in the smart contract",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			cidType, err := cmd.Flags().GetString("cid-type")
			if err != nil {
				return err
			}

			// Validate cidType
			if cidType != "id" && cidType != "cid" {
				cmd.PrintErrf("Error: cid-type must be either 'id' or 'cid', got: %s\n", cidType)
				return cmd.Usage()
			}

			var cidId string

			// Set the cidId
			if cidType == "id" {
				cidId = args[0]
			} else {
				cidStr := args[0]

				// Get the CID object ID from the CIDlist
				cidId, err = getCidIdFromList(cidStr)
				if err != nil {
					cmd.PrintErrf("Failed to find CID object: %v\n", err)
					return err
				}
			}


			cidList, err := getCidList()
			if err != nil {
				cmd.PrintErrf("Failed to get CID list: %v\n", err)
				return err
			}

			for _, cid := range cidList {
				if cid == cidId {
					cmd.Printf("✅ CID object %s is in CID list\n", cidId)
					return nil
				}
			}

			cmd.Printf("✅ CID object %s is not in CID list\n", cidId)
			return nil
		},
	}

	cmd.Flags().String("cid-type", "", "type of cid (id or cid)")
	cmd.MarkFlagRequired("cid-type")

	return cmd
}
