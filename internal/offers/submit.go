package offers

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/spf13/cobra"
	"github.com/teleconsys/DCS/internal/rebased"
	"github.com/teleconsys/DCS/internal/wallet"
)

type SubmitOfferParams struct {
	// target
	CIDObjectID string // 0x... (CID object id)
	Amount      uint64 // IOTA nanos

	// shared objects
	WhitelistID string // 0x... (&Whitelist)
	ClockID     string // 0x6 on testnet (&Clock)

	// execution
	PackageID string // 0x... (Move package)
	GasID     string // 0x... (signer’s gas coin)
	GasBudget uint64
	RPCURL    string

	// signer (provider)
	Signer  string // 0x...
	PrivKey string // iotaprivkey1...

	// debug
	Debug bool
}

// LoadSubmitOfferParams loads parameters from command flags and environment variables
func LoadSubmitOfferParams(ctx context.Context, cmd *cobra.Command, args []string) (SubmitOfferParams, error) {
	var p SubmitOfferParams

	// Get CID from flag
	cidArg, _ := cmd.Flags().GetString("cid")
	if cidArg == "" {
		return p, fmt.Errorf("--cid is required (object id 0x...)")
	}

	// Use the flag value directly as CID object ID
	p.CIDObjectID = cidArg

	// Get amount from flag
	amount, _ := cmd.Flags().GetUint64("amount")
	if amount == 0 {
		return p, fmt.Errorf("--amount must be > 0 (IOTA nanos)")
	}
	p.Amount = amount

	// Get RPC URL
	rpc := os.Getenv("REBASE_RPC")
	if rpc == "" {
		rpc = "https://api.testnet.iota.cafe:443"
	}
	p.RPCURL = rpc

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
	signer, err := wallet.ResolveSignerAddress(privKey, signerFlag, "ACTIVE_ADDRESS", "PROVIDER_ADDRESS")
	if err != nil {
		return p, err
	}
	p.Signer = signer

	// Resolve gas coin ID and verify ownership
	gasIDFlag, _ := cmd.Flags().GetString("signer-gas-id")
	gasID, err := wallet.ResolveGasCoinId(ctx, gasIDFlag, signer, p.RPCURL, "ACTIVE_GAS_COIN_ID", "PROVIDER_GAS_COIN_ID")
	if err != nil {
		return p, err
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

	// Get Whitelist ID
	whitelistID := os.Getenv("DCS_WHITELIST_ID")
	if whitelistID == "" {
		return p, fmt.Errorf("missing DCS_WHITELIST_ID env var")
	}
	p.WhitelistID = whitelistID

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

// SubmitReplicaOffer builds, signs, executes:
//
//	<PackageID>::dcs::create_offer(&mut CID, u64, &Whitelist, &Clock, &mut TxContext)
func SubmitReplicaOffer(ctx context.Context, p SubmitOfferParams) (*suitypes.SuiTransactionBlockResponse, error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial: %w", err)
	}

	// u64 must be sent as a decimal string for this RPC
	amountStr := strconv.FormatUint(p.Amount, 10)

	args := []any{
		p.CIDObjectID, // &mut CID (shared)
		amountStr,     // u64 as string
		p.WhitelistID, // &Whitelist (shared)
		p.ClockID,     // &Clock (shared)
	}

	if p.Debug {
		fmt.Println("== create_offer args ==")
		fmt.Printf("CID: %s\n", p.CIDObjectID)
		fmt.Printf("Amount (nanos): %s\n", amountStr)
		fmt.Printf("WhitelistID: %s\n", p.WhitelistID)
		fmt.Printf("ClockID: %s\n", p.ClockID)
	}

	gas := &p.GasID
	txb, err := w.UnsafeMoveCallUnsigned(ctx, p.Signer, p.PackageID, "dcs", "create_offer", nil, args, gas, p.GasBudget)
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
		return nil, fmt.Errorf("submit_offer aborted: %s", reason)
	}

	return resp, nil
}
