package iota_sc

import (
	"github.com/spf13/cobra"
	"github.com/teleconsys/DCS/internal/cid"
)

func NewWithdrawCmd() *cobra.Command {
	var (
		signerFlag  string
		privKeyFlag string
		gasIDFlag   string
		indexFlag   uint64
		debugFlag   bool
	)

	cmd := &cobra.Command{
		Use:   "withdraw",
		Short: "Withdraw IOTA from a fulfilled offer",
		RunE: func(cmd *cobra.Command, args []string) error {
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
	cmd.Flags().Uint64Var(&indexFlag, "idx", 0, "Payment index to withdraw (0-based)")
	cmd.Flags().BoolVar(&debugFlag, "debug", false, "Verbose debug")

	// Optional overrides
	cmd.Flags().StringVar(&signerFlag, "signer-address", "", "Signer address (0x...) overrides env")
	cmd.Flags().StringVar(&privKeyFlag, "signer-private-key", "",
		"Signer private key (iotaprivkey1... / suiprivkey1... or base64 keystore); if omitted, you will be prompted")
	cmd.Flags().StringVar(&gasIDFlag, "signer-gas-id", "", "Gas coin object id (0x...) overrides env")

	_ = cmd.MarkFlagRequired("cid")
	_ = cmd.MarkFlagRequired("idx")

	return cmd
}
