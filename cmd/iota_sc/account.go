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
	"golang.org/x/crypto/blake2b"
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
			if strings.TrimSpace(alias) == "" {
				return errors.New("missing --alias")
			}

			// 1) Generate ed25519 keypair.
			pub, priv, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				return fmt.Errorf("generate key: %w", err)
			}

			// Convert private key to bech32 iotaprivkey1... format
			iotapriv, err := ed25519PrivToIotaPrivKey(priv)
			if err != nil {
				return fmt.Errorf("encode private key: %w", err)
			}
			// 2) Derive address: 0x + sha3-256(pub).
			addr := deriveAddress(pub)

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
			}

			payload := struct {
				Alias      string `json:"alias"`
				Address    string `json:"address"`
				PublicKey  string `json:"public_key"`  // hex
				PrivateKey string `json:"private_key"` // iotaprivkey1...
			}{
				Alias:      alias,
				Address:    addr,
				PublicKey:  hex.EncodeToString(pub),
				PrivateKey: iotapriv,
			}

			b, _ := json.MarshalIndent(payload, "", "  ")
			if err := os.WriteFile(outPath, b, 0o600); err != nil {
				return fmt.Errorf("write %s: %w", outPath, err)
			}

			// 4) Print essentials.
			cmd.Println("account created")
			cmd.Printf("alias:       %s\n", payload.Alias)
			cmd.Printf("address:     %s\n", payload.Address)
			cmd.Printf("public key:  %s\n", payload.PublicKey)
			cmd.Printf("private key: %s\n", payload.PrivateKey)
			cmd.Println("NOTE: keep the private key secret; anyone with it can control this account.")
			cmd.Printf("saved file:  %s\n", outPath)

			// 5) Optional faucet funding via .env variable FAUCET_URL.
			if noFaucet {
				return nil
			}
			faucetURL := os.Getenv("FAUCET_URL") // loaded by internal/config/dotenv.go
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

// Ed25519 address = 0x + blake2b-256(pubkey)
func deriveAddress(pub ed25519.PublicKey) string {
	sum := blake2b.Sum256(pub)
	return "0x" + hex.EncodeToString(sum[:])
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
