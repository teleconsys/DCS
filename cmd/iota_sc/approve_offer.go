package iota_sc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/spf13/cobra"

	cidlib "github.com/teleconsys/DCS/internal/cid"
	"github.com/teleconsys/DCS/internal/offers"
	"github.com/teleconsys/DCS/internal/rebased"
)

func NewApproveOfferCmd() *cobra.Command {
	var (
		flagSigner  string
		flagPrivKey string
		flagGasID   string
		flagIdx     uint64
		flagDebug   bool
	)

	cmd := &cobra.Command{
		Use:   "approve_offer",
		Short: "Approve a provider offer in next_epoch_offers (CID owner)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cidArg, _ := cmd.Flags().GetString("cid")
			cidType, _ := cmd.Flags().GetString("cid-type") // id|cid
			if cidArg == "" {
				return fmt.Errorf("--cid is required (object id or CID string)")
			}
			if cidType != "id" && cidType != "cid" {
				return fmt.Errorf("--cid-type must be 'id' or 'cid'")
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

			//TODO: Get address from sender and other info from prompt?
			// flags > OWNER_* > USER_* > GC_*
			signer := firstNonEmpty(
				flagSigner,
				os.Getenv("USER_ADDRESS"),
			)
			if signer == "" {
				return fmt.Errorf("missing signer address (set --signer-address or OWNER_ADDRESS / USER_ADDRESS / GC_ADDRESS)")
			}
			privKey, err := ResolvePrivateKey(flagPrivKey)
			if err != nil {
				return err
			}

			gasID := firstNonEmpty(
				flagGasID,
				os.Getenv("USER_GAS_COIN_ID"),
			)
			if gasID == "" {
				return fmt.Errorf("missing gas coin id (set --gas-id or OWNER_GAS_COIN_ID / USER_GAS_COIN_ID / WALLET_GAS_ID)")
			}

			p := offers.ApproveOfferParams{
				CIDObjectID: cidObjectID,
				Index:       flagIdx,
				ClockID:     getenv("DCS_CLOCK_ID", "0x6"),
				PackageID:   must("DCS_PACKAGE_ID"),
				GasID:       gasID,
				GasBudget:   getEnvAsUint64("WALLET_GAS_BUDGET", 10_000_000),
				RPCURL:      rpc,
				Signer:      signer,
				PrivKey:     privKey,
				Debug:       flagDebug,
			}

			if flagDebug {
				fmt.Println("== preflight ==")
				fmt.Printf("signer=%s  gas=%s  idx=%d\n", p.Signer, p.GasID, p.Index)
				if err := debugCIDWindowApprove(cmd.Context(), rpc, cidObjectID); err != nil {
					fmt.Printf("preflight(cid): %v\n", err)
				}
				if err := debugCIDOfferLens(cmd.Context(), rpc, cidObjectID, "before"); err != nil {
					fmt.Printf("preflight(offers): %v\n", err)
				}
			}

			resp, err := offers.ApproveOffer(context.Background(), p)
			if err != nil {
				return err
			}

			cmd.Println("✅ offer approved")
			cmd.Printf("digest: %s\n", resp.Digest)

			if flagDebug {
				if err := debugTxStatus(resp); err != nil {
					fmt.Printf("after(tx): %v\n", err)
				}
				if err := debugCIDOfferLens(cmd.Context(), rpc, cidObjectID, "after"); err != nil {
					fmt.Printf("after(offers): %v\n", err)
				}
				if err := debugShowOffer(cmd.Context(), rpc, cidObjectID, p.Index); err != nil {
					fmt.Printf("after(offer[%d]): %v\n", p.Index, err)
				}
			}
			return nil
		},
	}

	cmd.Flags().String("cid", "", "CID (object id 0x... or CID string)")
	cmd.Flags().String("cid-type", "id", "interpret --cid as 'id' or 'cid'")
	cmd.Flags().Uint64Var(&flagIdx, "idx", 0, "Offer index in next_epoch_offers to approve (0-based)")
	cmd.Flags().BoolVar(&flagDebug, "debug", false, "Verbose debug")

	// optional overrides
	cmd.Flags().StringVar(&flagSigner, "signer-address", "", "Signer address (0x...) overrides env")
	cmd.Flags().StringVar(&flagPrivKey, "signer-private-key", "", "Signer private key (iotaprivkey1...); if omitted you will be promped to insert it")
	cmd.Flags().StringVar(&flagGasID, "gas-id", "", "Gas coin object id (0x...) overrides env")

	_ = cmd.MarkFlagRequired("cid")
	return cmd
}

// Approve window helper: now must be >= next_epoch_start + 10m
func debugCIDWindowApprove(ctx context.Context, rpc, cid string) error {
	w, err := rebased.Dial(rpc)
	if err != nil {
		return err
	}
	o, err := w.GetObject(ctx, cid, suitypes.SuiObjectDataOptions{ShowContent: true, ShowType: true})
	if err != nil {
		return err
	}
	b, _ := json.Marshal(o)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	fields := findFields(m)
	if fields == nil {
		return fmt.Errorf("cannot read CID fields")
	}
	nextStart := toInt64(fields["next_epoch_start"])
	curEnd := toInt64(fields["current_epoch_end"])
	now := time.Now().UnixMilli()
	allowFrom := nextStart + 600_000

	fmt.Println("== approve window (debug) ==")
	fmt.Printf("now_ms=%d  current_end=%d  next_start=%d  allow_from=%d  ok=%v\n",
		now, curEnd, nextStart, allowFrom, now >= allowFrom)
	return nil
}
