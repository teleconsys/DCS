package offers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/teleconsys/DCS/internal/rebased"
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

// SubmitReplicaOffer builds, signs, executes:
//
//	<PackageID>::dcs::create_offer(&mut CID, u64, &Whitelist, &Clock, &mut TxContext)
func SubmitReplicaOffer(ctx context.Context, p SubmitOfferParams) (*suitypes.SuiTransactionBlockResponse, error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial: %w", err)
	}

	// IMPORTANT: u64 must be sent as a decimal string for this RPC
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
	txb, err := w.MoveCallUnsigned(ctx, p.Signer, p.PackageID, "dcs", "create_offer", nil, args, gas, p.GasBudget)
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

// txStatusOK extracts effects.status from either:
// 1) TagJson wrapper: { "data": { "status": { "status":"success|failure","error":"..." } } }
// 2) Direct effects object: { "status": { ... } }
func txStatusOK(resp *suitypes.SuiTransactionBlockResponse) (bool, string) {
	if resp == nil || resp.Effects == nil {
		return false, "no effects in response"
	}
	b, _ := json.Marshal(resp.Effects)

	// Shape 1: TagJson wrapper
	var w1 struct {
		Data struct {
			Status struct {
				Status string `json:"status"`
				Error  string `json:"error"`
			} `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &w1); err == nil && w1.Data.Status.Status != "" {
		return w1.Data.Status.Status == "success", w1.Data.Status.Error
	}

	// Shape 2: direct effects object
	var w2 struct {
		Status struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"status"`
	}
	if err := json.Unmarshal(b, &w2); err == nil && w2.Status.Status != "" {
		return w2.Status.Status == "success", w2.Status.Error
	}

	return true, ""
}

func PrettyJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
