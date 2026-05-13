package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	suitypes "github.com/coming-chat/go-sui/v2/types"

	"github.com/teleconsys/DCS/cmd/gui/state"
	cidlib "github.com/teleconsys/DCS/internal/cid"
	"github.com/teleconsys/DCS/internal/rebased"
)

// CIDStatus is the dashboard-friendly state of a CID object derived
// from its on-chain epoch fields and the current wall clock.
type CIDStatus int

const (
	CIDStatusUnknown CIDStatus = iota
	// CIDStatusPending: current_epoch_end not reached yet — content is
	// being stored during the current window.
	CIDStatusPending
	// CIDStatusOfferWindow: offer window for the next epoch is open.
	CIDStatusOfferWindow
	// CIDStatusNextScheduled: offers approved/honored, awaiting epoch
	// transition.
	CIDStatusNextScheduled
	// CIDStatusExpired: window closed without scheduled next epoch.
	CIDStatusExpired
)

// String returns a label suitable for status badges.
func (s CIDStatus) String() string {
	switch s {
	case CIDStatusPending:
		return "Active"
	case CIDStatusOfferWindow:
		return "Offers open"
	case CIDStatusNextScheduled:
		return "Next epoch scheduled"
	case CIDStatusExpired:
		return "Expired"
	}
	return "Unknown"
}

// CIDSummary is a flattened view of one CID Move object's fields,
// suitable for rendering in the User dashboard tile grid.
type CIDSummary struct {
	ID                string
	CIDStr            string
	Owner             string
	CurrentEpochStart int64
	CurrentEpochEnd   int64
	NextEpochStart    int64
	NextEpochEnd      int64
	Balance           int64
	NextOffers        int
	CurrentOffers     int
	PrevOffers        int
	Status            CIDStatus
}

// OfferRow represents one entry in an offers array (next/current/prev)
// projected with the most useful fields for UI rendering.
type OfferRow struct {
	Index       int
	CIDObjectID string
	CIDStr      string
	Provider    string
	Amount      int64
	Approved    bool
	Honored     bool
	Withdrawn   bool
}

// WhitelistMembers returns the address list stored in the whitelist
// object's `fields.whitelist` array. Composed exclusively from
// internal/rebased; mirrors the parsing pattern used by
// internal/whitelist.HasAddress.
func WhitelistMembers(ctx context.Context, p state.ActorProfile) ([]string, error) {
	if strings.TrimSpace(p.WhitelistID) == "" {
		return nil, fmt.Errorf("missing whitelist id (DCS_WHITELIST_ID)")
	}
	if strings.TrimSpace(p.RPCURL) == "" {
		return nil, fmt.Errorf("RPC URL is empty")
	}
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial: %w", err)
	}
	obj, err := w.GetObject(ctx, p.WhitelistID, suitypes.SuiObjectDataOptions{ShowContent: true})
	if err != nil {
		return nil, err
	}
	if obj == nil || obj.Data == nil || obj.Data.Content == nil {
		return nil, nil
	}
	b, err := json.Marshal(obj.Data.Content)
	if err != nil {
		return nil, fmt.Errorf("marshal content: %w", err)
	}
	fields := contentFields(b)
	if fields == nil {
		return nil, nil
	}
	raw, ok := fields["whitelist"]
	if !ok {
		return nil, nil
	}
	return collectAddresses(raw), nil
}

// MyCIDs returns CIDSummary entries for every CID in the canonical CID
// list whose `owner` field matches ownerAddr (case-insensitive).
func MyCIDs(ctx context.Context, p state.ActorProfile, ownerAddr string) ([]CIDSummary, error) {
	if strings.TrimSpace(p.CIDListID) == "" {
		return nil, fmt.Errorf("missing CID list id (DCS_CIDLIST_ID)")
	}
	if strings.TrimSpace(ownerAddr) == "" {
		return nil, fmt.Errorf("owner address is empty")
	}
	ids, err := cidlib.GetCIDList(ctx, p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("get cid list: %w", err)
	}
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial: %w", err)
	}
	owner := strings.ToLower(strings.TrimSpace(ownerAddr))
	out := make([]CIDSummary, 0, len(ids))
	for _, id := range ids {
		fields, err := cidlib.GetCIDFields(ctx, w, id)
		if err != nil {
			continue
		}
		if !strings.EqualFold(asString(fields["owner"]), owner) {
			continue
		}
		out = append(out, summarizeCID(id, fields))
	}
	return out, nil
}

// OffersForCID reads all three offer arrays (next/current/prev) on a
// CID object and returns them as projected OfferRows.
func OffersForCID(ctx context.Context, p state.ActorProfile, cidID string) (next, current, prev []OfferRow, err error) {
	if strings.TrimSpace(cidID) == "" {
		return nil, nil, nil, fmt.Errorf("CID id is empty")
	}
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("rpc dial: %w", err)
	}
	fields, err := cidlib.GetCIDFields(ctx, w, cidID)
	if err != nil {
		return nil, nil, nil, err
	}
	cidStr := decodeCidStr(fields["cid_str"])
	next = projectOffers(cidID, cidStr, fields["next_epoch_offers"])
	current = projectOffers(cidID, cidStr, fields["current_epoch_offers"])
	prev = projectOffers(cidID, cidStr, fields["prev_epoch_offers"])
	return next, current, prev, nil
}

// MyOffersAsProvider enumerates the CID list and collects offers whose
// `provider` field matches providerAddr. Each returned OfferRow keeps
// its source CID object id so the UI can issue Withdraw against it.
func MyOffersAsProvider(ctx context.Context, p state.ActorProfile, providerAddr string) ([]OfferRow, error) {
	if strings.TrimSpace(p.CIDListID) == "" {
		return nil, fmt.Errorf("missing CID list id")
	}
	if strings.TrimSpace(providerAddr) == "" {
		return nil, fmt.Errorf("provider address is empty")
	}
	ids, err := cidlib.GetCIDList(ctx, p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("get cid list: %w", err)
	}
	w, err := rebased.Dial(p.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("rpc dial: %w", err)
	}
	target := strings.ToLower(strings.TrimSpace(providerAddr))
	out := make([]OfferRow, 0, len(ids))
	for _, id := range ids {
		fields, err := cidlib.GetCIDFields(ctx, w, id)
		if err != nil {
			continue
		}
		cidStr := decodeCidStr(fields["cid_str"])
		for _, key := range []string{"next_epoch_offers", "current_epoch_offers", "prev_epoch_offers"} {
			for _, row := range projectOffers(id, cidStr, fields[key]) {
				if strings.EqualFold(row.Provider, target) {
					out = append(out, row)
				}
			}
		}
	}
	return out, nil
}

// IOTABalance returns the total balance (in nanos) of native
// 0x2::iota::IOTA coins owned by address. Composed via AccountCoins so
// it benefits from the existing pagination loop.
func IOTABalance(ctx context.Context, rpcURL, address string) (uint64, error) {
	res, err := AccountCoins(ctx, rpcURL, address, 25*time.Second, discardWriter{})
	if err != nil {
		return 0, err
	}
	var total uint64
	for _, c := range res.Coins {
		if strings.TrimSpace(c.CoinType) != "0x2::iota::IOTA" {
			continue
		}
		n, perr := strconv.ParseUint(strings.TrimSpace(c.Balance), 10, 64)
		if perr != nil {
			continue
		}
		total += n
	}
	return total, nil
}

// discardWriter is a tiny no-op io.Writer used to silence services
// that take an io.Writer for logging.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// GCHealth issues a quick GET to the GC API's /healthz endpoint and
// reports availability. It does NOT need a token (the endpoint is
// unauthenticated in cmd/api/main.go).
func GCHealth(ctx context.Context, p state.ActorProfile) error {
	endpoint := strings.TrimSpace(p.GCEndpoint)
	if endpoint == "" {
		return fmt.Errorf("GC endpoint is empty")
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}
	url := strings.TrimRight(endpoint, "/") + "/healthz"
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// helpers (no /internal modifications)
// ---------------------------------------------------------------------------

// contentFields walks the same shape variants handled by
// internal/whitelist.extractFieldsFromContent.
func contentFields(jsonBytes []byte) map[string]any {
	var root map[string]any
	if err := json.Unmarshal(jsonBytes, &root); err != nil {
		return nil
	}
	for _, route := range [][]string{
		{"Data", "moveObject", "fields"},
		{"data", "moveObject", "fields"},
		{"moveObject", "fields"},
		{"fields"},
		{"bcs", "fields"},
	} {
		if f := walk(root, route); f != nil {
			return f
		}
	}
	return nil
}

func walk(m map[string]any, route []string) map[string]any {
	cur := m
	for _, k := range route {
		next, ok := cur[k].(map[string]any)
		if !ok {
			return nil
		}
		cur = next
	}
	return cur
}

// collectAddresses extracts a flat []string of addresses from a Move
// vector<address> field. Handles both plain strings and the wrapped
// shapes `{ "bytes": "0x…" }` and `{ "id": "0x…" }`.
func collectAddresses(raw any) []string {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s := asString(item); s != "" {
			out = append(out, strings.ToLower(s))
		}
	}
	return out
}

// asString flattens the assortment of Sui JSON address encodings into a
// plain string.
func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if s, _ := t["bytes"].(string); s != "" {
			return s
		}
		if s, _ := t["id"].(string); s != "" {
			return s
		}
		if f, ok := t["fields"].(map[string]any); ok {
			if s, _ := f["bytes"].(string); s != "" {
				return s
			}
			if s, _ := f["id"].(string); s != "" {
				return s
			}
		}
	}
	return ""
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	}
	return 0
}

func asBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// decodeCidStr reproduces the byte-array → string decoding used by
// internal/offers/utils.decodeBytesString. The on-chain cid_str field
// is a vector<u8>.
func decodeCidStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	arr, ok := v.([]any)
	if !ok {
		return ""
	}
	buf := make([]byte, 0, len(arr))
	for _, x := range arr {
		switch n := x.(type) {
		case float64:
			buf = append(buf, byte(n))
		case int:
			buf = append(buf, byte(n))
		case string:
			// stringified numbers, "104" etc.
			if i, err := strconv.Atoi(n); err == nil {
				buf = append(buf, byte(i))
			}
		}
	}
	return string(buf)
}

// projectOffers normalizes the Move offer struct into OfferRow.
// Offers are stored as `{ "fields": { provider, amount, approved, … } }`
// in the SDK JSON, but some shapes inline the fields one level higher.
func projectOffers(cidID, cidStr string, raw any) []OfferRow {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]OfferRow, 0, len(arr))
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		f, ok := m["fields"].(map[string]any)
		if !ok {
			f = m
		}
		out = append(out, OfferRow{
			Index:       i,
			CIDObjectID: cidID,
			CIDStr:      cidStr,
			Provider:    strings.ToLower(asString(f["provider"])),
			Amount:      asInt64(f["amount"]),
			Approved:    asBool(f["approved"]),
			Honored:     asBool(f["honored"]),
			Withdrawn:   asBool(f["withdrawn"]),
		})
	}
	return out
}

// summarizeCID derives a CIDSummary (including a status badge) from the
// raw fields map returned by cidlib.GetCIDFields.
func summarizeCID(id string, fields map[string]any) CIDSummary {
	s := CIDSummary{
		ID:                id,
		CIDStr:            decodeCidStr(fields["cid_str"]),
		Owner:             strings.ToLower(asString(fields["owner"])),
		CurrentEpochStart: asInt64(fields["current_epoch_start"]),
		CurrentEpochEnd:   asInt64(fields["current_epoch_end"]),
		NextEpochStart:    asInt64(fields["next_epoch_start"]),
		NextEpochEnd:      asInt64(fields["next_epoch_end"]),
		NextOffers:        lenSlice(fields["next_epoch_offers"]),
		CurrentOffers:     lenSlice(fields["current_epoch_offers"]),
		PrevOffers:        lenSlice(fields["prev_epoch_offers"]),
		Balance:           extractBalance(fields["funds"]),
	}
	s.Status = deriveStatus(s, time.Now().UnixMilli())
	return s
}

// extractBalance walks `funds.fields.balance` (a u64 in Move). The CID
// object's funds Coin has `fields.balance`.
func extractBalance(raw any) int64 {
	m, ok := raw.(map[string]any)
	if !ok {
		return 0
	}
	if f, ok := m["fields"].(map[string]any); ok {
		return asInt64(f["balance"])
	}
	return asInt64(m["balance"])
}

func lenSlice(v any) int {
	if s, ok := v.([]any); ok {
		return len(s)
	}
	return 0
}

// deriveStatus mirrors the heuristic used by
// internal/offers.FindOpenOfferCIDs: the offer window for the next
// epoch is open while `current_epoch_end ≤ now < next_epoch_start + 10m`.
func deriveStatus(s CIDSummary, nowMs int64) CIDStatus {
	const tenMin = int64(600_000)
	if s.NextEpochStart > 0 && nowMs >= s.CurrentEpochEnd && nowMs < s.NextEpochStart+tenMin {
		return CIDStatusOfferWindow
	}
	if nowMs < s.CurrentEpochEnd {
		return CIDStatusPending
	}
	if s.NextEpochStart > 0 && nowMs >= s.NextEpochStart+tenMin && nowMs < s.NextEpochEnd {
		return CIDStatusNextScheduled
	}
	if s.NextEpochEnd > 0 && nowMs >= s.NextEpochEnd {
		return CIDStatusExpired
	}
	return CIDStatusExpired
}
