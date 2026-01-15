package iota_sc

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/teleconsys/DCS/internal/rebased"
)

func newAccountCoinsCmd() *cobra.Command {
	var alias string
	var address string
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "coins",
		Short: "List coin object IDs owned by an address",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := resolveCoinsAddress(alias, address)
			if err != nil {
				return err
			}

			rpcURL := resolveRPCFromCmd(cmd)
			if rpcURL == "" {
				rpcURL = os.Getenv("REBASE_RPC")
			}
			if rpcURL == "" {
				rpcURL = "https://api.testnet.iota.cafe:443"
			}

			ctx := cmd.Context()
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			w, err := rebased.Dial(rpcURL)
			if err != nil {
				return fmt.Errorf("rpc dial failed: %w", err)
			}

			coins, err := fetchAllCoins(ctx, w, addr)
			if err != nil {
				return fmt.Errorf("get all coins failed: %w", err)
			}

			if len(coins) == 0 {
				fmt.Printf("address: %s\n", addr)
				fmt.Println("no coins found")
				return nil
			}

			sort.Slice(coins, func(i, j int) bool {
				return balanceAsBig(coins[i].Balance).Cmp(balanceAsBig(coins[j].Balance)) > 0
			})

			fmt.Printf("address: %s\n", addr)
			fmt.Printf("rpc:     %s\n", rpcURL)
			fmt.Printf("coins:   %d\n\n", len(coins))

			for _, c := range coins {
				fmt.Printf("%s  balance=%s  type=%s\n", c.CoinObjectID, c.Balance, c.CoinType)
			}

			best := pickBestIotaCoin(coins)
			if best != "" {
				fmt.Printf("\nrecommended_gas_coin_id: %s\n", best)
				fmt.Printf("export USER_GAS_COIN_ID=%s\n", best)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&alias, "alias", "", "Account alias (reads ./accounts/<alias>.json)")
	cmd.Flags().StringVar(&address, "address", "", "Owner address (0x...)")
	cmd.Flags().DurationVar(&timeout, "timeout", 25*time.Second, "RPC timeout")

	return cmd
}

func resolveRPCFromCmd(cmd *cobra.Command) string {
	if f := cmd.Flags().Lookup("rpc"); f != nil {
		return strings.TrimSpace(f.Value.String())
	}
	if f := cmd.InheritedFlags().Lookup("rpc"); f != nil {
		return strings.TrimSpace(f.Value.String())
	}
	return ""
}

func resolveCoinsAddress(alias, address string) (string, error) {
	if a := strings.TrimSpace(address); a != "" {
		return a, nil
	}

	if al := strings.TrimSpace(alias); al != "" {
		p := filepath.Join(".", "accounts", al+".json")
		raw, err := os.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("read account file %s: %w", p, err)
		}

		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return "", fmt.Errorf("parse account file %s: %w", p, err)
		}

		if v, ok := m["address"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v), nil
		}
		return "", fmt.Errorf("account file %s has no address field", p)
	}

	for _, k := range []string{"ACTIVE_ADDRESS", "USER_ADDRESS", "PROVIDER_ADDRESS"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v, nil
		}
	}

	return "", fmt.Errorf("missing address: use --address, or --alias, or set USER_ADDRESS/ACTIVE_ADDRESS")
}

func fetchAllCoins(ctx context.Context, w *rebased.Wrapper, owner string) ([]rebased.Coin, error) {
	var out []rebased.Coin
	var cursor *string

	for {
		page, err := w.GetAllCoins(ctx, owner, cursor, 50)
		if err != nil {
			return nil, err
		}
		if page == nil || len(page.Data) == 0 {
			break
		}

		out = append(out, page.Data...)

		if !page.HasNextPage || page.NextCursor == nil || *page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}

	return out, nil
}

func balanceAsBig(s string) *big.Int {
	x := new(big.Int)
	if _, ok := x.SetString(strings.TrimSpace(s), 10); !ok {
		return big.NewInt(0)
	}
	return x
}

func pickBestIotaCoin(coins []rebased.Coin) string {
	bestID := ""
	bestBal := big.NewInt(-1)

	for _, c := range coins {
		if c.CoinObjectID == "" {
			continue
		}
		if strings.TrimSpace(c.CoinType) != "0x2::iota::IOTA" {
			continue
		}
		b := balanceAsBig(c.Balance)
		if b.Cmp(bestBal) > 0 {
			bestBal = b
			bestID = c.CoinObjectID
		}
	}

	return bestID
}
