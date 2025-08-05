package iota_sc

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
)

const (
	defaultFaucet = "https://faucet.testnet.iota.cafe/gas"
	keyStoreDir   = ".dcs"        // under user home
	keyFileName   = "keypair.hex" // privkey||pubkey hex, 64+64 bytes
)

// newAccountCmd wires `iota-sc account`
func newAccountCmd() *cobra.Command {
	var (
		privHex   string
		noFaucet  bool
		faucetURL string
	)

	cmd := &cobra.Command{
		Use:   "account",
		Short: "Create or import an IOTA Rebased Ed25519 account and optionally fund it",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var priv ed25519.PrivateKey
			// import or generate address
			if privHex != "" {
				pb, err := hex.DecodeString(privHex)
				if err != nil || len(pb) != ed25519.PrivateKeySize {
					return fmt.Errorf("invalid --private-key hex")
				}
				priv = ed25519.PrivateKey(pb)
			} else {
				_, p, err := ed25519.GenerateKey(rand.Reader)
				if err != nil {
					return err
				}
				priv = p
			}

			pub := priv.Public().(ed25519.PublicKey)
			addr := moveAddress(pub)

			// persist for next runs
			if err := saveKeypair(priv, pub); err != nil {
				cmd.Printf("⚠  could not save keypair locally: %v\n", err)
			}

			cmd.Printf("✅ address: %s\n", addr)

			// faucet request
			if noFaucet {
				return nil
			}
			if faucetURL == "" {
				faucetURL = defaultFaucet
			}
			if err := hitFaucet(faucetURL, addr); err != nil {
				return fmt.Errorf("faucet: %w", err)
			}
			cmd.Println("🚰 faucet request accepted")
			return nil
		},
	}

	cmd.Flags().StringVar(&privHex, "private-key", "", "hex-encoded 64-byte Ed25519 private key to import")
	cmd.Flags().BoolVar(&noFaucet, "no-faucet", false, "skip asking the faucet for funds")
	cmd.Flags().StringVar(&faucetURL, "faucet-url", "", "override faucet endpoint")
	return cmd
}

// moveAddress = 0x + sha3-256(pubkey)
func moveAddress(pub ed25519.PublicKey) string {
	h := sha3.Sum256(pub)
	return "0x" + hex.EncodeToString(h[:])
}

// saveKeypair writes keypair to $HOME/.dcs/keypair.hex (idempotent)
func saveKeypair(priv ed25519.PrivateKey, pub ed25519.PublicKey) error {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, keyStoreDir, keyFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	buf := make([]byte, 0, 2*ed25519.PrivateKeySize)
	buf = append(buf, hex.EncodeToString(priv)...)
	buf = append(buf, '\n')
	buf = append(buf, hex.EncodeToString(pub)...)
	return os.WriteFile(path, buf, 0o600)
}

// hitFaucet POSTs { "FixedAmountRequest": { "recipient": "<addr>" } }.
func hitFaucet(url, addr string) error {
	payload := struct {
		FixedAmountRequest struct {
			Recipient string `json:"recipient"`
		} `json:"FixedAmountRequest"`
	}{}
	payload.FixedAmountRequest.Recipient = addr

	b, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
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
