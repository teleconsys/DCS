package iota_sc

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/teleconsys/DCS/internal/offers"
)

func NewSubmitOfferCmd() *cobra.Command {
	var (
		signerFlag  string
		privKeyFlag string
		gasIDFlag   string
		debugFlag   bool
	)

	cmd := &cobra.Command{
		Use:   "submit_offer",
		Short: "Submit an offer for the next epoch (provider wallet)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := offers.LoadSubmitOfferParams(cmd.Context(), cmd, args)
			if err != nil {
				return err
			}

			if p.Debug {
				fmt.Println("== preflight ==")
				fmt.Printf("signer=%s  gas=%s  amount(nanos)=%d\n", p.Signer, p.GasID, p.Amount)
				if err := debugCIDWindow(cmd.Context(), p.RPCURL, p.CIDObjectID); err != nil {
					fmt.Printf("preflight(cid): %v\n", err)
				}
				if err := debugCIDOfferLens(cmd.Context(), p.RPCURL, p.CIDObjectID, "before"); err != nil {
					fmt.Printf("preflight(offers): %v\n", err)
				}
			}

			resp, err := offers.SubmitReplicaOffer(context.Background(), p)
			if err != nil {
				return err
			}

			cmd.Println("✅ offer submitted")
			cmd.Printf("digest: %s\n", resp.Digest)

			if p.Debug {
				if err := debugTxStatus(resp); err != nil {
					fmt.Printf("after(tx): %v\n", err)
				}
				if err := debugCIDOfferLens(cmd.Context(), p.RPCURL, p.CIDObjectID, "after"); err != nil {
					fmt.Printf("after(offers): %v\n", err)
				}
			}
			return nil
		},
	}

	cmd.Flags().String("cid", "", "CID object id (0x...)")
	cmd.Flags().Uint64("amount", 0, "Offer amount (IOTA nanos)")
	cmd.Flags().BoolVar(&debugFlag, "debug", false, "Verbose debug (preflight + postflight)")

	// optional overrides
	cmd.Flags().StringVar(&signerFlag, "signer-address", "", "Signer address (0x...) overrides env")
	cmd.Flags().StringVar(&privKeyFlag, "signer-private-key", "", "Signer private key (iotaprivkey1...); if omitted you will be promped to insert it")
	cmd.Flags().StringVar(&gasIDFlag, "signer-gas-id", "", "Gas coin object id (0x...) overrides env")

	_ = cmd.MarkFlagRequired("cid")
	_ = cmd.MarkFlagRequired("amount")
	return cmd
}
