package iota_sc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	cidlib "github.com/teleconsys/DCS/internal/cid"
	"github.com/teleconsys/DCS/internal/offers"
	"github.com/teleconsys/DCS/internal/rebased"
)

func NewSubmitOfferCmd() *cobra.Command {
	var (
		flagSigner  string
		flagPrivKey string
		flagGasID   string
		flagDebug   bool
	)

	cmd := &cobra.Command{
		Use:   "submit_offer",
		Short: "Submit an offer for the next epoch (provider wallet)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cidArg, _ := cmd.Flags().GetString("cid")
			cidType, _ := cmd.Flags().GetString("cid-type") // id|cid
			amount, _ := cmd.Flags().GetUint64("amount")

			if cidArg == "" {
				return fmt.Errorf("--cid is required (object id or CID string)")
			}
			if cidType != "id" && cidType != "cid" {
				return fmt.Errorf("--cid-type must be 'id' or 'cid'")
			}
			if amount == 0 {
				return fmt.Errorf("--amount must be > 0 (IOTA nanos)")
			}

			rpc := getenv("REBASE_RPC", "https://api.testnet.iota.cafe:443")

			cidObjectID := cidArg
			if cidType == "cid" {
				id, err := cidlib.GetCIDIdFromList(cmd.Context(), cidArg, rpc)
				if err != nil {
					return err
				}
				cidObjectID = id
			}

			// flags > PROVIDER_* > GC_* > USER_*
			signer := firstNonEmpty(
				flagSigner,
				os.Getenv("PROVIDER_ADDRESS"),
				os.Getenv("GC_ADDRESS"),
				os.Getenv("USER_ADDRESS"),
			)
			if signer == "" {
				return fmt.Errorf("missing signer address (set --signer-address or PROVIDER_ADDRESS / GC_ADDRESS / USER_ADDRESS)")
			}
			privKey := firstNonEmpty(
				flagPrivKey,
				os.Getenv("PROVIDER_PRIVATE_KEY"),
				os.Getenv("GC_PRIVATE_KEY"),
				os.Getenv("USER_PRIVATE_KEY"),
			)
			if privKey == "" {
				return fmt.Errorf("missing private key (set --signer-private-key or PROVIDER_PRIVATE_KEY / GC_PRIVATE_KEY / USER_PRIVATE_KEY)")
			}
			gasID := firstNonEmpty(
				flagGasID,
				os.Getenv("PROVIDER_GAS_COIN_ID"),
				os.Getenv("WALLET_GAS_ID"),
				os.Getenv("USER_GAS_COIN_ID"),
			)
			if gasID == "" {
				return fmt.Errorf("missing gas coin id (set --gas-id or PROVIDER_GAS_COIN_ID / WALLET_GAS_ID / USER_GAS_COIN_ID)")
			}

			p := offers.SubmitOfferParams{
				CIDObjectID: cidObjectID,
				Amount:      amount,
				WhitelistID: must("DCS_WHITELIST_ID"),
				ClockID:     getenv("DCS_CLOCK_ID", "0x6"),
				PackageID:   must("DCS_PACKAGE_ID"),
				GasID:       gasID,
				GasBudget:   getEnvAsUint64("WALLET_GAS_BUDGET", 10_000_000),
				RPCURL:      rpc,
				Signer:      signer,
				PrivKey:     privKey,
				Debug:       flagDebug,
			}

			// ========= DEBUG: preflight =========
			if flagDebug {
				fmt.Println("== preflight ==")
				fmt.Printf("signer=%s  gas=%s  amount(nanos)=%d\n", p.Signer, p.GasID, p.Amount)
				if err := debugCIDWindow(cmd.Context(), rpc, cidObjectID); err != nil {
					fmt.Printf("preflight(cid): %v\n", err)
				}
				if err := debugCIDOfferLens(cmd.Context(), rpc, cidObjectID, "before"); err != nil {
					fmt.Printf("preflight(offers): %v\n", err)
				}
			}

			resp, err := offers.SubmitReplicaOffer(context.Background(), p)
			if err != nil {
				return err
			}

			cmd.Println("✅ offer submitted")
			cmd.Printf("digest: %s\n", resp.Digest)

			// ========= DEBUG: afterflight =========
			if flagDebug {
				if err := debugTxStatus(resp); err != nil {
					fmt.Printf("after(tx): %v\n", err)
				}
				// re-read CID
				if err := debugCIDOfferLens(cmd.Context(), rpc, cidObjectID, "after"); err != nil {
					fmt.Printf("after(offers): %v\n", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().String("cid", "", "CID (object id 0x... or CID string)")
	cmd.Flags().String("cid-type", "id", "interpret --cid as 'id' or 'cid'")
	cmd.Flags().Uint64("amount", 0, "Offer amount (IOTA nanos)")
	cmd.Flags().BoolVar(&flagDebug, "debug", false, "Verbose debug (preflight + postflight)")

	// optional overrides
	cmd.Flags().StringVar(&flagSigner, "signer-address", "", "Signer address (0x...) overrides env")
	cmd.Flags().StringVar(&flagPrivKey, "signer-private-key", "", "Signer private key (iotaprivkey1...) overrides env")
	cmd.Flags().StringVar(&flagGasID, "gas-id", "", "Gas coin object id (0x...) overrides env")

	_ = cmd.MarkFlagRequired("cid")
	_ = cmd.MarkFlagRequired("amount")
	return cmd
}

// ---------- debug helpers ----------

func debugTxStatus(resp *suitypes.SuiTransactionBlockResponse) error {
	// effects status/error are inside a TagJson wrapper in some SDK versions; just print the whole resp for clarity
	fmt.Println("== tx effects/events (debug) ==")
	b, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(b))
	return nil
}

func debugCIDWindow(ctx context.Context, rpc, cid string) error {
	w, err := rebased.Dial(rpc)
	if err != nil {
		return err
	}
	o, err := w.GetObject(ctx, cid, suitypes.SuiObjectDataOptions{ShowContent: true, ShowType: true})
	if err != nil {
		return err
	}

	// parse via JSON to avoid SDK shape drift
	b, _ := json.Marshal(o)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	fields := getMap(m, "data", "content", "fields")
	if fields == nil {
		return fmt.Errorf("cannot read CID fields")
	}
	now := time.Now().UnixMilli()
	nextStart := toInt64(fields["next_epoch_start"])
	nextEnd := toInt64(fields["next_epoch_end"])
	windowClose := nextStart + 600_000

	fmt.Println("== epoch window (debug) ==")
	fmt.Printf("now_ms=%d  next_start=%d  next_end=%d  window_closes=%d\n", now, nextStart, nextEnd, windowClose)
	fmt.Printf("in_window=%v  (now >= current_end && now < next_start+10m)\n", now < windowClose)
	return nil
}

func debugCIDOfferLens(ctx context.Context, rpc, cid, label string) error {
	w, err := rebased.Dial(rpc)
	if err != nil {
		return err
	}
	o, err := w.GetObject(ctx, cid, suitypes.SuiObjectDataOptions{ShowContent: true})
	if err != nil {
		return err
	}
	b, _ := json.Marshal(o)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	fields := getMap(m, "data", "content", "fields")
	if fields == nil {
		return fmt.Errorf("cannot read CID fields")
	}
	nNext := len(getSlice(fields["next_epoch_offers"]))
	nCur := len(getSlice(fields["current_epoch_offers"]))
	nPrev := len(getSlice(fields["prev_epoch_offers"]))

	fmt.Printf("== offers %s ==\n", label)
	fmt.Printf("next=%d  current=%d  prev=%d\n", nNext, nCur, nPrev)
	return nil
}

// tiny JSON navigation helpers (robust to type drift)
func getMap(m map[string]any, path ...string) map[string]any {
	cur := any(m)
	for _, k := range path {
		asMap, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = asMap[k]
		if !ok {
			return nil
		}
	}
	res, _ := cur.(map[string]any)
	return res
}
func getSlice(v any) []any {
	if v == nil {
		return nil
	}
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}
func toInt64(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case string:
		i, _ := strconv.ParseInt(x, 10, 64)
		return i
	default:
		return 0
	}
}

// ---------- util ----------

func getEnvAsUint64(key string, def uint64) uint64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return def
	}
	return i
}
func must(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing required env: " + k)
	}
	return v
}
func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
