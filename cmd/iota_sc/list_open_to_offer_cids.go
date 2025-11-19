package iota_sc

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cidlist "github.com/teleconsys/DCS/internal/cid"
	"github.com/teleconsys/DCS/internal/offers"
)

func NewListOpenOfferingWindowsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list-open-offers",
		Short:        "Print CIDs whose offer window is open now",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			endpoint := viper.GetString("graphql.endpoint")
			if endpoint == "" {
				if s := os.Getenv("IOTA_GRAPHQL_ENDPOINT"); s != "" {
					endpoint = s
				} else {
					endpoint = "https://graphql.testnet.iota.cafe"
				}
			}

			cidListID, err := cidlist.GetCIDListID()
			if err != nil || cidListID == "" {
				// allow explicit override if user passed a flag
				cidListID = viper.GetString("dcs.cidlist_id")
				if cidListID == "" {
					cidListID = os.Getenv("DCS_CIDLIST_ID")
				}
			}
			if cidListID == "" {
				return fmt.Errorf("missing CID list id (set DCS_CIDLIST_ID or --cidlist-id)")
			}

			items, err := offers.FindOpenOfferCIDs(
				cmd.Context(),
				endpoint,
				cidListID,
				time.Now().UnixMilli(),
			)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(items)
		},
	}

	cmd.Flags().String("graphql-endpoint", "", "GraphQL endpoint (overrides .env)")
	_ = viper.BindPFlag("graphql.endpoint", cmd.Flags().Lookup("graphql-endpoint"))
	cmd.Flags().String("cidlist-id", "", "CID list object ID (overrides .env)")
	_ = viper.BindPFlag("dcs.cidlist_id", cmd.Flags().Lookup("cidlist-id"))

	return cmd
}
