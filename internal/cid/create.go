package cid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/teleconsys/DCS/internal/rebased"
)

type CreateParams struct {
	CID               string
	EpochStart        uint64
	EpochEnd          uint64
	WhitelistID       string
	CIDListID         string
	PackageID         string
	GasID             string
	GasBudget         uint64
	RPCURL            string
	UserSignerAddress string
	UserPrivateKey    string
}

func LoadCreateParams(cmd *cobra.Command, args []string) (CreateParams, error) {
	var p CreateParams

	// Get CID from args (handled by caller)
	if len(args) > 0 {
		p.CID = args[0]
	}

	// Get epoch parameters from flags
	epochStart, _ := cmd.Flags().GetUint64("epoch-start")
	epochEnd, _ := cmd.Flags().GetUint64("epoch-end")
	p.EpochStart = epochStart
	p.EpochEnd = epochEnd

	// Auto-compute epochs when user passes 0 (no pre-computation needed)
	if p.EpochStart == 0 || p.EpochEnd == 0 {
		now := uint64(time.Now().UnixMilli())

		// Defaults: start in +30 minutes, end 20 minutes after start.
		// (This gives you ~40 minutes from now to submit offers:
		//  window is open until start + 10 minutes per the Move guard.)
		const startOffsetMinutes = 30
		const windowAfterStartMinutes = 20

		p.EpochStart = now + startOffsetMinutes*60*1000
		p.EpochEnd = p.EpochStart + windowAfterStartMinutes*60*1000

		// Echo so you can see what got used
		fmt.Fprintf(cmd.OutOrStdout(),
			"auto-epochs: now=%d start=%d end=%d (offset=%dmin window=%dmin)\n",
			now, p.EpochStart, p.EpochEnd, startOffsetMinutes, windowAfterStartMinutes)
	}

	// Get package ID
	p.PackageID = viper.GetString("dcs.package_id")
	if p.PackageID == "" {
		p.PackageID = os.Getenv("DCS_PACKAGE_ID")
	}
	if p.PackageID == "" {
		return p, fmt.Errorf("set DCS_PACKAGE_ID env var or pass --package-id (0x...)")
	}

	// Get whitelist ID
	p.WhitelistID = viper.GetString("dcs.whitelist_id")
	if p.WhitelistID == "" {
		p.WhitelistID = os.Getenv("DCS_WHITELIST_ID")
	}
	if p.WhitelistID == "" {
		return p, fmt.Errorf("set DCS_WHITELIST_ID env var or pass --id (0x...)")
	}

	// Get CID list ID
	p.CIDListID = viper.GetString("dcs.cidlist_id")
	if p.CIDListID == "" {
		p.CIDListID = os.Getenv("DCS_CIDLIST_ID")
	}
	if p.CIDListID == "" {
		return p, fmt.Errorf("set DCS_CIDLIST_ID env var or pass --cidlist-id (0x...)")
	}

	// Get private key: flag takes priority over env var
	if privateKeyFlag, _ := cmd.Flags().GetString("user-private-key"); privateKeyFlag != "" {
		p.UserPrivateKey = privateKeyFlag
	} else if userPrivateKeyEnv := os.Getenv("USER_PRIVATE_KEY"); userPrivateKeyEnv != "" {
		p.UserPrivateKey = userPrivateKeyEnv
	} else {
		return p, fmt.Errorf("set USER_PRIVATE_KEY env var or pass --user-private-key")
	}

	// Get user signer address
	if signerAddress, _ := cmd.Flags().GetString("user-address"); signerAddress != "" {
		p.UserSignerAddress = signerAddress
	} else if userAddressEnv := os.Getenv("USER_ADDRESS"); userAddressEnv != "" {
		p.UserSignerAddress = userAddressEnv
	} else {
		return p, fmt.Errorf("set USER_ADDRESS env var or pass --user-address")
	}

	// Get gas gas coin ID for user, this will be used to create the new COIN object for the cid creation
	p.GasID = os.Getenv("USER_GAS_COIN_ID")
	if p.GasID == "" {
		return p, fmt.Errorf("set DCS_CIDCOIN_ID env var or pass --user-coin-id (0x...)")
	}

	if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			p.GasBudget = v
		}
	}
	if p.GasBudget == 0 {
		p.GasBudget = 10_000_000 // default
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

	return p, nil
}

// splitCoin splits a coin into multiple coins with specified amounts
// func SplitCoin(ctx context.Context, p CreateParams, coinID string, amount int64) (string, error) {
// 	w, err := rebased.Dial(p.RPCURL)
// 	if err != nil {
// 		return "", fmt.Errorf("rpc dial failed: %w", err)
// 	}

// 	gasPtr := &p.GasID

// 	// Build unsigned transaction
// 	txb, err := w.SplitCoinUnsigned(
// 		ctx,
// 		p.UserSignerAddress,
// 		coinID,
// 		[]uint64{uint64(amount)},
// 		gasPtr,
// 		p.GasBudget,
// 	)	
// 	if err != nil {
// 		return "", fmt.Errorf("build move call: %w", err)
// 	}

// 	// Sign transaction
// 	rawTx := []byte(txb.TxBytes)
// 	base64Tx := base64.StdEncoding.EncodeToString(rawTx)
// 	sigB64, err := rebased.SignTxBytes(ctx, rawTx, p.UserPrivateKey)
// 	if err != nil {
// 		return "", fmt.Errorf("sign tx: %w", err)
// 	}

// 	// Execute transaction
// 	opts := &suitypes.SuiTransactionBlockResponseOptions{
// 		ShowEffects:       true,
// 		ShowEvents:        true,
// 		ShowObjectChanges: true,
// 	}
// 	reqType := suitypes.ExecuteTransactionRequestType("WaitForLocalExecution")

// 	rsp, err := w.ExecuteTransactionBlock(ctx, base64Tx, []any{sigB64}, opts, reqType)
// 	if err != nil {
// 		return "", fmt.Errorf("execute: %w", err)
// 	}

// 	fmt.Println(rsp)

// 	// Extract new coin ID from response
// 	newCoinID, err := extractNewCoinIdFromResponse(rsp)
// 	if err != nil {
// 		return "", fmt.Errorf("extract new coin ID: %w", err)
// 	}

// 	return newCoinID, nil
// }

func SplitCoin(ctx context.Context, p CreateParams, coinID string, amount int64) (string, error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return "", fmt.Errorf("rpc dial failed: %w", err)
	}

	gasPtr := &p.GasID

	// 	txb, err := w.PaySuiUnsigned(
	// 	ctx,
	// 	p.UserSignerAddress,
	// 	[]string{coinID},
	// 	[]string{p.UserSignerAddress},
	// 	[]uint64{uint64(amount)},
	// 	p.GasBudget,
	// )	
	// if err != nil {
	// 	return "", fmt.Errorf("build pay sui call: %w", err)
	// }

	// Build unsigned transaction
	txb, err := w.SplitCoinUnsignedRPC(
		ctx,
		p.UserSignerAddress,
		coinID,
		[]uint64{uint64(amount)},
		gasPtr,
		p.GasBudget,
	)	
	if err != nil {
		return "", fmt.Errorf("build move call: %w", err)
	}

	// Sign transaction
	rawTx := []byte(txb.TxBytes)
	base64Tx := base64.StdEncoding.EncodeToString(rawTx)
	sigB64, err := rebased.SignTxBytes(ctx, rawTx, p.UserPrivateKey)
	if err != nil {
		return "", fmt.Errorf("sign tx: %w", err)
	}

	// Execute transaction
	opts := &suitypes.SuiTransactionBlockResponseOptions{
		ShowEffects:       true,
		ShowEvents:        true,
		ShowObjectChanges: true,
	}
	reqType := suitypes.ExecuteTransactionRequestType("WaitForLocalExecution")

	rsp, err := w.ExecuteTransactionBlock(ctx, base64Tx, []any{sigB64}, opts, reqType)
	if err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}

	fmt.Println(rsp)

	// Extract new coin ID from response
	newCoinID, err := extractNewCoinIdFromResponse(rsp)
	if err != nil {
		return "", fmt.Errorf("extract new coin ID: %w", err)
	}

	return newCoinID, nil
}

func SplitCoinDummy(ctx context.Context, p CreateParams, coinID string, amount int64) (string, error) {
		return "0xe79a28cb2fa14816279b97301b596983ae62f22a2eac3f5adedde134e9a01e99", nil
}

func CreateCID(ctx context.Context, p CreateParams, cidCoinId string) ([]byte, string, error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, "", fmt.Errorf("rpc dial failed: %w", err)
	}

	// Build arguments for create_cid function
	args := []any{p.CID, cidCoinId, fmt.Sprintf("%d", p.EpochStart), fmt.Sprintf("%d", p.EpochEnd), p.WhitelistID}
	gasPtr := &p.GasID

	// Build unsigned transaction
	txb, err := w.MoveCallUnsigned(
		ctx,
		p.UserSignerAddress,
		p.PackageID,
		"dcs",
		"create_cid",
		nil,
		args,
		gasPtr,
		p.GasBudget,
	)
	if err != nil {
		return nil, "", fmt.Errorf("build move call: %w", err)
	}

	// Sign transaction
	rawTx := []byte(txb.TxBytes)
	base64Tx := base64.StdEncoding.EncodeToString(rawTx)
	sigB64, err := rebased.SignTxBytes(ctx, rawTx, p.UserPrivateKey)
	if err != nil {
		return nil, "", fmt.Errorf("sign tx: %w", err)
	}

	// Execute transaction
	opts := &suitypes.SuiTransactionBlockResponseOptions{
		ShowEffects:       true,
		ShowEvents:        true,
		ShowObjectChanges: true,
	}
	reqType := suitypes.ExecuteTransactionRequestType("WaitForLocalExecution")

	rsp, err := w.ExecuteTransactionBlock(ctx, base64Tx, []any{sigB64}, opts, reqType)
	if err != nil {
		return nil, "", fmt.Errorf("execute: %w", err)
	}

	// Extract CID object ID from response
	cidId, err := extractCidIdFromResponse(rsp)
	if err != nil {
		return nil, "", fmt.Errorf("extract CID object ID: %w", err)
	}

	b, _ := json.Marshal(rsp)
	return b, cidId, nil
}

func AddToCIDList(ctx context.Context, p CreateParams, cidId string) ([]byte, error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial failed: %w", err)
	}

	// Build arguments for add_to_cidlist function
	args := []any{cidId, p.CIDListID}
	gasPtr := &p.GasID

	// Build unsigned transaction
	txb, err := w.MoveCallUnsigned(
		ctx,
		p.UserSignerAddress,
		p.PackageID,
		"dcs",
		"add_to_cidlist",
		nil,
		args,
		gasPtr,
		p.GasBudget,
	)
	if err != nil {
		return nil, fmt.Errorf("build move call: %w", err)
	}

	// Sign transaction
	rawTx := []byte(txb.TxBytes)
	base64Tx := base64.StdEncoding.EncodeToString(rawTx)
	sigB64, err := rebased.SignTxBytes(ctx, rawTx, p.UserPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("sign tx: %w", err)
	}

	// Execute transaction
	opts := &suitypes.SuiTransactionBlockResponseOptions{
		ShowEffects:       true,
		ShowEvents:        true,
		ShowObjectChanges: true,
	}
	reqType := suitypes.ExecuteTransactionRequestType("WaitForLocalExecution")

	rsp, err := w.ExecuteTransactionBlock(ctx, base64Tx, []any{sigB64}, opts, reqType)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}

	b, _ := json.Marshal(rsp)

	return b, nil
}

// extractNewCoinIdFromResponse extracts the new coin ID from split transaction response
func extractNewCoinIdFromResponse(rsp *suitypes.SuiTransactionBlockResponse) (string, error) {
	if len(rsp.ObjectChanges) == 0 {
		return "", fmt.Errorf("no object changes found in transaction response")
	}

	// Marshal to JSON and extract created coin object IDs
	b, err := json.Marshal(rsp.ObjectChanges)
	if err != nil {
		return "", fmt.Errorf("failed to marshal object changes: %w", err)
	}

	// Look for created coin objects in the JSON
	// Pattern matches: "type":"created" followed by "objectType":"...Coin..." and "objectId":"0x..."
	re := regexp.MustCompile(`"type"\s*:\s*"created"[^}]*"objectType"\s*:\s*"[^"]*Coin[^"]*"[^}]*"objectId"\s*:\s*"(0x[a-fA-F0-9]{64})"`)
	matches := re.FindStringSubmatch(string(b))
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("no new coin ID found in transaction response")
}

// extractCidIdFromResponse extracts the CID object ID from transaction response
func extractCidIdFromResponse(rsp *suitypes.SuiTransactionBlockResponse) (string, error) {
	if len(rsp.ObjectChanges) == 0 {
		return "", fmt.Errorf("no object changes found in transaction response")
	}

	// Marshal to JSON and extract created object IDs
	b, err := json.Marshal(rsp.ObjectChanges)
	if err != nil {
		return "", fmt.Errorf("failed to marshal object changes: %w", err)
	}

	// Look for all object IDs in the JSON
	re := regexp.MustCompile(`"objectId"\s*:\s*"(0x[a-fA-F0-9]{64})"`)
	matches := re.FindAllStringSubmatch(string(b), -1)

	if len(matches) == 0 {
		return "", fmt.Errorf("no object ID found in transaction response")
	}

	// Return the objectId from the LAST match (last index in the array)
	lastMatch := matches[len(matches)-1]
	if len(lastMatch) > 1 {
		return lastMatch[1], nil
	}

	return "", fmt.Errorf("no CID object ID found in transaction response")
}
