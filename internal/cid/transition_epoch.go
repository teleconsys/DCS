package cid

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/teleconsys/DCS/internal/rebased"
	"github.com/teleconsys/DCS/internal/wallet"
)

type TransitionParams struct {
	RPCURL            string
	GasID             string
	GasBudget         uint64
	UserSignerAddress string
	UserPrivateKey    string
	PackageID         string
	CIDType           string
}

func LoadTransitionParams(cmd *cobra.Command, args []string) (TransitionParams, error) {
	var p TransitionParams

	cidType, err := cmd.Flags().GetString("cid-type")
	if err != nil {
		return p, err
	}

	// Validate cidType
	if cidType != "id" && cidType != "cid" {
		return p, fmt.Errorf("cid-type must be either 'id' or 'cid', got: %s", cidType)
	}
	p.CIDType = cidType
	// Get package ID
	p.PackageID = viper.GetString("dcs.package_id")
	if p.PackageID == "" {
		p.PackageID = os.Getenv("DCS_PACKAGE_ID")
	}
	if p.PackageID == "" {
		return p, fmt.Errorf("set DCS_PACKAGE_ID env var or pass --package-id (0x...)")
	}

	// Read private key
	privKeyFlag, _ := cmd.Flags().GetString("user-private-key")
	privKey, err := wallet.ResolvePrivateKey(privKeyFlag)
	if err != nil {
		return p, err
	}
	p.UserPrivateKey = privKey

	// Resolve signer address (derive from private key, compare with flag/env, confirm if mismatch)
	signerFlag, _ := cmd.Flags().GetString("user-address")
	signer, err := wallet.ResolveSignerAddress(privKey, signerFlag, "ACTIVE_ADDRESS", "USER_ADDRESS")
	if err != nil {
		return p, err
	}
	p.UserSignerAddress = signer

	// Get gas gas coin ID for user, this will be used to create the new COIN object for the cid creation (flags > ACTIVE_* > USER_*)
	gasIDFlag, _ := cmd.Flags().GetString("user-coin-id")
	gasID := wallet.FirstNonEmpty(
		gasIDFlag,
		os.Getenv("ACTIVE_GAS_COIN_ID"),
		os.Getenv("USER_GAS_COIN_ID"),
	)
	if gasID == "" {
		return p, fmt.Errorf("missing gas coin id (set --user-coin-id or ACTIVE_GAS_COIN_ID / USER_GAS_COIN_ID)")
	}
	p.GasID = gasID

	if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			p.GasBudget = v
		}
	}
	if p.GasBudget == 0 {
		p.GasBudget = 10_000_000 // default
	}

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

	return p, nil
}

func TransitionEpoch(ctx context.Context, p TransitionParams, cid string) (bool, error) {

	CID := cid
	var err error

	if p.CIDType == "cid" {
		CID, err = GetCIDIdFromList(ctx, cid, p.RPCURL)
		if err != nil {
			return false, fmt.Errorf("failed to get CID ID: %w", err)
		}
	}

	if err != nil {
		return false, fmt.Errorf("failed to get CID ID: %w", err)
	}

	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return false, fmt.Errorf("rpc dial failed: %w", err)
	}

	// Build arguments for create_cid function
	args := []any{CID, "0x6"}
	gasPtr := &p.GasID

	// Build unsigned transaction
	txb, err := w.UnsafeMoveCallUnsigned(
		ctx,
		p.UserSignerAddress,
		p.PackageID,
		"dcs",
		"transition_epoch",
		nil,
		args,
		gasPtr,
		p.GasBudget,
	)
	if err != nil {
		return false, fmt.Errorf("build move call: %w", err)
	}

	// Sign transaction
	rawTx := []byte(txb.TxBytes)
	base64Tx := base64.StdEncoding.EncodeToString(rawTx)
	sigB64, err := rebased.SignTxBytes(ctx, rawTx, p.UserPrivateKey)
	if err != nil {
		return false, fmt.Errorf("sign tx: %w", err)
	}

	// Execute transaction
	opts := &suitypes.SuiTransactionBlockResponseOptions{
		ShowEffects:       true,
		ShowEvents:        true,
		ShowObjectChanges: true,
	}
	reqType := suitypes.ExecuteTransactionRequestType("WaitForLocalExecution")

	_, err = w.ExecuteTransactionBlock(ctx, base64Tx, []any{sigB64}, opts, reqType)
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}

	return true, nil

}
