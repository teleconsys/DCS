package whitelist

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	suitypes "github.com/coming-chat/go-sui/v2/types"

	"github.com/teleconsys/DCS/internal/rebased"
)

type AddParams struct {
	WhitelistID   string
	Member        string
	PackageID     string
	GasID         string
	GasBudget     uint64
	RPCURL        string
	SignerAddress string
	PrivateKey    string
}

func AddToWhitelist(ctx context.Context, p AddParams) (out []byte, already bool, err error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, false, fmt.Errorf("rpc dial failed: %w", err)
	}

	// Skip if address is already whitelisted
	found, err := HasAddress(ctx, HasParams{WhitelistID: p.WhitelistID, Member: p.Member, RPCURL: p.RPCURL})
	if err == nil && found {
		return nil, true, nil
	}

	args := []any{p.Member, p.WhitelistID}
	gasPtr := &p.GasID

	txb, err := w.UnsafeMoveCallUnsigned(
		ctx,
		p.SignerAddress,
		p.PackageID,
		"dcs",
		"add_id_to_whitelist",
		nil,
		args,
		gasPtr,
		p.GasBudget,
	)
	if err != nil {
		return nil, false, fmt.Errorf("build move call: %w", err)
	}

	rawTx := []byte(txb.TxBytes)                         // sign raw bytes
	base64Tx := base64.StdEncoding.EncodeToString(rawTx) // submit base64
	sigB64, err := rebased.SignTxBytes(ctx, rawTx, p.PrivateKey)
	if err != nil {
		return nil, false, fmt.Errorf("sign tx: %w", err)
	}

	opts := &suitypes.SuiTransactionBlockResponseOptions{
		ShowEffects:       true,
		ShowEvents:        true,
		ShowObjectChanges: true,
	}
	reqType := suitypes.ExecuteTransactionRequestType("WaitForLocalExecution")

	rsp, err := w.ExecuteTransactionBlock(ctx, base64Tx, []any{sigB64}, opts, reqType)
	if err != nil {
		return nil, false, fmt.Errorf("execute: %w", err)
	}

	b, _ := json.Marshal(rsp)
	return b, false, nil
}
