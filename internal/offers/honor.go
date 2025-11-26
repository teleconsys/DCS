package offers

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/spf13/cobra"
	cidlib "github.com/teleconsys/DCS/internal/cid"
	"github.com/teleconsys/DCS/internal/rebased"
	"github.com/teleconsys/DCS/internal/wallet"
)

type HonorOfferParams struct {
	CIDObjectID string
	Index       uint64
	ClockID     string
	PackageID   string
	GasID       string
	GasBudget   uint64
	RPCURL      string
	Signer      string
	PrivKey     string
	Debug       bool
}

// LoadHonorParams loads parameters from command flags and environment variables
func LoadHonorParams(ctx context.Context, cmd *cobra.Command, args []string) (HonorOfferParams, error) {
	var p HonorOfferParams

	// Get CID from flag
	cidArg, _ := cmd.Flags().GetString("cid")
	if cidArg == "" {
		return p, fmt.Errorf("--cid is required (object id or CID string)")
	}

	// Get CID type
	cidType, _ := cmd.Flags().GetString("cid-type")
	if cidType != "id" && cidType != "cid" {
		return p, fmt.Errorf("--cid-type must be 'id' or 'cid'")
	}

	// Get RPC URL (needed for CID resolution)
	rpc := os.Getenv("REBASE_RPC")
	if rpc == "" {
		rpc = "https://api.testnet.iota.cafe:443"
	}
	p.RPCURL = rpc

	// Resolve CID to object ID if needed
	cidObjectID := cidArg
	if cidType == "cid" {
		id, err := cidlib.GetCIDIdFromList(ctx, cidArg, rpc)
		if err != nil {
			return p, err
		}
		cidObjectID = id
	}
	p.CIDObjectID = cidObjectID

	// Get index from flag
	idx, _ := cmd.Flags().GetUint64("idx")
	p.Index = idx

	// Get debug flag
	debug, _ := cmd.Flags().GetBool("debug")
	p.Debug = debug

	// Read private key
	privKeyFlag, _ := cmd.Flags().GetString("signer-private-key")
	privKey, err := wallet.ResolvePrivateKey(privKeyFlag)
	if err != nil {
		return p, err
	}
	p.PrivKey = privKey

	// Resolve signer address (derive from private key, compare with flag/env, confirm if mismatch)
	signerFlag, _ := cmd.Flags().GetString("signer-address")
	signer, err := wallet.ResolveSignerAddress(privKey, signerFlag, "ACTIVE_ADDRESS", "USER_ADDRESS")
	if err != nil {
		return p, err
	}
	p.Signer = signer

	// Get gas coin ID: flags > ACTIVE_* > USER_*
	gasIDFlag, _ := cmd.Flags().GetString("gas-id")
	gasID := wallet.FirstNonEmpty(
		gasIDFlag,
		os.Getenv("ACTIVE_GAS_COIN_ID"),
		os.Getenv("USER_GAS_COIN_ID"),
	)
	if gasID == "" {
		return p, fmt.Errorf("missing gas coin id (set --gas-id or ACTIVE_GAS_COIN_ID / USER_GAS_COIN_ID)")
	}
	p.GasID = gasID

	// Get Clock ID
	clockID := os.Getenv("DCS_CLOCK_ID")
	if clockID == "" {
		clockID = "0x6"
	}
	p.ClockID = clockID

	// Get Package ID
	packageID := os.Getenv("DCS_PACKAGE_ID")
	if packageID == "" {
		return p, fmt.Errorf("missing DCS_PACKAGE_ID env var")
	}
	p.PackageID = packageID

	// Get Gas Budget
	gasBudgetStr := os.Getenv("WALLET_GAS_BUDGET")
	if gasBudgetStr == "" {
		p.GasBudget = 10_000_000 // default
	} else {
		gasBudget, err := strconv.ParseUint(gasBudgetStr, 10, 64)
		if err != nil {
			return p, fmt.Errorf("invalid WALLET_GAS_BUDGET: %w", err)
		}
		p.GasBudget = gasBudget
	}

	return p, nil
}

// Calls: <PackageID>::dcs::approve_offer(&mut CID, u64, &Clock, &mut TxContext)
func HonorOffer(ctx context.Context, p HonorOfferParams) (*suitypes.SuiTransactionBlockResponse, error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial: %w", err)
	}

	// u64 must be a decimal string for this RPC
	idxStr := strconv.FormatUint(p.Index, 10)
	args := []any{
		p.CIDObjectID, // &mut CID (shared)
		idxStr,        // u64 as string
		p.ClockID,     // &Clock (shared)
	}

	if p.Debug {
		fmt.Println("== approve_offer args ==")
		fmt.Printf("CID: %s\n", p.CIDObjectID)
		fmt.Printf("Index: %s\n", idxStr)
		fmt.Printf("ClockID: %s\n", p.ClockID)
	}

	gas := &p.GasID
	txb, err := w.UnsafeMoveCallUnsigned(
		ctx,
		p.Signer,
		p.PackageID,
		"dcs",
		"honor_offer",
		nil,
		args,
		gas,
		p.GasBudget,
	)
	if err != nil {
		return nil, fmt.Errorf("build tx: %w", err)
	}

	raw := txb.TxBytes
	txB64 := base64.StdEncoding.EncodeToString(raw)

	sigB64, err := rebased.SignTxBytes(ctx, raw, p.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("sign tx: %w", err)
	}

	opts := &suitypes.SuiTransactionBlockResponseOptions{
		ShowInput:         true,
		ShowEffects:       true,
		ShowEvents:        true,
		ShowObjectChanges: true,
	}
	reqType := suitypes.ExecuteTransactionRequestType("WaitForLocalExecution")

	resp, err := w.ExecuteTransactionBlock(ctx, txB64, []any{sigB64}, opts, reqType)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}

	ok, reason := txStatusOK(resp)
	if p.Debug {
		fmt.Println("== tx status ==")
		if ok {
			fmt.Println("status: success")
		} else {
			fmt.Printf("status: failure\nreason: %s\n", reason)
		}
	}

	if !ok {
		return nil, fmt.Errorf("approve_offer aborted: %s", reason)
	}
	return resp, nil
}
