package cid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/teleconsys/DCS/internal/rebased"
)

type AddFundsParams struct {
	CIDId            string
	CoinID           string
	PackageID        string
	GasID            string
	GasBudget        uint64
	RPCURL           string
	UserSignerAddress string
	UserPrivateKey   string
}

func LoadAddFundsParams(cmd *cobra.Command, args []string) (AddFundsParams, error) {
	var p AddFundsParams

	// Get CID from flag or args
	var cidArg string
	if len(args) > 0 {
		cidArg = args[0]
	} else {
		return p, fmt.Errorf("provide CID ID or CID string as argument or --cid flag")
	}

	cidType, err := cmd.Flags().GetString("cid-type")
	if err != nil {
		return p, err
	}

	// Validate cidType
	if cidType != "id" && cidType != "cid" {
		return p, fmt.Errorf("cid-type must be either 'id' or 'cid', got: %s", cidType)
	}

	// Set the cidId based on type
	if cidType == "id" {
		p.CIDId = cidArg
	} else {
		cidStr := cidArg

		// Get RPC URL first (needed for GetCIDIdFromList)
		p.RPCURL = viper.GetString("rpc")
		if p.RPCURL == "" {
			if u := os.Getenv("REBASE_RPC"); u != "" {
				p.RPCURL = u
			} else if u := os.Getenv("DCS_RPC"); u != "" {
				p.RPCURL = u
			} else {
				p.RPCURL = "https://api.testnet.iota.cafe:443"
			}
		}

		// Get the CID object ID from the CIDlist
		cidId, err := GetCIDIdFromList(cmd.Context(), cidStr, p.RPCURL)
		if err != nil {
			return p, fmt.Errorf("failed to find CID object: %w", err)
		}
		p.CIDId = cidId
	}

	// Get coin ID to deposit
	if coinIDFlag, _ := cmd.Flags().GetString("coin-id"); coinIDFlag != "" {
		p.CoinID = coinIDFlag
	} else if coinIDEnv := os.Getenv("COIN_ID"); coinIDEnv != "" {
		p.CoinID = coinIDEnv
	} else {
		return p, fmt.Errorf("set COIN_ID env var or pass --coin-id (0x...)")
	}

	// Get private key: flag takes priority over env var
	if privateKeyFlag, _ := cmd.Flags().GetString("user-private-key"); privateKeyFlag != "" {
		p.UserPrivateKey = privateKeyFlag
	} else if userPrivateKeyEnv := os.Getenv("USER_PRIVATE_KEY"); userPrivateKeyEnv != "" {
		p.UserPrivateKey = userPrivateKeyEnv
	} else {
		return p, fmt.Errorf("set USER_PRIVATE_KEY env var or pass --user-private-key")
	}

	// Get user signer address
	if signerAddress, _ := cmd.Flags().GetString("user-address"); signerAddress != "" {
		p.UserSignerAddress = signerAddress
	} else if userAddressEnv := os.Getenv("USER_ADDRESS"); userAddressEnv != "" {
		p.UserSignerAddress = userAddressEnv
	} else {
		return p, fmt.Errorf("set USER_ADDRESS env var or pass --user-address")
	}

	// Get gas coin ID for user: flag takes priority over env var
	if gasIDFlag, _ := cmd.Flags().GetString("user-gas-coin-id"); gasIDFlag != "" {
		p.GasID = gasIDFlag
	} else if gasIDEnv := os.Getenv("USER_GAS_COIN_ID"); gasIDEnv != "" {
		p.GasID = gasIDEnv
	} else {
		return p, fmt.Errorf("set USER_GAS_COIN_ID env var or pass --user-gas-coin-id (0x...)")
	}

	// Get package ID
	p.PackageID = viper.GetString("dcs.package_id")
	if p.PackageID == "" {
		p.PackageID = os.Getenv("DCS_PACKAGE_ID")
	}
	if p.PackageID == "" {
		return p, fmt.Errorf("set DCS_PACKAGE_ID env var or pass --package-id (0x...)")
	}

	if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			p.GasBudget = v
		}
	}
	if p.GasBudget == 0 {
		p.GasBudget = 10_000_000 // default
	}

	// Get RPC URL if not already set
	if p.RPCURL == "" {
		p.RPCURL = viper.GetString("rpc")
		if p.RPCURL == "" {
			if u := os.Getenv("REBASE_RPC"); u != "" {
				p.RPCURL = u
			} else if u := os.Getenv("DCS_RPC"); u != "" {
				p.RPCURL = u
			} else {
				p.RPCURL = "https://api.testnet.iota.cafe:443"
			}
		}
	}

	return p, nil
}

func AddFunds(ctx context.Context, p AddFundsParams) ([]byte, error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial failed: %w", err)
	}

	// Build arguments for deposit_funds function
	// deposit_funds(&mut CID, coin::Coin<IOTA>, &mut TxContext)
	args := []any{p.CIDId, p.CoinID}
	gasPtr := &p.GasID

	// Build unsigned transaction
	txb, err := w.UnsafeMoveCallUnsigned(
		ctx,
		p.UserSignerAddress,
		p.PackageID,
		"dcs",
		"deposit_funds",
		nil,
		args,
		gasPtr,
		p.GasBudget,
	)
	if err != nil {
		return nil, fmt.Errorf("build move call: %w", err)
	}

	// Sign transaction
	rawTx := []byte(txb.TxBytes)
	base64Tx := base64.StdEncoding.EncodeToString(rawTx)
	sigB64, err := rebased.SignTxBytes(ctx, rawTx, p.UserPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("sign tx: %w", err)
	}

	// Execute transaction
	opts := &suitypes.SuiTransactionBlockResponseOptions{
		ShowEffects:       true,
		ShowEvents:        true,
		ShowObjectChanges: true,
	}
	reqType := suitypes.ExecuteTransactionRequestType("WaitForLocalExecution")

	rsp, err := w.ExecuteTransactionBlock(ctx, base64Tx, []any{sigB64}, opts, reqType)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}

	b, _ := json.Marshal(rsp)
	return b, nil
}