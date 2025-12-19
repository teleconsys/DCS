package whitelist

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
	"github.com/spf13/viper"

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
}

func LoadAddParams(cmd *cobra.Command, args []string) (AddParams, error) {
	var p AddParams

	p.WhitelistID = viper.GetString("dcs.whitelist_id")
	if p.WhitelistID == "" {
		p.WhitelistID = os.Getenv("DCS_WHITELIST_ID")
	}
	if p.WhitelistID == "" {
		return p, fmt.Errorf("set DCS_WHITELIST_ID env var or pass --id (0x...)")
	}

	if mFlag, _ := cmd.Flags().GetString("member"); mFlag != "" {
		p.Member = mFlag
	} else if len(args) > 0 {
		p.Member = args[0]
	}
	if p.Member == "" {
		return p, fmt.Errorf("provide address as positional arg or --member (0x...)")
	}
	p.Member = strings.ToLower(p.Member)

	p.PackageID = viper.GetString("dcs.package_id")
	if p.PackageID == "" {
		p.PackageID = os.Getenv("DCS_PACKAGE_ID")
	}
	if p.PackageID == "" {
		return p, fmt.Errorf("set DCS_PACKAGE_ID env var or --package-id (0x...)")
	}

	gasIDFlag, _ := cmd.Flags().GetString("signer-gas-id")
	p.GasID = gasIDFlag
	if p.GasID == "" {
		p.GasID = os.Getenv("GC_WALLET_GAS_ID")
	}
	if p.GasID == "" {
		return p, fmt.Errorf("set GC_WALLET_GAS_ID env var or --signer-gas-id (0x...)")
	}

	if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			p.GasBudget = v
		}
	}
	if p.GasBudget == 0 {
		p.GasBudget = 10_000_000
	}

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

	if s := os.Getenv("GC_ADDRESS"); s != "" {
		p.SignerAddress = strings.ToLower(s)
	}
	if p.SignerAddress == "" {
		return p, fmt.Errorf("set GC_ADDRESS (0x...) for signer/fee payer")
	}

	return p, nil
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

	gcKey := os.Getenv("GC_PRIVATE_KEY")
	if gcKey == "" {
		return nil, false, fmt.Errorf("GC_PRIVATE_KEY not set")
	}

	rawTx := []byte(txb.TxBytes)                         // sign raw bytes
	base64Tx := base64.StdEncoding.EncodeToString(rawTx) // submit base64
	sigB64, err := rebased.SignTxBytes(ctx, rawTx, gcKey)
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
