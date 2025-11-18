package iota_sc

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	cidlib "github.com/teleconsys/DCS/internal/cid"
	"github.com/teleconsys/DCS/internal/offers"
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
				return fmt.Errorf("--cid is required (object id 0x... or CID string)")
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
			)
			if signer == "" {
				return fmt.Errorf("missing signer address (set --signer-address or PROVIDER_ADDRESS / GC_ADDRESS / USER_ADDRESS)")
			}
			privKey, err := ResolvePrivateKey(flagPrivKey)
			if err != nil {
				return err
			}
			gasID := firstNonEmpty(
				flagGasID,
				os.Getenv("PROVIDER_GAS_COIN_ID"),
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

			if flagDebug {
				if err := debugTxStatus(resp); err != nil {
					fmt.Printf("after(tx): %v\n", err)
				}
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
	cmd.Flags().StringVar(&flagPrivKey, "signer-private-key", "", "Signer private key (iotaprivkey1...); if omitted you will be promped to insert it")
	cmd.Flags().StringVar(&flagGasID, "gas-id", "", "Gas coin object id (0x...) overrides env")

	_ = cmd.MarkFlagRequired("cid")
	_ = cmd.MarkFlagRequired("amount")
	return cmd
}
