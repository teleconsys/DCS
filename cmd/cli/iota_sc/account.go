package iota_sc

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"

	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/spf13/cobra"

	"github.com/teleconsys/DCS/internal/wallet"
)

// Command: dcs iota_sc account new --alias <name> [--no_faucet] [--faucet-amount <n>]
func newAccountCmd() *cobra.Command {
	var alias string
	var noFaucet bool
	var faucetAmount uint64

	cmd := &cobra.Command{
		Use:   "account",
		Short: "Account utilities (local key generation + optional faucet)",
	}

	newCmd := &cobra.Command{
		Use:   "new",
		Short: "Generate an ed25519 keypair and address; optionally fund via faucet from .env",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 0) Validate alias.
			alias = strings.TrimSpace(alias)
			if alias == "" {
				return errors.New("missing --alias")
			}

			for _, r := range alias {
				if !((r >= 'a' && r <= 'z') ||
					(r >= 'A' && r <= 'Z') ||
					(r >= '0' && r <= '9') ||
					r == '-' || r == '_') {
					return fmt.Errorf(
						"invalid alias %q: only letters, digits, '-' and '_' are allowed",
						alias,
					)
				}
			}

			// 1) Generate ed25519 keypair.
			pub, priv, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				return fmt.Errorf("generate key: %w", err)
			}

			// 2) Derive address: 0x + blake2b-256(pub).
			addr := wallet.DeriveAddress(pub)

			// 3) Save to ./accounts/<alias>.json (relative to execution folder).
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("resolve working dir: %w", err)
			}
			dir := filepath.Join(wd, "accounts")
			// Safe even if it already exists.
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", dir, err)
			}

			outPath := filepath.Join(dir, alias+".json")

			if _, err := os.Stat(outPath); err == nil {
				return fmt.Errorf("account file already exists: %s", outPath)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("stat %s: %w", outPath, err)
			}

			// 4) Serialize.
			payload := struct {
				Alias      string `json:"alias"`
				Address    string `json:"address"`
				PublicKey  string `json:"public_key"`  // hex
				PrivateKey string `json:"private_key"` // hex (keep secret)
			}{
				Alias:      alias,
				Address:    addr,
				PublicKey:  hex.EncodeToString(pub),
				PrivateKey: hex.EncodeToString(priv),
			}

			b, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal account payload: %w", err)
			}

			if err := os.WriteFile(outPath, b, 0o600); err != nil {
				return fmt.Errorf("write %s: %w", outPath, err)
			}

			// 5) Print account summary.
			cmd.Println("account created")
			cmd.Printf("alias:   %s\n", alias)
			cmd.Printf("address: %s\n", addr)
			cmd.Printf("privkey: %s\n", hex.EncodeToString(priv))
			cmd.Printf("saved:   %s\n", outPath)

			// 6) Optional faucet request.
			if noFaucet {
				return nil
			}

			faucetURL := os.Getenv("FAUCET_URL")
			if faucetURL == "" {
				return errors.New("FAUCET_URL not set in environment/.env; use --no_faucet to skip")
			}

			if err := faucetRequest(faucetURL, addr, faucetAmount); err != nil {
				return fmt.Errorf("faucet request failed: %w", err)
			}

			cmd.Println("faucet: requested")
			return nil
		},
	}

	newCmd.Flags().StringVar(&alias, "alias", "", "Account alias; saved as $HOME/DCS/accounts/<alias>.json")
	newCmd.Flags().BoolVar(&noFaucet, "no_faucet", false, "Do not call the faucet even if FAUCET_URL is set")
	newCmd.Flags().Uint64Var(&faucetAmount, "faucet-amount", 0, "Optional amount to request from faucet")

	cmd.AddCommand(newCmd)
	return cmd
}

func faucetRequest(base, addr string, amt uint64) error {
	// Ensure /gas endpoint
	u := strings.TrimRight(base, "/") + "/gas"

	type fixed struct {
		Recipient string `json:"recipient"`
		Amount    string `json:"amount,omitempty"`
	}
	body := map[string]any{
		"FixedAmountRequest": fixed{Recipient: addr},
	}
	if amt > 0 {
		body["FixedAmountRequest"] = fixed{
			Recipient: addr,
			Amount:    fmt.Sprintf("%d", amt),
		}
	}

	b, _ := json.Marshal(body)
	resp, err := http.Post(u, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}
	return nil
}

// ed25519PrivToIotaPrivKey encodes an ed25519 private key into the
// iotaprivkey1... bech32 format used by IOTA Rebased / Sui keystores.
//
// Layout: bech32(hrp="iotaprivkey", convertBits( [0x00 | seed32], 8->5, pad=true ))
func ed25519PrivToIotaPrivKey(priv ed25519.PrivateKey) (string, error) {
	// ed25519.PrivateKey.Seed() returns the 32-byte private seed.
	seed := priv.Seed()
	if len(seed) != ed25519.SeedSize {
		return "", fmt.Errorf("unexpected ed25519 seed length: %d", len(seed))
	}

	// 0x00 is the Ed25519 scheme flag (same as used by Sui / IOTA keystore).
	raw := append([]byte{0x00}, seed...)

	// 8-bit -> 5-bit groups, with padding.
	fiveBit, err := bech32.ConvertBits(raw, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("bech32 8->5 convert: %w", err)
	}

	// HRP is iotaprivkey for IOTA Rebased.
	return bech32.Encode("iotaprivkey", fiveBit)
}
