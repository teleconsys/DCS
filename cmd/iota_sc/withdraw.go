package iota_sc

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/teleconsys/DCS/internal/cid"
)

func NewWithdrawCmd() *cobra.Command {
	var (
		flagSigner  string
		flagPrivKey string
		flagGasID   string
		flagIdx     uint64
		flagDebug   bool
	)

	cmd := &cobra.Command{
		Use:   "withdraw",
		Short: "Withdraw IOTA from a fulfilled offer",
		RunE: func(cmd *cobra.Command, args []string) error {

			privKey, err := ResolvePrivateKey(flagPrivKey)
			if err != nil {
				return err
			}

			if err := cmd.Flags().Set("signer-private-key", privKey); err != nil {
				return fmt.Errorf("set signer-private-key flag: %w", err)
			}

			p, err := cid.LoadWithdrawParams(cmd.Context(), cmd, args)
			if err != nil {
				return err
			}

			resp, err := cid.Withdraw(cmd.Context(), p)
			if err != nil {
				return err
			}

			cmd.Println("✅ payment withdrawn")
			if resp != nil {
				cmd.Printf("digest: %s\n", resp.Digest)
			}

			return nil
		},
	}

	// Required / core flags
	cmd.Flags().String("cid", "", "CID (object id 0x... or CID string)")
	cmd.Flags().String("cid-type", "id", "interpret --cid as 'id' or 'cid'")
	cmd.Flags().Uint64Var(&flagIdx, "idx", 0, "Payment index to withdraw (0-based)")
	cmd.Flags().BoolVar(&flagDebug, "debug", false, "Verbose debug")

	// Optional overrides
	cmd.Flags().StringVar(&flagSigner, "signer-address", "", "Signer address (0x...) overrides env")
	cmd.Flags().StringVar(&flagPrivKey, "signer-private-key", "",
		"Signer private key (iotaprivkey1... / suiprivkey1... or base64 keystore); if omitted, you will be prompted")
	cmd.Flags().StringVar(&flagGasID, "gas-id", "", "Gas coin object id (0x...) overrides env")

	_ = cmd.MarkFlagRequired("cid")
	_ = cmd.MarkFlagRequired("idx")
	// signer-private-key is optional; ResolvePrivateKey will prompt if empty.

	return cmd
}
