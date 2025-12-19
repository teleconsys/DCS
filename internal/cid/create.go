package cid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/teleconsys/DCS/internal/rebased"
	"github.com/teleconsys/DCS/internal/wallet"
)

type CreateParams struct {
	CID               string
	EpochStart        uint64
	EpochEnd          uint64
	Amount            uint64
	WhitelistID       string
	CIDListID         string
	PackageID         string
	GasID             string
	GasBudget         uint64
	RPCURL            string
	UserSignerAddress string
	UserPrivateKey    string
}

type GasCoinParams struct {
	RPCURL            string
	GasID             string
	GasBudget         uint64
	UserSignerAddress string
	UserPrivateKey    string
}

func (p CreateParams) GasCoinConfig() GasCoinParams {
	return GasCoinParams{
		RPCURL:            p.RPCURL,
		GasID:             p.GasID,
		GasBudget:         p.GasBudget,
		UserSignerAddress: p.UserSignerAddress,
		UserPrivateKey:    p.UserPrivateKey,
	}
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

	// Get amount from flag
	amount, _ := cmd.Flags().GetUint64("amount")
	if amount == 0 {
		amount = 100000 // default value
	}
	p.Amount = amount

	// Auto-compute epochs when user passes 0 (no pre-computation needed)
	if p.EpochStart == 0 || p.EpochEnd == 0 {
		now := uint64(time.Now().UnixMilli())

		// Defaults: start in +30 minutes, end 20 minutes after start.
		// (This gives you ~40 minutes from now to submit offers:
		//  window is open until start + 10 minutes per the Move guard.)
		const startOffsetMinutes = 2
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

	// Read private key
	privKeyFlag, _ := cmd.Flags().GetString("signer-private-key")
	privKey, err := wallet.ResolvePrivateKey(privKeyFlag)
	if err != nil {
		return p, err
	}
	p.UserPrivateKey = privKey

	// Resolve signer address (derive from private key, compare with flag/env, confirm if mismatch)
	signerFlag, _ := cmd.Flags().GetString("signer-address")
	signer, err := wallet.ResolveSignerAddress(privKey, signerFlag, "ACTIVE_ADDRESS", "USER_ADDRESS")
	if err != nil {
		return p, err
	}
	p.UserSignerAddress = signer

	// Resolve gas coin ID and verify ownership
	gasIDFlag, _ := cmd.Flags().GetString("signer-gas-id")
	gasID, err := wallet.ResolveGasCoinId(cmd.Context(), gasIDFlag, signer, p.RPCURL, "ACTIVE_GAS_COIN_ID", "USER_GAS_COIN_ID")
	if err != nil {
		return p, err
	}
	p.GasID = gasID

	if s := os.Getenv("WALLET_GAS_BUDGET"); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			p.GasBudget = v
		}
	}
	if p.GasBudget == 0 {
		p.GasBudget = 10_000_000 // default
	}

	return p, nil
}

// CreateGasCoin creates a new gas coin with payIota function
func CreateGasCoin(ctx context.Context, p GasCoinParams, amount uint64) (string, error) {
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return "", fmt.Errorf("error RPC dial failed: %w", err)
	}

	// Build unsigned transaction
	txb, err := w.PayIotaUnsigned(ctx,
		p.UserSignerAddress,           // signer
		[]string{p.GasID},             // input_coins
		[]string{p.UserSignerAddress}, // recipients (pay to yourself)
		[]string{fmt.Sprintf("%d", amount)}, // amounts
		p.GasBudget, // gas_budget
	)

	if err != nil {
		return "", fmt.Errorf("error building the transaction: %w", err)
	}

	// Sign transaction
	rawTx := []byte(txb.TxBytes)
	base64Tx := base64.StdEncoding.EncodeToString(rawTx)
	sigB64, err := rebased.SignTxBytes(ctx, rawTx, p.UserPrivateKey)
	if err != nil {
		return "", fmt.Errorf("error signing the transaction: %w", err)
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
		return "", fmt.Errorf("error executing the transaction: %w", err)
	}

	// Extract the new Coin ID from the response
	newCoinID, err := extractNewCoinIdFromResponse(rsp)
	if err != nil {
		return "", fmt.Errorf("error extracting the new gas coin ID: %w", err)
	}

	return newCoinID, nil
}

func SplitCoinDummy(ctx context.Context, p CreateParams, coinID string, amount int64) (string, error) {
	return "0xfaced597f3fad647f331f9215711adbac7fa5cbd462456bf1908e5e7c8743c6a", nil
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
	txb, err := w.UnsafeMoveCallUnsigned(
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
	txb, err := w.UnsafeMoveCallUnsigned(
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

// extractNewCoinIdFromResponse extracts the new coin ID from payIota transaction response
func extractNewCoinIdFromResponse(rsp *suitypes.SuiTransactionBlockResponse) (string, error) {
	if len(rsp.ObjectChanges) == 0 {
		return "", fmt.Errorf("no object changes found in transaction response")
	}

	// Search in the ObjectChanges array for objects "created" with type Coin
	for _, change := range rsp.ObjectChanges {
		// Marshal to access the fields
		changeJSON, err := json.Marshal(change)
		if err != nil {
			continue
		}

		var changeMap map[string]interface{}
		if err := json.Unmarshal(changeJSON, &changeMap); err != nil {
			continue
		}

		// Search for the "Data": {"created": {...}} structure
		if data, ok := changeMap["Data"].(map[string]interface{}); ok {
			if created, ok := data["created"].(map[string]interface{}); ok {
				// Verify that it is a Coin
				if objectType, ok := created["objectType"].(string); ok {
					if strings.Contains(objectType, "Coin") {
						if objectId, ok := created["objectId"].(string); ok {
							return objectId, nil
						}
					}
				}
			}
		}
	}

	// Fallback: search with regex in the marshaled JSON
	b, err := json.Marshal(rsp.ObjectChanges)
	if err != nil {
		return "", fmt.Errorf("failed to marshal object changes: %w", err)
	}

	// Pattern to search for "created" with objectId
	re := regexp.MustCompile(`"created"\s*:\s*\{[^}]*"objectType"\s*:\s*"[^"]*Coin[^"]*"[^}]*"objectId"\s*:\s*"(0x[a-fA-F0-9]{64})"`)
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

	// Search in the ObjectChanges array for objects "created" with type CID and owner Shared
	for _, change := range rsp.ObjectChanges {
		// Marshal to access the fields
		changeJSON, err := json.Marshal(change)
		if err != nil {
			continue
		}

		var changeMap map[string]interface{}
		if err := json.Unmarshal(changeJSON, &changeMap); err != nil {
			continue
		}

		// Search for the "Data": {"created": {...}} structure
		if data, ok := changeMap["Data"].(map[string]interface{}); ok {
			if created, ok := data["created"].(map[string]interface{}); ok {
				// Verify that it is a CID object
				if objectType, ok := created["objectType"].(string); ok {
					if strings.Contains(objectType, "CID") {
						// Verify that it has Shared owner (indicating it was shared)
						if owner, ok := created["owner"].(map[string]interface{}); ok {
							if shared, ok := owner["Shared"].(map[string]interface{}); ok && shared != nil {
								if objectId, ok := created["objectId"].(string); ok {
									return objectId, nil
								}
							}
						}
					}
				}
			}
		}
	}

	// Fallback: search with regex in the marshaled JSON
	b, err := json.Marshal(rsp.ObjectChanges)
	if err != nil {
		return "", fmt.Errorf("failed to marshal object changes: %w", err)
	}

	// Pattern to search for "created" with objectId and type containing CID, with Shared owner
	re := regexp.MustCompile(`"created"\s*:\s*\{[^}]*"objectType"\s*:\s*"[^"]*CID[^"]*"[^}]*"owner"\s*:\s*\{[^}]*"Shared"[^}]*\}[^}]*"objectId"\s*:\s*"(0x[a-fA-F0-9]{64})"`)
	matches := re.FindStringSubmatch(string(b))
	if len(matches) > 1 {
		return matches[1], nil
	}

	// If no created CID object with Shared owner is found, the user was not whitelisted
	return "", fmt.Errorf("no CID object found in transaction response: user may not be whitelisted")
}
