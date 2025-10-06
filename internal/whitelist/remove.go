package whitelist

//TODO: WIP: error on call probably due to the response parsing
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

type RemoveParams struct {
	WhitelistID   string
	Member        string
	PackageID     string
	GasID         string
	GasBudget     uint64
	RPCURL        string
	SignerAddress string
}

func LoadRemoveParams(cmd *cobra.Command, args []string) (RemoveParams, error) {
	var p RemoveParams

	// whitelist id
	p.WhitelistID = viper.GetString("dcs.whitelist_id")
	if p.WhitelistID == "" {
		p.WhitelistID = os.Getenv("DCS_WHITELIST_ID")
	}
	if p.WhitelistID == "" {
		return p, fmt.Errorf("set DCS_WHITELIST_ID env var or pass --id (0x...)")
	}

	// member
	if mFlag, _ := cmd.Flags().GetString("member"); mFlag != "" {
		p.Member = mFlag
	} else if len(args) > 0 {
		p.Member = args[0]
	}
	if p.Member == "" {
		return p, fmt.Errorf("provide address as positional arg or --member (0x...)")
	}
	p.Member = strings.ToLower(p.Member)

	// package id
	p.PackageID = viper.GetString("dcs.package_id")
	if p.PackageID == "" {
		p.PackageID = os.Getenv("DCS_PACKAGE_ID")
	}
	if p.PackageID == "" {
		return p, fmt.Errorf("set DCS_PACKAGE_ID env var or --package-id (0x...)")
	}

	// gas id
	p.GasID = os.Getenv("WALLET_GAS_ID")
	if p.GasID == "" {
		return p, fmt.Errorf("set WALLET_GAS_ID env var or --gas (0x...)")
	}

	// gas budget
	if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			p.GasBudget = v
		}
	}
	if p.GasBudget == 0 {
		p.GasBudget = 10_000_000
	}

	// RPC URL
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

	// signer address
	if s := os.Getenv("GC_ADDRESS"); s != "" {
		p.SignerAddress = strings.ToLower(s)
	}
	if p.SignerAddress == "" {
		return p, fmt.Errorf("set GC_ADDRESS (0x...) for signer/fee payer")
	}

	return p, nil
}

// RemoveFromWhitelist runs fully via RPC: Has -> MoveCallUnsigned -> Sign -> ExecuteTransactionBlock.
func RemoveFromWhitelist(ctx context.Context, p RemoveParams) (out []byte, notPresent bool, err error) {
	// 1) Dial
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, false, fmt.Errorf("rpc dial failed: %w", err)
	}

	// 2) Presence check
	found, err := HasAddress(ctx, HasParams{WhitelistID: p.WhitelistID, Member: p.Member, RPCURL: p.RPCURL})
	if err == nil && !found {
		return nil, true, nil // nothing to do
	}

	// 3) Build unsigned
	txb, err := w.MoveCallUnsigned(
		ctx,
		p.SignerAddress,
		p.PackageID,
		"dcs",
		"remove_id_from_whitelist",
		nil,
		[]any{p.Member, p.WhitelistID},
		&p.GasID,
		p.GasBudget,
	)
	if err != nil {
		return nil, false, fmt.Errorf("build move call: %w", err)
	}

	// 4) Sign
	if SignTx == nil {
		return nil, false, fmt.Errorf("no signer configured: set whitelist.SignTx")
	}
	base64Tx := base64.StdEncoding.EncodeToString([]byte(txb.TxBytes)) // <-- convert to base64 string
	sig, err := SignTx(ctx, base64Tx)
	if err != nil {
		return nil, false, fmt.Errorf("sign tx: %w", err)
	}

	// 5) Submit
	rsp, err := w.ExecuteTransactionBlock(
		ctx,
		base64Tx,
		[]any{sig},
		&suitypes.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowEvents:        true,
			ShowObjectChanges: true,
		},
		suitypes.ExecuteTransactionRequestType("WaitForLocalExecution"),
	)
	if err != nil {
		return nil, false, fmt.Errorf("execute: %w", err)
	}

	// 6) Return JSON
	b, _ := json.Marshal(rsp)
	return b, false, nil
}
