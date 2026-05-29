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

	"github.com/teleconsys/DCS/internal/dcserrors"
	"github.com/teleconsys/DCS/internal/rebased"
	"github.com/teleconsys/DCS/internal/wallet"
)

type RemoveParams struct {
	CIDId             string
	CIDListID         string
	PackageID         string
	GasID             string
	GasBudget         uint64
	RPCURL            string
	UserSignerAddress string
	UserPrivateKey    string
}

func LoadRemoveParams(cmd *cobra.Command, args []string) (RemoveParams, error) {
	var p RemoveParams

	if len(args) == 0 {
		return p, fmt.Errorf("provide CID object ID as argument")
	}

	// Use the argument directly as CID object ID
	p.CIDId = args[0]

	// Get RPC URL
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

	// Read private key
	privKeyFlag, _ := cmd.Flags().GetString("signer-private-key")
	privKey, err := wallet.ResolvePrivateKey(privKeyFlag)
	if err != nil {
		return p, err
	}
	p.UserPrivateKey = privKey

	// Resolve signer address (derive from private key, compare with flag/env, confirm if mismatch)
	signerFlag, _ := cmd.Flags().GetString("signer-address")
	signer, err := wallet.ResolveSignerAddress(privKey, signerFlag, "ACTIVE_ADDRESS", "USER_ADDRESS")
	if err != nil {
		return p, err
	}
	p.UserSignerAddress = signer

	// Resolve gas coin ID and verify ownership
	gasIDFlag, _ := cmd.Flags().GetString("signer-gas-id")
	gasID, err := wallet.ResolveGasCoinId(cmd.Context(), gasIDFlag, signer, p.RPCURL, "ACTIVE_GAS_COIN_ID", "USER_GAS_COIN_ID")
	if err != nil {
		return p, err
	}
	p.GasID = gasID

	// Get package ID
	p.PackageID = viper.GetString("dcs.package_id")
	if p.PackageID == "" {
		p.PackageID = os.Getenv("DCS_PACKAGE_ID")
	}
	if p.PackageID == "" {
		return p, fmt.Errorf("set DCS_PACKAGE_ID env var or pass --package-id (0x...)")
	}

	// Get CID list ID
	p.CIDListID = viper.GetString("dcs.cidlist_id")
	if p.CIDListID == "" {
		p.CIDListID = os.Getenv("DCS_CIDLIST_ID")
	}
	if p.CIDListID == "" {
		return p, fmt.Errorf("set DCS_CIDLIST_ID env var or pass --cidlist-id (0x...)")
	}

	if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			p.GasBudget = v
		}
	}
	if p.GasBudget == 0 {
		p.GasBudget = 10_000_000 // default
	}

	return p, nil
}

func RemoveCID(ctx context.Context, p RemoveParams) ([]byte, error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial failed: %w", err)
	}

	// Build arguments for remove_from_cidlist function
	args := []any{p.CIDId, p.CIDListID}
	gasPtr := &p.GasID

	// Build unsigned transaction
	txb, err := w.UnsafeMoveCallUnsigned(
		ctx,
		p.UserSignerAddress,
		p.PackageID,
		"dcs",
		"remove_from_cidlist",
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
		return nil, dcserrors.Wrap("remove_from_cidlist", err)
	}
	if err := dcserrors.TxError("remove_from_cidlist", rsp); err != nil {
		return nil, err
	}

	b, _ := json.Marshal(rsp)
	return b, nil
}
