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

type ApproveOfferParams struct {
	CIDObjectID string // 0x... (CID id)
	Index       uint64 // offer index in next_epoch_offers
	ClockID     string // 0x6 on testnet
	PackageID   string
	GasID       string
	GasBudget   uint64
	RPCURL      string
	Signer      string // CID owner 0x...
	PrivKey     string // iotaprivkey1...
	Debug       bool
}

// LoadApproveOfferParams loads parameters from command flags and environment variables
func LoadApproveOfferParams(ctx context.Context, cmd *cobra.Command, args []string) (ApproveOfferParams, error) {
	var p ApproveOfferParams

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

	// Resolve gas coin ID and verify ownership
	gasIDFlag, _ := cmd.Flags().GetString("signer-gas-id")
	gasID, err := wallet.ResolveGasCoinId(ctx, gasIDFlag, signer, p.RPCURL, "ACTIVE_GAS_COIN_ID", "USER_GAS_COIN_ID")
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

func ApproveOffer(ctx context.Context, p ApproveOfferParams) (*suitypes.SuiTransactionBlockResponse, error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial: %w", err)
	}

	inBudget, err := isInBudget(ctx, w, p.CIDObjectID, p.Index)
	if err != nil {
		return nil, fmt.Errorf("check budget: %w", err)
	}
	if !inBudget {
		return nil, fmt.Errorf("not enough budget to approve offer, please add some funds to the CID")
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
	txb, err := w.UnsafeMoveCallUnsigned(ctx, p.Signer, p.PackageID, "dcs", "approve_offer", nil, args, gas, p.GasBudget)
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

func isInBudget(ctx context.Context, w *rebased.Wrapper, cidId string, offerIndex uint64) (bool, error) {
	fields, err := cidlib.GetCIDFields(ctx, w, cidId)
	if err != nil {
		return false, fmt.Errorf("get CID fields: %w", err)
	}

	// Get funds.balance
	fundsAny, ok := fields["funds"]
	if !ok {
		return false, fmt.Errorf("funds field not found")
	}
	funds, ok := fundsAny.(map[string]any)
	if !ok {
		return false, fmt.Errorf("funds is not a map")
	}

	// Access balance through fields sub-object
	fieldsObj, ok := funds["fields"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("funds.fields is not a map")
	}
	balanceAny, ok := fieldsObj["balance"]
	if !ok {
		return false, fmt.Errorf("funds.fields.balance field not found")
	}
	balance := toInt64(balanceAny)
	if balance < 0 {
		return false, fmt.Errorf("invalid balance: %d", balance)
	}

	// Get the offer at the given index
	nextOffers := asSlice(fields["next_epoch_offers"])
	if nextOffers == nil {
		return false, fmt.Errorf("next_epoch_offers is nil")
	}
	if int(offerIndex) >= len(nextOffers) {
		return false, fmt.Errorf("offer index %d out of range (max: %d)", offerIndex, len(nextOffers)-1)
	}
	offerAny := nextOffers[offerIndex]
	offer, ok := offerAny.(map[string]any)
	if !ok {
		return false, fmt.Errorf("offer at index %d is not a map", offerIndex)
	}

	// Access amount through fields sub-object
	offerFields, ok := offer["fields"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("offer at index %d fields is not a map", offerIndex)
	}
	offerAmount := toInt64(offerFields["amount"])
	if offerAmount < 0 {
		return false, fmt.Errorf("invalid offer amount: %d", offerAmount)
	}

	// Calculate total of approved offers
	var total uint64
	for _, oAny := range nextOffers {
		o, ok := oAny.(map[string]any)
		if !ok {
			continue
		}

		// Access fields sub-object
		oFields, ok := o["fields"].(map[string]any)
		if !ok {
			continue
		}

		// Check if approved is true
		approved, ok := oFields["approved"].(bool)
		if !ok || !approved {
			continue
		}

		// Get amount and add to total
		amount := toInt64(oFields["amount"])
		if amount < 0 {
			continue // skip negative amounts
		}
		total += uint64(amount)
	}

	fmt.Printf("total approved offers: %d\n- CID balance: %d\n- offer amount: %d\n", total, balance, offerAmount)

	// Check if funds.balance - total >= offer
	available := uint64(balance) - total
	return available >= uint64(offerAmount), nil
}
