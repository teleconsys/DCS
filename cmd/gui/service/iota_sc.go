package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/teleconsys/DCS/cmd/gui/state"
	cidlib "github.com/teleconsys/DCS/internal/cid"
	"github.com/teleconsys/DCS/internal/offers"
	"github.com/teleconsys/DCS/internal/rebased"
	"github.com/teleconsys/DCS/internal/wallet"
	"github.com/teleconsys/DCS/internal/whitelist"
)

// signer is the verified signing triple produced by prepareSigner.
type signer struct {
	address string
	privKey string
	gasID   string
}

// prepareSigner validates the private key, derives + cross-checks the
// address, and verifies the gas coin is owned by the signer. It mirrors
// the CLI's pre-call resolution but without any stdin prompts.
func prepareSigner(ctx context.Context, p state.ActorProfile) (signer, error) {
	var s signer
	priv, err := wallet.ResolvePrivateKeyStrict(p.PrivateKey)
	if err != nil {
		return s, fmt.Errorf("private key: %w", err)
	}
	addr, err := wallet.ResolveSignerAddressStrict(priv, p.Address)
	if err != nil {
		return s, err
	}
	rpc := p.RPCURL
	if strings.TrimSpace(rpc) == "" {
		return s, fmt.Errorf("RPC URL is empty")
	}
	gasID, err := wallet.ResolveGasCoinId(ctx, p.GasCoinID, addr, rpc)
	if err != nil {
		return s, fmt.Errorf("gas coin: %w", err)
	}
	return signer{address: addr, privKey: priv, gasID: gasID}, nil
}

// ----------------------------------------------------------------------------
// Ping
// ----------------------------------------------------------------------------

// PingResult bundles the values needed for a friendly "node is alive"
// message in the UI.
type PingResult struct {
	Checkpoint uint64
	Timestamp  time.Time
}

// Ping is actor-agnostic: it just needs an RPC URL.
func Ping(ctx context.Context, rpcURL string, timeout time.Duration, out io.Writer) (PingResult, error) {
	var res PingResult
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	w, err := rebased.Dial(rpcURL)
	if err != nil {
		return res, fmt.Errorf("dial: %w", err)
	}
	cp, err := w.Ping(ctx)
	if err != nil {
		return res, err
	}
	checkpoint, err := w.GetCheckpoint(ctx, cp)
	if err != nil {
		return res, err
	}
	var ms int64
	if _, err := fmt.Sscanf(checkpoint.TimestampMs, "%d", &ms); err != nil {
		return res, fmt.Errorf("parse timestamp: %w", err)
	}
	res.Checkpoint = cp
	res.Timestamp = time.UnixMilli(ms)
	fmt.Fprintf(out, "✅ node alive  checkpoint=%d  ts=%s\n",
		cp, res.Timestamp.Format("2006-01-02 15:04:05"))
	return res, nil
}

// ----------------------------------------------------------------------------
// Account coins
// ----------------------------------------------------------------------------

// AccountCoinsResult is the friendlier shape consumed by the UI table.
type AccountCoinsResult struct {
	Address     string
	Coins       []rebased.Coin
	BestIotaGas string
}

// AccountCoins lists coin objects owned by `address`. The "best" coin is
// the largest 0x2::iota::IOTA balance, suitable as the signer's gas coin.
func AccountCoins(ctx context.Context, rpcURL, address string, timeout time.Duration, out io.Writer) (AccountCoinsResult, error) {
	var res AccountCoinsResult
	if strings.TrimSpace(address) == "" {
		return res, fmt.Errorf("address is empty")
	}
	if strings.TrimSpace(rpcURL) == "" {
		return res, fmt.Errorf("RPC URL is empty")
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	w, err := rebased.Dial(rpcURL)
	if err != nil {
		return res, fmt.Errorf("dial: %w", err)
	}
	var all []rebased.Coin
	var cursor *string
	for {
		page, err := w.GetAllCoins(ctx, address, cursor, 50)
		if err != nil {
			return res, fmt.Errorf("get_all_coins: %w", err)
		}
		if page == nil || len(page.Data) == 0 {
			break
		}
		all = append(all, page.Data...)
		if !page.HasNextPage || page.NextCursor == nil || *page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}

	sort.Slice(all, func(i, j int) bool {
		return coinBalance(all[i].Balance).Cmp(coinBalance(all[j].Balance)) > 0
	})
	res.Address = address
	res.Coins = all
	res.BestIotaGas = bestIotaGas(all)

	fmt.Fprintf(out, "address: %s\nrpc:     %s\ncoins:   %d\n", address, rpcURL, len(all))
	for _, c := range all {
		fmt.Fprintf(out, "%s  balance=%s  type=%s\n", c.CoinObjectID, c.Balance, c.CoinType)
	}
	if res.BestIotaGas != "" {
		fmt.Fprintf(out, "\nrecommended gas coin: %s\n", res.BestIotaGas)
	}
	return res, nil
}

func coinBalance(s string) *big.Int {
	x := new(big.Int)
	if _, ok := x.SetString(strings.TrimSpace(s), 10); !ok {
		return big.NewInt(0)
	}
	return x
}

func bestIotaGas(coins []rebased.Coin) string {
	best := ""
	bestBal := big.NewInt(-1)
	for _, c := range coins {
		if c.CoinObjectID == "" {
			continue
		}
		if strings.TrimSpace(c.CoinType) != "0x2::iota::IOTA" {
			continue
		}
		b := coinBalance(c.Balance)
		if b.Cmp(bestBal) > 0 {
			bestBal = b
			best = c.CoinObjectID
		}
	}
	return best
}

// ----------------------------------------------------------------------------
// Whitelist (RPC read + GC API write)
// ----------------------------------------------------------------------------

// WhitelistHas is read-only, so any actor profile with a valid RPC URL
// and whitelist id can call it.
func WhitelistHas(ctx context.Context, p state.ActorProfile, member string, out io.Writer) (bool, error) {
	if strings.TrimSpace(p.WhitelistID) == "" {
		return false, fmt.Errorf("missing whitelist id (DCS_WHITELIST_ID)")
	}
	if strings.TrimSpace(member) == "" {
		return false, fmt.Errorf("member address is empty")
	}
	hp := whitelist.HasParams{
		WhitelistID: p.WhitelistID,
		Member:      strings.ToLower(strings.TrimSpace(member)),
		RPCURL:      p.RPCURL,
	}
	found, err := whitelist.HasAddress(ctx, hp)
	if err != nil {
		return false, err
	}
	if found {
		fmt.Fprintf(out, "✓ %s is whitelisted\n", hp.Member)
	} else {
		fmt.Fprintf(out, "✗ %s is NOT whitelisted\n", hp.Member)
	}
	return found, nil
}

// ----------------------------------------------------------------------------
// CID list reads
// ----------------------------------------------------------------------------

// CIDIsInList returns true when the given CID object id appears in the
// canonical CID list referenced by `p`.
func CIDIsInList(ctx context.Context, p state.ActorProfile, cidID string, out io.Writer) (bool, error) {
	if strings.TrimSpace(cidID) == "" {
		return false, fmt.Errorf("CID object id is empty")
	}
	if strings.TrimSpace(p.CIDListID) == "" {
		return false, fmt.Errorf("missing CID list id (DCS_CIDLIST_ID)")
	}
	params := cidlib.IsInListParams{CIDId: cidID, CIDListID: p.CIDListID, RPCURL: p.RPCURL}
	found, err := cidlib.IsInList(ctx, params)
	if err != nil {
		return false, err
	}
	if found {
		fmt.Fprintf(out, "✓ %s is in the CID list\n", cidID)
	} else {
		fmt.Fprintf(out, "✗ %s is NOT in the CID list\n", cidID)
	}
	return found, nil
}

// ----------------------------------------------------------------------------
// CID lifecycle (signed by User)
// ----------------------------------------------------------------------------

// CIDCreateForm collects the user-facing inputs for `iota_sc cid create`.
// If FilePath is non-empty, the file is uploaded to IPFS first and the
// resulting CID is used as the on-chain identifier; otherwise CID is
// used verbatim.
type CIDCreateForm struct {
	FilePath   string
	CID        string
	EpochStart uint64
	EpochEnd   uint64
	Amount     uint64
}

// CIDCreateResult is the outcome of the multi-step `cid create` pipeline.
type CIDCreateResult struct {
	CIDStr  string
	CIDID   string
	CoinID  string
	EpochMS struct{ Start, End uint64 }
}

// CIDCreate runs the same three-step flow as the CLI:
//
//	(optional) ipfsutil.LoadFileToIPFS
//	cid.CreateGasCoin
//	cid.CreateCID
//	cid.AddToCIDList
func CIDCreate(ctx context.Context, p state.ActorProfile, form CIDCreateForm, out io.Writer) (CIDCreateResult, error) {
	var res CIDCreateResult
	if p.Actor != state.ActorUser {
		return res, fmt.Errorf("cid create requires the User actor")
	}
	if strings.TrimSpace(p.PackageID) == "" {
		return res, fmt.Errorf("missing DCS_PACKAGE_ID")
	}
	if strings.TrimSpace(p.WhitelistID) == "" {
		return res, fmt.Errorf("missing DCS_WHITELIST_ID")
	}
	if strings.TrimSpace(p.CIDListID) == "" {
		return res, fmt.Errorf("missing DCS_CIDLIST_ID")
	}

	cidStr := strings.TrimSpace(form.CID)
	if path := strings.TrimSpace(form.FilePath); path != "" {
		s, err := LoadFile(ctx, path, out)
		if err != nil {
			return res, fmt.Errorf("upload: %w", err)
		}
		cidStr = s
	}
	if cidStr == "" {
		return res, fmt.Errorf("provide either a file path or a CID")
	}
	res.CIDStr = cidStr

	sg, err := prepareSigner(ctx, p)
	if err != nil {
		return res, err
	}

	epochStart, epochEnd := form.EpochStart, form.EpochEnd
	if epochStart == 0 || epochEnd == 0 {
		now := uint64(time.Now().UnixMilli())
		epochStart = now + 2*60*1000
		epochEnd = epochStart + 20*60*1000
		fmt.Fprintf(out, "auto-epochs: start=%d end=%d\n", epochStart, epochEnd)
	}
	res.EpochMS.Start = epochStart
	res.EpochMS.End = epochEnd

	amount := form.Amount
	if amount == 0 {
		amount = 100_000
	}

	createParams := cidlib.CreateParams{
		CID:               cidStr,
		EpochStart:        epochStart,
		EpochEnd:          epochEnd,
		Amount:            amount,
		WhitelistID:       p.WhitelistID,
		CIDListID:         p.CIDListID,
		PackageID:         p.PackageID,
		GasID:             sg.gasID,
		GasBudget:         nonZero(p.GasBudget, 10_000_000),
		RPCURL:            p.RPCURL,
		UserSignerAddress: sg.address,
		UserPrivateKey:    sg.privKey,
	}

	fmt.Fprintf(out, "Creating gas coin (amount=%d)…\n", amount)
	coinID, err := cidlib.CreateGasCoin(ctx, createParams.GasCoinConfig(), amount)
	if err != nil {
		return res, fmt.Errorf("create gas coin: %w", err)
	}
	res.CoinID = coinID
	fmt.Fprintf(out, "✅ coin id: %s\n", coinID)

	fmt.Fprintf(out, "Creating CID object…\n")
	_, cidID, err := cidlib.CreateCID(ctx, createParams, coinID)
	if err != nil {
		return res, fmt.Errorf("create CID: %w", err)
	}
	res.CIDID = cidID
	fmt.Fprintf(out, "✅ CID object: %s\n", cidID)

	fmt.Fprintf(out, "Registering in CID list…\n")
	if _, err := cidlib.AddToCIDList(ctx, createParams, cidID); err != nil {
		return res, fmt.Errorf("add to cid list: %w", err)
	}
	fmt.Fprintf(out, "✅ CID %s added to list\n", cidID)
	return res, nil
}

// CIDRemove deletes a CID object from the CID list (User action).
func CIDRemove(ctx context.Context, p state.ActorProfile, cidID string, out io.Writer) (string, error) {
	if p.Actor != state.ActorUser {
		return "", fmt.Errorf("cid remove requires the User actor")
	}
	if strings.TrimSpace(cidID) == "" {
		return "", fmt.Errorf("CID object id is empty")
	}
	sg, err := prepareSigner(ctx, p)
	if err != nil {
		return "", err
	}
	params := cidlib.RemoveParams{
		CIDId:             cidID,
		CIDListID:         p.CIDListID,
		PackageID:         p.PackageID,
		GasID:             sg.gasID,
		GasBudget:         nonZero(p.GasBudget, 10_000_000),
		RPCURL:            p.RPCURL,
		UserSignerAddress: sg.address,
		UserPrivateKey:    sg.privKey,
	}
	respBytes, err := cidlib.RemoveCID(ctx, params)
	if err != nil {
		return "", err
	}
	digest := extractDigest(respBytes)
	fmt.Fprintf(out, "✅ removed %s  digest=%s\n", cidID, digest)
	return digest, nil
}

// CIDAddFunds tops up a CID's storage budget. The CLI creates a fresh
// gas coin with the requested amount and then calls deposit_funds; the
// GUI mirrors that.
func CIDAddFunds(ctx context.Context, p state.ActorProfile, cidID string, amount uint64, out io.Writer) (string, error) {
	if p.Actor != state.ActorUser {
		return "", fmt.Errorf("cid add-funds requires the User actor")
	}
	if amount == 0 {
		return "", fmt.Errorf("amount must be > 0")
	}
	sg, err := prepareSigner(ctx, p)
	if err != nil {
		return "", err
	}
	params := cidlib.AddFundsParams{
		CIDId:             cidID,
		Amount:            amount,
		PackageID:         p.PackageID,
		GasID:             sg.gasID,
		GasBudget:         nonZero(p.GasBudget, 10_000_000),
		RPCURL:            p.RPCURL,
		UserSignerAddress: sg.address,
		UserPrivateKey:    sg.privKey,
	}
	fmt.Fprintf(out, "Creating gas coin (amount=%d)…\n", amount)
	coinID, err := cidlib.CreateGasCoin(ctx, params.GasCoinConfig(), amount)
	if err != nil {
		return "", fmt.Errorf("create gas coin: %w", err)
	}
	params.CoinID = coinID
	fmt.Fprintf(out, "✅ coin id: %s\n", coinID)

	respBytes, err := cidlib.AddFunds(ctx, params)
	if err != nil {
		return "", err
	}
	digest := extractDigest(respBytes)
	fmt.Fprintf(out, "✅ funds deposited  digest=%s\n", digest)
	return digest, nil
}

// CIDTransitionEpoch advances a CID to the next epoch (User action).
func CIDTransitionEpoch(ctx context.Context, p state.ActorProfile, cidID string, out io.Writer) error {
	if p.Actor != state.ActorUser {
		return fmt.Errorf("cid next-epoch requires the User actor")
	}
	sg, err := prepareSigner(ctx, p)
	if err != nil {
		return err
	}
	if err := cidlib.CheckEpochTransitionAllowed(ctx, cidID, p.RPCURL); err != nil {
		fmt.Fprintf(out, "⚠ %v\n", err)
		return err
	}
	params := cidlib.TransitionParams{
		RPCURL:            p.RPCURL,
		GasID:             sg.gasID,
		GasBudget:         nonZero(p.GasBudget, 10_000_000),
		UserSignerAddress: sg.address,
		UserPrivateKey:    sg.privKey,
		PackageID:         p.PackageID,
	}
	if _, err := cidlib.TransitionEpoch(ctx, params, cidID); err != nil {
		return err
	}
	fmt.Fprintf(out, "✅ epoch transition successful\n")
	return nil
}

// ----------------------------------------------------------------------------
// Offers (Provider submit/withdraw, User approve/honor)
// ----------------------------------------------------------------------------

// SubmitOfferForm collects the inputs for `submit_offer` (Provider).
type SubmitOfferForm struct {
	CIDObjectID string
	Amount      uint64
	Debug       bool
}

// SubmitOffer issues a provider offer for the next epoch.
func SubmitOffer(ctx context.Context, p state.ActorProfile, form SubmitOfferForm, out io.Writer) (string, error) {
	if p.Actor != state.ActorProvider {
		return "", fmt.Errorf("submit_offer requires the Provider actor")
	}
	if strings.TrimSpace(form.CIDObjectID) == "" {
		return "", fmt.Errorf("CID is required")
	}
	if form.Amount == 0 {
		return "", fmt.Errorf("amount must be > 0")
	}
	sg, err := prepareSigner(ctx, p)
	if err != nil {
		return "", err
	}
	params := offers.SubmitOfferParams{
		CIDObjectID: form.CIDObjectID,
		Amount:      form.Amount,
		WhitelistID: p.WhitelistID,
		ClockID:     defaultStr(p.ClockID, "0x6"),
		PackageID:   p.PackageID,
		GasID:       sg.gasID,
		GasBudget:   nonZero(p.GasBudget, 10_000_000),
		RPCURL:      p.RPCURL,
		Signer:      sg.address,
		PrivKey:     sg.privKey,
		Debug:       form.Debug,
	}
	resp, err := offers.SubmitReplicaOffer(ctx, params)
	if err != nil {
		return "", err
	}
	digest := fmt.Sprintf("%s", resp.Digest)
	fmt.Fprintf(out, "✅ offer submitted  digest=%s\n", digest)
	return digest, nil
}

// OfferIndexForm collects the shared inputs for approve/honor/withdraw.
type OfferIndexForm struct {
	CIDObjectID string
	Index       uint64
	Debug       bool
}

// ApproveOffer (User action) approves a provider offer.
func ApproveOffer(ctx context.Context, p state.ActorProfile, form OfferIndexForm, out io.Writer) (string, error) {
	if p.Actor != state.ActorUser {
		return "", fmt.Errorf("approve_offer requires the User actor")
	}
	if strings.TrimSpace(form.CIDObjectID) == "" {
		return "", fmt.Errorf("CID is required")
	}
	sg, err := prepareSigner(ctx, p)
	if err != nil {
		return "", err
	}
	params := offers.ApproveOfferParams{
		CIDObjectID: form.CIDObjectID,
		Index:       form.Index,
		ClockID:     defaultStr(p.ClockID, "0x6"),
		PackageID:   p.PackageID,
		GasID:       sg.gasID,
		GasBudget:   nonZero(p.GasBudget, 10_000_000),
		RPCURL:      p.RPCURL,
		Signer:      sg.address,
		PrivKey:     sg.privKey,
		Debug:       form.Debug,
	}
	resp, err := offers.ApproveOffer(ctx, params)
	if err != nil {
		return "", err
	}
	digest := fmt.Sprintf("%s", resp.Digest)
	fmt.Fprintf(out, "✅ offer approved  digest=%s\n", digest)
	return digest, nil
}

// HonorOffer (User action) honors a confirmed offer in the current epoch.
func HonorOffer(ctx context.Context, p state.ActorProfile, form OfferIndexForm, out io.Writer) (string, error) {
	if p.Actor != state.ActorUser {
		return "", fmt.Errorf("honor_offer requires the User actor")
	}
	if strings.TrimSpace(form.CIDObjectID) == "" {
		return "", fmt.Errorf("CID is required")
	}
	sg, err := prepareSigner(ctx, p)
	if err != nil {
		return "", err
	}
	params := offers.HonorOfferParams{
		CIDObjectID: form.CIDObjectID,
		Index:       form.Index,
		ClockID:     defaultStr(p.ClockID, "0x6"),
		PackageID:   p.PackageID,
		GasID:       sg.gasID,
		GasBudget:   nonZero(p.GasBudget, 10_000_000),
		RPCURL:      p.RPCURL,
		Signer:      sg.address,
		PrivKey:     sg.privKey,
		Debug:       form.Debug,
	}
	resp, err := offers.HonorOffer(ctx, params)
	if err != nil {
		return "", err
	}
	digest := fmt.Sprintf("%s", resp.Digest)
	fmt.Fprintf(out, "✅ offer honored  digest=%s\n", digest)
	return digest, nil
}

// Withdraw (Provider action) collects payment from a fulfilled offer.
func Withdraw(ctx context.Context, p state.ActorProfile, form OfferIndexForm, out io.Writer) (string, error) {
	if p.Actor != state.ActorProvider {
		return "", fmt.Errorf("withdraw requires the Provider actor")
	}
	if strings.TrimSpace(form.CIDObjectID) == "" {
		return "", fmt.Errorf("CID is required")
	}
	sg, err := prepareSigner(ctx, p)
	if err != nil {
		return "", err
	}
	params := offers.WithdrawParams{
		CIDObjectID: form.CIDObjectID,
		Index:       form.Index,
		ClockID:     defaultStr(p.ClockID, "0x6"),
		PackageID:   p.PackageID,
		GasID:       sg.gasID,
		GasBudget:   nonZero(p.GasBudget, 10_000_000),
		RPCURL:      p.RPCURL,
		Signer:      sg.address,
		PrivKey:     sg.privKey,
		Debug:       form.Debug,
	}
	resp, err := offers.Withdraw(ctx, params)
	if err != nil {
		return "", err
	}
	digest := fmt.Sprintf("%s", resp.Digest)
	fmt.Fprintf(out, "✅ payment withdrawn  digest=%s\n", digest)
	return digest, nil
}

// ----------------------------------------------------------------------------
// Open offer windows (Provider read)
// ----------------------------------------------------------------------------

// ListOpenOffers queries the GraphQL endpoint for currently-open offer
// windows. The result is also pretty-printed to `out`.
func ListOpenOffers(ctx context.Context, p state.ActorProfile, out io.Writer) ([]offers.OpenOfferCID, error) {
	endpoint := p.GraphQLEndpoint
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "https://graphql.testnet.iota.cafe"
	}
	if strings.TrimSpace(p.CIDListID) == "" {
		return nil, fmt.Errorf("missing CID list id (DCS_CIDLIST_ID)")
	}
	items, err := offers.FindOpenOfferCIDs(ctx, endpoint, p.CIDListID, time.Now().UnixMilli())
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(out, "found %d open offer windows\n", len(items))
	for _, it := range items {
		fmt.Fprintf(out,
			"  %s  cid=%s  owner=%s  closes_in=%dm  next_offers=%d\n",
			it.ID, it.CID, it.Owner, it.ClosesInMinutes, it.NextOffers)
	}
	return items, nil
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

func defaultStr(s, def string) string {
	if v := strings.TrimSpace(s); v != "" {
		return v
	}
	return def
}

func nonZero(v, def uint64) uint64 {
	if v == 0 {
		return def
	}
	return v
}

// extractDigest is best-effort: it parses the JSON response of a tx
// execution and returns the digest if present.
func extractDigest(b []byte) string {
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return ""
	}
	if d, ok := m["digest"].(string); ok {
		return d
	}
	return ""
}
