package cid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/spf13/cobra"
	"github.com/teleconsys/DCS/internal/rebased"
)

type WithdrawParams struct {
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

// firstNonEmpty returns the first non-empty string from the provided values
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// txStatusOK inspects effects status across SDK JSON shapes.
// Returns (ok, reason). ok==true when status == "success".
func txStatusOK(resp *suitypes.SuiTransactionBlockResponse) (bool, string) {
	if resp == nil || resp.Effects == nil {
		return false, "no effects in response"
	}
	b, _ := json.Marshal(resp.Effects)

	// Shape A: effects.Data.v1.status
	var a struct {
		Data struct {
			V1 struct {
				Status struct {
					Status string `json:"status"`
					Error  string `json:"error"`
				} `json:"status"`
			} `json:"v1"`
		} `json:"Data"`
	}
	if json.Unmarshal(b, &a) == nil && a.Data.V1.Status.Status != "" {
		return a.Data.V1.Status.Status == "success", a.Data.V1.Status.Error
	}

	// Shape B: effects.data.status
	var bshape struct {
		Data struct {
			Status struct {
				Status string `json:"status"`
				Error  string `json:"error"`
			} `json:"status"`
		} `json:"data"`
	}
	if json.Unmarshal(b, &bshape) == nil && bshape.Data.Status.Status != "" {
		return bshape.Data.Status.Status == "success", bshape.Data.Status.Error
	}

	// Shape C: effects.status
	var c struct {
		Status struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"status"`
	}
	if json.Unmarshal(b, &c) == nil && c.Status.Status != "" {
		return c.Status.Status == "success", c.Status.Error
	}

	// Unknown shape: assume success; caller should still verify state if needed.
	return true, ""
}

// LoadWithdrawParams loads parameters from command flags and environment variables
func LoadWithdrawParams(ctx context.Context, cmd *cobra.Command, args []string) (WithdrawParams, error) {
	var p WithdrawParams

	// Get CID from flag
	cidArg, _ := cmd.Flags().GetString("cid")
	if cidArg == "" {
		return p, fmt.Errorf("--cid is required (object id or CID string)")
	}

	// Get CID type
	cidType, _ := cmd.Flags().GetString("cid-type")
	if cidType == "" {
		cidType = "id" // default
	}
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
		id, err := GetCIDIdFromList(ctx, cidArg, rpc)
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

	// Get signer address: flags > OWNER_* > USER_* > GC_*
	flagSigner, _ := cmd.Flags().GetString("signer-address")
	signer := firstNonEmpty(
		flagSigner,
		os.Getenv("PROVIDER_ADDRESS"),
	)
	if signer == "" {
		return p, fmt.Errorf("missing signer address (set --signer-address or OWNER_ADDRESS / USER_ADDRESS / GC_ADDRESS)")
	}
	p.Signer = signer

	// Get private key: flags > OWNER_* > USER_* > GC_*
	flagPrivKey, _ := cmd.Flags().GetString("signer-private-key")
	privKey := firstNonEmpty(
		flagPrivKey,
		os.Getenv("PROVIDER_PRIVATE_KEY"),
	)
	if privKey == "" {
		return p, fmt.Errorf("missing private key (set --signer-private-key or OWNER_PRIVATE_KEY / USER_PRIVATE_KEY / GC_PRIVATE_KEY)")
	}
	p.PrivKey = privKey

	// Get gas coin ID: flags > OWNER_* > USER_* > WALLET_*
	flagGasID, _ := cmd.Flags().GetString("gas-id")
	gasID := firstNonEmpty(
		flagGasID,
		os.Getenv("PROVIDER_GAS_COIN_ID"),
	)
	if gasID == "" {
		return p, fmt.Errorf("missing gas coin id (set --gas-id or OWNER_GAS_COIN_ID / USER_GAS_COIN_ID / WALLET_GAS_ID)")
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

// Calls: <PackageID>::dcs::withdraw_payment(&mut CID, u64, &Clock, &mut TxContext)
func Withdraw(ctx context.Context, p WithdrawParams) (*suitypes.SuiTransactionBlockResponse, error) {
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
		fmt.Println("== withdraw_payment args ==")
		fmt.Printf("CID: %s\n", p.CIDObjectID)
		fmt.Printf("Index: %s\n", idxStr)
		fmt.Printf("ClockID: %s\n", p.ClockID)
		fmt.Printf("Signer: %s\n", p.Signer)
		fmt.Printf("PrivKey: %s\n", p.PrivKey)
		fmt.Printf("GasID: %s\n", p.GasID)
		fmt.Printf("GasBudget: %d\n", p.GasBudget)
	}

	gas := &p.GasID
	txb, err := w.MoveCallUnsigned(
		ctx,
		p.Signer,
		p.PackageID,
		"dcs",
		"withdraw_payment",
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
		return nil, fmt.Errorf("withdraw_payment aborted: %s", reason)
	}
	return resp, nil
}
