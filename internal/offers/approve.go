package offers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/teleconsys/DCS/internal/rebased"
)

type ApproveOfferParams struct {
	CIDObjectID string // 0x... (CID id)
	Index       uint64 // offer index in next_epoch_offers
	ClockID     string // 0x6 on testnet

	PackageID string
	GasID     string
	GasBudget uint64
	RPCURL    string

	Signer  string // CID owner 0x...
	PrivKey string // iotaprivkey1...

	Debug bool
}

// Calls: <PackageID>::dcs::approve_offer(&mut CID, u64, &Clock, &mut TxContext)
func ApproveOffer(ctx context.Context, p ApproveOfferParams) (*suitypes.SuiTransactionBlockResponse, error) {
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
	txb, err := w.MoveCallUnsigned(ctx, p.Signer, p.PackageID, "dcs", "approve_offer", nil, args, gas, p.GasBudget)
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
