package service

import (
	"bytes"
	"context"
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
	"time"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/internal/wallet"
)

// NewAccountForm collects the inputs of the "account new" view.
type NewAccountForm struct {
	Alias        string
	NoFaucet     bool
	FaucetAmount uint64
	Role         state.Actor // tag written into accounts/<alias>.json
	FaucetURL    string      // optional override; falls back to env
}

// NewAccount generates an ed25519 keypair, writes ./accounts/<alias>.json
// and optionally requests funding from the faucet. It mirrors the CLI's
// `iota_sc account new` but takes a NewAccountForm instead of cobra
// flags, and accepts an io.Writer for progress output.
func NewAccount(ctx context.Context, form NewAccountForm, out io.Writer) (state.AccountFile, error) {
	alias := strings.TrimSpace(form.Alias)
	if alias == "" {
		return state.AccountFile{}, errors.New("missing alias")
	}
	for _, r := range alias {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_') {
			return state.AccountFile{}, fmt.Errorf(
				"invalid alias %q: only letters, digits, '-' and '_' are allowed", alias)
		}
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return state.AccountFile{}, fmt.Errorf("generate key: %w", err)
	}
	addr := wallet.DeriveAddress(pub)

	wd, err := os.Getwd()
	if err != nil {
		return state.AccountFile{}, fmt.Errorf("resolve working dir: %w", err)
	}
	dir := filepath.Join(wd, "accounts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return state.AccountFile{}, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	outPath := filepath.Join(dir, alias+".json")
	if _, err := os.Stat(outPath); err == nil {
		return state.AccountFile{}, fmt.Errorf("account file already exists: %s", outPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return state.AccountFile{}, fmt.Errorf("stat %s: %w", outPath, err)
	}

	acc := state.AccountFile{
		Alias:      alias,
		Address:    addr,
		PublicKey:  hex.EncodeToString(pub),
		PrivateKey: hex.EncodeToString(priv),
		Role:       form.Role.RoleTag(),
	}
	b, err := json.MarshalIndent(acc, "", "  ")
	if err != nil {
		return state.AccountFile{}, fmt.Errorf("marshal account: %w", err)
	}
	if err := os.WriteFile(outPath, b, 0o600); err != nil {
		return state.AccountFile{}, fmt.Errorf("write %s: %w", outPath, err)
	}

	fmt.Fprintf(out, "account created\n")
	fmt.Fprintf(out, "alias:   %s\n", alias)
	fmt.Fprintf(out, "role:    %s\n", acc.Role)
	fmt.Fprintf(out, "address: %s\n", addr)
	fmt.Fprintf(out, "saved:   %s\n", outPath)

	if form.NoFaucet {
		return acc, nil
	}
	faucet := strings.TrimSpace(form.FaucetURL)
	if faucet == "" {
		faucet = os.Getenv("FAUCET_URL")
	}
	if faucet == "" {
		fmt.Fprintf(out, "faucet: skipped (FAUCET_URL not set)\n")
		return acc, nil
	}
	if err := faucetRequest(ctx, faucet, addr, form.FaucetAmount); err != nil {
		return acc, fmt.Errorf("faucet request failed: %w", err)
	}
	fmt.Fprintf(out, "faucet: requested\n")
	return acc, nil
}

func faucetRequest(ctx context.Context, base, addr string, amt uint64) error {
	u := strings.TrimRight(base, "/") + "/gas"
	type fixed struct {
		Recipient string `json:"recipient"`
		Amount    string `json:"amount,omitempty"`
	}
	body := map[string]any{"FixedAmountRequest": fixed{Recipient: addr}}
	if amt > 0 {
		body["FixedAmountRequest"] = fixed{Recipient: addr, Amount: fmt.Sprintf("%d", amt)}
	}
	b, _ := json.Marshal(body)

	hctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(hctx, http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
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

// LoadAccountIntoProfile fills the signing fields of `prof` from the
// account file matching `alias`. Network/contract fields are not touched.
func LoadAccountIntoProfile(prof *state.ActorProfile, alias string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(wd, "accounts", alias+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var a state.AccountFile
	if err := json.Unmarshal(raw, &a); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	prof.Alias = a.Alias
	if a.Address != "" {
		prof.Address = a.Address
	}
	if a.PrivateKey != "" {
		// Account files store hex; the GUI accepts any format that
		// rebased.KeyStringToKeystoreB64 understands, so just pass it
		// through.
		prof.PrivateKey = a.PrivateKey
	}
	return nil
}
