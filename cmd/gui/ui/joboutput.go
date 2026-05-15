package ui

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	digestEqRE = regexp.MustCompile(`digest=([A-Za-z0-9]+)`)
	cidLineRE  = regexp.MustCompile(`(?im)^cid:\s*(\S+)\s*$`)
)

// RunSuccessBody turns captured service log text into a short, user-facing
// message (CIDs, digests, counts) instead of echoing the full CLI transcript.
func RunSuccessBody(jobTitle, buttonLabel, captured string) string {
	c := strings.TrimSpace(captured)
	switch jobTitle {
	case "ipfs load-file":
		return summarizeIPFSLoad(c)
	case "ipfs check-cid":
		return summarizeIPFSCheckCID(c)
	case "ipfs check-pins":
		return summarizeIPFSPins(c)
	case "cid create":
		return summarizeCIDCreatePipeline(c)
	case "cid add-funds":
		return summarizeCoinAndDigest(c, "Funds deposited.")
	case "cid remove":
		return summarizeDigestOnly(c, "CID removed from list.")
	case "cid next-epoch":
		return firstLineWithPrefix(c, "✅", "Epoch updated.")
	case "cid is-in-list":
		return summarizeYesNoLine(c, "in the CID list")
	case "whitelist has":
		return summarizeYesNoLine(c, "on the whitelist")
	case "whitelist add", "whitelist remove":
		return firstCheckOrInfoLine(c, "Whitelist updated.")
	case "submit_offer", "withdraw", "approve_offer", "honor_offer":
		return summarizeDigestOnly(c, "Transaction completed.")
	case "list-open-offers":
		return summarizeOpenOffers(c)
	case "account new":
		return summarizeAccountNew(c)
	case "account coins":
		return summarizeAccountCoins(c)
	default:
		return summarizeDefault(c, buttonLabel)
	}
}

func summarizeIPFSLoad(c string) string {
	if cid := extractCIDLine(c); cid != "" {
		return fmt.Sprintf("File pinned on IPFS.\n\nCID:\n%s", cid)
	}
	if cid := extractCIDFromPathLine(c); cid != "" {
		return fmt.Sprintf("File pinned on IPFS.\n\nCID:\n%s", cid)
	}
	return "File uploaded and pinned."
}

func extractCIDLine(c string) string {
	if m := cidLineRE.FindStringSubmatch(c); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func extractCIDFromPathLine(c string) string {
	for _, line := range strings.Split(c, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "/ipfs/") {
			i := strings.Index(line, "/ipfs/")
			rest := strings.TrimSpace(line[i+len("/ipfs/"):])
			rest = strings.TrimSuffix(rest, "/")
			if rest != "" {
				return rest
			}
		}
	}
	return ""
}

func summarizeIPFSCheckCID(c string) string {
	var pin string
	for _, line := range strings.Split(c, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "📌") {
			pin = line
			break
		}
	}
	if pin != "" {
		return pin
	}
	return "Check finished."
}

func summarizeIPFSPins(c string) string {
	lines := strings.Split(c, "\n")
	start := -1
	for i, line := range lines {
		if strings.Contains(line, "— summary —") {
			start = i
			break
		}
	}
	if start < 0 {
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.Contains(lines[i], "total:") {
				return strings.TrimSpace(lines[i])
			}
		}
		return "Pin check finished."
	}
	var b strings.Builder
	for j := start; j < len(lines) && j < start+5; j++ {
		if t := strings.TrimSpace(lines[j]); t != "" {
			b.WriteString(t)
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String())
}

func summarizeCIDCreatePipeline(c string) string {
	return joinFilteredLines(c, func(s string) bool {
		t := strings.TrimSpace(s)
		return strings.HasPrefix(t, "✅") || strings.HasPrefix(t, "auto-epochs:")
	}, 8)
}

func summarizeCoinAndDigest(c, tail string) string {
	var coin, digest string
	for _, line := range strings.Split(c, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "✅ coin id:") {
			coin = strings.TrimSpace(strings.TrimPrefix(line, "✅ coin id:"))
		}
		if strings.Contains(line, "digest=") {
			if d := extractDigestSuffix(line); d != "" {
				digest = d
			}
		}
	}
	var b strings.Builder
	if coin != "" {
		b.WriteString("Gas coin:\n")
		b.WriteString(coin)
	}
	if digest != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("Digest:\n")
		b.WriteString(digest)
	}
	if b.Len() == 0 {
		return tail
	}
	return strings.TrimSpace(b.String()) + "\n\n" + tail
}

func summarizeDigestOnly(c, tail string) string {
	if d := lastDigestIn(c); d != "" {
		return fmt.Sprintf("Digest:\n%s\n\n%s", d, tail)
	}
	return tail
}

func lastDigestIn(c string) string {
	var last string
	for _, m := range digestEqRE.FindAllStringSubmatch(c, -1) {
		if len(m) > 1 {
			last = m[1]
		}
	}
	return last
}

// LastDigestFromLog returns the last digest=value occurrence in captured log text.
func LastDigestFromLog(c string) string {
	return lastDigestIn(c)
}

func extractDigestSuffix(line string) string {
	if m := digestEqRE.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	return ""
}

func firstLineWithPrefix(c, prefix, fallback string) string {
	for _, line := range strings.Split(c, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return fallback
}

func summarizeYesNoLine(c, ctxPhrase string) string {
	for _, line := range strings.Split(c, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "✓") || strings.HasPrefix(line, "✗") {
			return line
		}
	}
	return "Check finished."
}

func firstCheckOrInfoLine(c, fallback string) string {
	for _, line := range strings.Split(c, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "✅") || strings.HasPrefix(line, "ℹ") {
			return line
		}
	}
	return fallback
}

func summarizeOpenOffers(c string) string {
	for _, line := range strings.Split(c, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "found ") && strings.Contains(line, "open offer") {
			return line
		}
	}
	return "Open windows refreshed."
}

func summarizeAccountNew(c string) string {
	return joinFilteredLines(c, func(s string) bool {
		t := strings.TrimSpace(s)
		switch {
		case strings.HasPrefix(t, "alias:"),
			strings.HasPrefix(t, "address:"),
			strings.HasPrefix(t, "saved:"),
			strings.HasPrefix(t, "faucet:"):
			return true
		default:
			return false
		}
	}, 6)
}

func summarizeAccountCoins(c string) string {
	var addr, coins, rec string
	for _, line := range strings.Split(c, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "address:"):
			addr = line
		case strings.HasPrefix(line, "coins:"):
			coins = line
		case strings.Contains(strings.ToLower(line), "recommended gas"):
			rec = line
		}
	}
	var b strings.Builder
	for _, s := range []string{addr, coins, rec} {
		if s != "" {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(s)
		}
	}
	if b.Len() == 0 {
		return "Coins listed."
	}
	return b.String()
}

func summarizeDefault(c, buttonLabel string) string {
	s := joinFilteredLines(c, func(line string) bool {
		t := strings.TrimSpace(line)
		return strings.HasPrefix(t, "✅")
	}, 5)
	if s != "" {
		return s
	}
	return fmt.Sprintf("%s finished.", buttonLabel)
}

func joinFilteredLines(c string, keep func(string) bool, max int) string {
	var out []string
	for _, line := range strings.Split(c, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !keep(line) {
			continue
		}
		out = append(out, line)
		if len(out) >= max {
			break
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n")
}
