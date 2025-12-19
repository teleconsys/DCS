package iota_sc

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/teleconsys/DCS/internal/offers"
)

func NewHonorOfferCmd() *cobra.Command {
	var (
		signerFlag  string
		privKeyFlag string
		gasIDFlag   string
		indexFlag   uint64
		debugFlag   bool
	)

	cmd := &cobra.Command{
		Use:   "honor_offer",
		Short: "Honor a confirmed offer in the current epoch (CID owner)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := offers.LoadHonorParams(cmd.Context(), cmd, args)
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

			resp, err := offers.HonorOffer(context.Background(), p)
			if err != nil {
				return err
			}

			cmd.Println("✅ offer honored")
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

	cmd.Flags().String("cid", "", "CID object id (0x...)")
	cmd.Flags().Uint64Var(&indexFlag, "idx", 0, "Offer index in next_epoch_offers to approve (0-based)")
	cmd.Flags().BoolVar(&debugFlag, "debug", false, "Verbose debug")

	// optional overrides
	cmd.Flags().StringVar(&signerFlag, "signer-address", "", "Signer address (0x...) overrides env")
	cmd.Flags().StringVar(&privKeyFlag, "signer-private-key", "", "Signer private key (iotaprivkey1...) if omitted you will be promped to insert it")
	cmd.Flags().StringVar(&gasIDFlag, "signer-gas-id", "", "Gas coin object id (0x...) overrides env")

	_ = cmd.MarkFlagRequired("cid")
	return cmd
}
