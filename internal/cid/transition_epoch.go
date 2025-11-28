package cid

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"

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

	// Get gas gas coin ID for user, this will be used to create the new COIN object for the cid creation (flags > ACTIVE_* > USER_*)
	gasIDFlag, _ := cmd.Flags().GetString("signer-gas-id")
	gasID := wallet.FirstNonEmpty(
		gasIDFlag,
		os.Getenv("ACTIVE_GAS_COIN_ID"),
		os.Getenv("USER_GAS_COIN_ID"),
	)
	if gasID == "" {
		return p, fmt.Errorf("missing gas coin id (set --signer-gas-id or ACTIVE_GAS_COIN_ID / USER_GAS_COIN_ID)")
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

// CheckEpochTransitionAllowed verifies that the current timestamp allows epoch transition.
// Transition is only allowed after current_epoch_end has passed.
func CheckEpochTransitionAllowed(ctx context.Context, cidArg, cidType, rpcURL string) error {
	// Get CID object ID
	cidObjectID := cidArg
	if cidType == "cid" {
		var err error
		cidObjectID, err = GetCIDIdFromList(ctx, cidArg, rpcURL)
		if err != nil {
			return fmt.Errorf("failed to get CID ID: %w", err)
		}
	}

	// Dial RPC
	w, err := rebased.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("rpc dial failed: %w", err)
	}

	// Get CID fields
	fields, err := GetCIDFields(ctx, w, cidObjectID)
	if err != nil {
		return fmt.Errorf("failed to get CID fields: %w", err)
	}

	// Extract current_epoch_end
	currentEpochEndRaw := fields["current_epoch_end"]
	currentEpochEnd := asI64FromFields(currentEpochEndRaw)

	// Get current timestamp in milliseconds
	nowMs := time.Now().UnixMilli()

	// Check if transition is allowed (only after current_epoch_end)
	if nowMs < currentEpochEnd {
		remainingMs := currentEpochEnd - nowMs
		remainingMinutes := remainingMs / 60_000
		return fmt.Errorf("epoch transition not allowed yet. Current epoch ends in %d minutes (at timestamp %d, current: %d). Transition is only effective after the actual end of the current epoch", remainingMinutes, currentEpochEnd, nowMs)
	}

	return nil
}

// asI64FromFields converts a field value to int64, similar to offers.asI64
func asI64FromFields(v any) int64 {
	switch t := v.(type) {
	case string:
		u, _ := strconv.ParseUint(t, 10, 64)
		return int64(u)
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	default:
		return 0
	}
}
