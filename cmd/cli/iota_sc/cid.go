package iota_sc

import (
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"

	"github.com/teleconsys/DCS/cmd/cli/ipfs"
	cid_sc "github.com/teleconsys/DCS/internal/cid"
)

func cidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "cid",
		Short:        "CID utilities",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		createCidCmd(),
		removeCidCmd(),
		isInListCidCmd(),
		transitionEpochCmd(),
		addFundsCidCmd(),
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
  iota_sc cid create --type cid QmWtM9FSHL8pvXVGT9dGumMSFLoGZJqRNsikSB9mWq5KgZ --amount 1000000 --epoch-start 1000 --epoch-end 2000

  # Upload a file to IPFS and create CID object
  iota_sc cid create --type path /path/to/your/file.txt --amount 1000000 --epoch-start 1000 --epoch-end 2000`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var cidStr string
			var err error

			// Check input type
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
				cidStr = strings.TrimPrefix(ipfsPath, "/ipfs/")
				cmd.Printf("✅ File uploaded to IPFS with CID: %s\n", cidStr)
			} else {
				if len(args) == 0 {
					cmd.PrintErr("Error: provide a CID as argument\n")
					return cmd.Usage()
				}
				// Use CID from argument
				cidStr = args[0]
				cmd.Printf("Using provided CID: %s\n", cidStr)
			}

			// Load parameters using the new wrapper
			params, err := cid_sc.LoadCreateParams(cmd, []string{cidStr})
			if err != nil {
				cmd.PrintErrf("Failed to load parameters: %v\n", err)
				return err
			}

			// Split coin first to get the coin ID for CID creation
			cmd.Printf("Creating a new gas coin for the CID creation...\n")
			cidCoinId, err := cid_sc.CreateGasCoin(cmd.Context(), params.GasCoinConfig(), params.Amount)
			if err != nil {
				cmd.PrintErrf("❌ Failed to create a new gas coin: %v\n", err)
				return err
			}
			cmd.Printf("✅ New gas coin created successfully, new coin ID: %s\n", cidCoinId)

			// Create CID object using new wrapper
			cmd.Printf("Creating CID object...\n")
			_, cidId, err := cid_sc.CreateCID(cmd.Context(), params, cidCoinId)
			if err != nil {
				cmd.PrintErrf("❌ Failed to create CID object: %v\n", err)
				return err
			}

			cmd.Printf("✅ CID object created with ID: %s\n", cidId)
			cmd.Printf("✅ Coin ID created for CID object: %s\n", cidCoinId)

			// Add CID to CID list using new wrapper
			cmd.Printf("Adding CID to CID list...\n")
			_, err = cid_sc.AddToCIDList(cmd.Context(), params, cidId)
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
	cmd.Flags().Uint64("epoch-start", 0, "Next epoch start timestamp (required)")
	cmd.Flags().Uint64("epoch-end", 0, "Next epoch end timestamp (required)")
	cmd.Flags().Uint64("amount", 100000, "Amount for the gas coin object associated to the CID object (IOTA nanos, default: 100000)")
	cmd.Flags().String("signer-private-key", "", "Private key for signing (overrides ACTIVE_PRIVATE_KEY env var); if omitted you will be promped to insert it")
	cmd.Flags().String("signer-address", "", "Address of the signer (overwrite ACTIVE_ADDRESS and USER_ADDRESS env var)")
	cmd.Flags().String("signer-gas-id", "", "Gas coin object ID (overwrite ACTIVE_GAS_COIN_ID and USER_GAS_COIN_ID env var)")

	// Mark required flags
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("epoch-start")
	cmd.MarkFlagRequired("epoch-end")
	return cmd
}

/*
Remove a CID
*/
func removeCidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [objectId]",
		Short: "Remove a CID listed in the smart contract",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load parameters using the new wrapper
			params, err := cid_sc.LoadRemoveParams(cmd, args)
			if err != nil {
				cmd.PrintErrf("Failed to load parameters: %v\n", err)
				return err
			}

			// Remove CID using new wrapper
			_, err = cid_sc.RemoveCID(cmd.Context(), params)
			if err != nil {
				cmd.PrintErrf("Failed to remove CID from CID list: %v\n", err)
				return err
			}

			cmd.Printf("✅ Object with CID id %s successfully removed from CID list\n", params.CIDId)
			return nil
		},
	}

	cmd.Flags().String("signer-private-key", "", "Private key for signing (overrides ACTIVE_PRIVATE_KEY env var)")
	cmd.Flags().String("signer-address", "", "Address of the signer (overwrite ACTIVE_ADDRESS and USER_ADDRESS env var)")
	cmd.Flags().String("signer-gas-id", "", "Gas coin object ID (overwrite ACTIVE_GAS_COIN_ID and USER_GAS_COIN_ID env var)")

	return cmd
}

/*
Check if a CID is listed in the smart contract
*/
func isInListCidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "is-in-list [objectId]",
		Short: "Check if a CID ID is listed in the smart contract",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load parameters using the new wrapper
			params, err := cid_sc.LoadIsInListParams(cmd, args)
			if err != nil {
				cmd.PrintErrf("Failed to load parameters: %v\n", err)
				return err
			}

			// Check if CID is in list using new wrapper
			found, err := cid_sc.IsInList(cmd.Context(), params)
			if err != nil {
				cmd.PrintErrf("Failed to check CID list: %v\n", err)
				return err
			}

			if found {
				cmd.Printf("✅ CID object %s is in CID list\n", params.CIDId)
			} else {
				cmd.Printf("✅ CID object %s is not in CID list\n", params.CIDId)
			}
			return nil
		},
	}

	return cmd
}

func transitionEpochCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next-epoch [objectId]",
		Short: "Transition to the next epoch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Load parameters using the new wrapper
			params, err := cid_sc.LoadTransitionParams(cmd, args)
			if err != nil {
				cmd.PrintErrf("❌ Failed to load parameters: %v\n", err)
				return err
			}

			// Check if epoch transition is allowed (only after current_epoch_end)
			if err := cid_sc.CheckEpochTransitionAllowed(cmd.Context(), args[0], params.RPCURL); err != nil {
				cmd.Printf("⚠️  Warning: %v\n", err)
				return nil
			}

			_, err = cid_sc.TransitionEpoch(cmd.Context(), params, args[0])
			if err != nil {
				cmd.PrintErrf("❌ Failed to transition epoch: %v\n", err)
				return err
			}

			cmd.Printf("✅ Epoch transition successful\n")
			return nil
		},
	}

	cmd.Flags().String("signer-private-key", "", "Private key for signing (overrides ACTIVE_PRIVATE_KEY env var)")
	cmd.Flags().String("signer-address", "", "Address of the signer (overwrite ACTIVE_ADDRESS and USER_ADDRESS env var)")
	cmd.Flags().String("signer-gas-id", "", "Gas coin object ID (overwrite ACTIVE_GAS_COIN_ID and USER_GAS_COIN_ID env var)")

	return cmd
}

/*
Add funds to a CID
*/
func addFundsCidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add-funds [objectId]",
		Aliases: []string{"add_funds"},
		Short:   "Deposit IOTA coins into a CID",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load parameters using the new wrapper
			params, err := cid_sc.LoadAddFundsParams(cmd, args)
			if err != nil {
				cmd.PrintErrf("Failed to load parameters: %v\n", err)
				return err
			}

			// Create a fresh gas coin with the requested amount
			cmd.Printf("Creating a gas coin with amount %d...\n", params.Amount)
			coinID, err := cid_sc.CreateGasCoin(cmd.Context(), params.GasCoinConfig(), params.Amount)
			if err != nil {
				cmd.PrintErrf("Failed to create gas coin: %v\n", err)
				return err
			}
			cmd.Printf("✅ Gas coin created: %s\n", coinID)
			params.CoinID = coinID

			// Add funds using new wrapper with freshly minted coin
			respBytes, err := cid_sc.AddFunds(cmd.Context(), params)
			if err != nil {
				cmd.PrintErrf("Failed to add funds: %v\n", err)
				return err
			}

			// Parse response to get digest
			var resp map[string]interface{}
			if err := json.Unmarshal(respBytes, &resp); err == nil {
				if digest, ok := resp["digest"].(string); ok {
					cmd.Println("✅ funds deposited")
					cmd.Printf("digest: %s\n", digest)
				} else {
					cmd.Println("✅ funds deposited")
					cmd.Printf("response: %s\n", string(respBytes))
				}
			} else {
				cmd.Println("✅ funds deposited")
				cmd.Printf("response: %s\n", string(respBytes))
			}

			return nil
		},
	}

	cmd.Flags().Uint64("amount", 0, "Amount to deposit into the CID (IOTA nanos)")
	cmd.Flags().String("signer-address", "", "Signer address (0x...) overrides env")
	cmd.Flags().String("signer-private-key", "", "Private key for signing, if omitted you will be prompted to insert it")
	cmd.Flags().String("signer-gas-id", "", "Gas coin object ID to pay for the transaction (0x...) overrides env")

	cmd.MarkFlagRequired("amount")

	return cmd
}
