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
	"github.com/teleconsys/DCS/internal/dcserrors"
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
}

func LoadTransitionParams(cmd *cobra.Command, args []string) (TransitionParams, error) {
	var p TransitionParams

	// Get package ID
	p.PackageID = viper.GetString("dcs.package_id")
	if p.PackageID == "" {
		p.PackageID = os.Getenv("DCS_PACKAGE_ID")
	}
	if p.PackageID == "" {
		return p, fmt.Errorf("set DCS_PACKAGE_ID env var or pass --package-id (0x...)")
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

func TransitionEpoch(ctx context.Context, p TransitionParams, cidObjectID string) (bool, error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return false, fmt.Errorf("rpc dial failed: %w", err)
	}

	// Build arguments for create_cid function
	args := []any{cidObjectID, "0x6"}
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

	resp, err := w.ExecuteTransactionBlock(ctx, base64Tx, []any{sigB64}, opts, reqType)
	if err != nil {
		return false, dcserrors.Wrap("transition_epoch", err)
	}
	if err := dcserrors.TxError("transition_epoch", resp); err != nil {
		return false, err
	}

	return true, nil

}

// CheckEpochTransitionAllowed verifies that the current timestamp allows epoch transition.
// Transition is only allowed after next_epoch_start has passed.
func CheckEpochTransitionAllowed(ctx context.Context, cidObjectID, rpcURL string) error {
	// Use the argument directly as CID object ID

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

	// Extract next_epoch_start
	currentEpochEndRaw := fields["next_epoch_start"]
	currentEpochEnd := asI64FromFields(currentEpochEndRaw)

	// Get current timestamp in milliseconds
	nowMs := time.Now().UnixMilli()

	// Check if transition is allowed (only after next_epoch_start)
	if nowMs < currentEpochEnd {
		remainingMs := currentEpochEnd - nowMs
		remainingMinutes := remainingMs / 60_000
		return fmt.Errorf("epoch transition not allowed yet. Next epoch starts in %d minutes (at timestamp %d, current: %d). Transition is only effective after the actual start of the next epoch", remainingMinutes, currentEpochEnd, nowMs)
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
