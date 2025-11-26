package iota_sc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/spf13/cobra"

	"github.com/teleconsys/DCS/internal/offers"
	"github.com/teleconsys/DCS/internal/rebased"
)

func NewApproveOfferCmd() *cobra.Command {
	var (
		signerFlag  string
		privKeyFlag string
		gasIDFlag   string
		indexFlag   uint64
		debugFlag   bool
	)

	cmd := &cobra.Command{
		Use:   "approve_offer",
		Short: "Approve a provider offer in next_epoch_offers (CID owner)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := offers.LoadApproveOfferParams(cmd.Context(), cmd, args)
			if err != nil {
				return err
			}

			if p.Debug {
				fmt.Println("== preflight ==")
				fmt.Printf("signer=%s  gas=%s  idx=%d\n", p.Signer, p.GasID, p.Index)
				if err := debugCIDWindowApprove(cmd.Context(), p.RPCURL, p.CIDObjectID); err != nil {
					fmt.Printf("preflight(cid): %v\n", err)
				}
				if err := debugCIDOfferLens(cmd.Context(), p.RPCURL, p.CIDObjectID, "before"); err != nil {
					fmt.Printf("preflight(offers): %v\n", err)
				}
			}

			resp, err := offers.ApproveOffer(context.Background(), p)
			if err != nil {
				return err
			}

			cmd.Println("✅ offer approved")
			cmd.Printf("digest: %s\n", resp.Digest)

			if p.Debug {
				if err := debugTxStatus(resp); err != nil {
					fmt.Printf("after(tx): %v\n", err)
				}
				if err := debugCIDOfferLens(cmd.Context(), p.RPCURL, p.CIDObjectID, "after"); err != nil {
					fmt.Printf("after(offers): %v\n", err)
				}
				if err := debugShowOffer(cmd.Context(), p.RPCURL, p.CIDObjectID, p.Index); err != nil {
					fmt.Printf("after(offer[%d]): %v\n", p.Index, err)
				}
			}
			return nil
		},
	}

	cmd.Flags().String("cid", "", "CID (object id 0x... or CID string)")
	cmd.Flags().String("cid-type", "id", "interpret --cid as 'id' or 'cid'")
	cmd.Flags().Uint64Var(&indexFlag, "idx", 0, "Offer index in next_epoch_offers to approve (0-based)")
	cmd.Flags().BoolVar(&debugFlag, "debug", false, "Verbose debug")

	// optional overrides
	cmd.Flags().StringVar(&signerFlag, "signer-address", "", "Signer address (0x...) overrides env")
	cmd.Flags().StringVar(&privKeyFlag, "signer-private-key", "", "Signer private key (iotaprivkey1...); if omitted you will be promped to insert it")
	cmd.Flags().StringVar(&gasIDFlag, "gas-id", "", "Gas coin object id (0x...) overrides env")

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
