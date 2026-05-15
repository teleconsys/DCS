package service

import "strconv"

// OfferWindowGraceMs is the post–next-epoch-start window for submitting offers
// (matches internal/offers.FindOpenOfferCIDs).
const OfferWindowGraceMs = int64(600_000)

// IsOfferWindowOpen reports whether providers may still submit offers for the next epoch.
func IsOfferWindowOpen(s CIDSummary, nowMs int64) bool {
	return s.NextEpochStart > 0 &&
		nowMs >= s.CurrentEpochEnd &&
		nowMs < s.NextEpochStart+OfferWindowGraceMs
}

// CanApproveOffers is true once the offer submission window has closed.
func CanApproveOffers(s CIDSummary, nowMs int64) bool {
	if s.NextEpochStart == 0 {
		return false
	}
	return !IsOfferWindowOpen(s, nowMs) && nowMs < s.NextEpochEnd
}

// OfferWindowEndsAtMs is when the offer window closes (unix ms).
func OfferWindowEndsAtMs(s CIDSummary) int64 {
	if s.NextEpochStart == 0 {
		return 0
	}
	return s.NextEpochStart + OfferWindowGraceMs
}

// HonorDeadlineMs is the latest time to honor current-epoch offers (unix ms).
func HonorDeadlineMs(s CIDSummary, nowMs int64) int64 {
	if nowMs < s.CurrentEpochEnd && s.CurrentEpochEnd > 0 {
		return s.CurrentEpochEnd
	}
	if s.NextEpochStart > 0 {
		return s.NextEpochStart
	}
	return s.CurrentEpochEnd
}

// RemainingMS returns milliseconds until deadline, or 0 if past.
func RemainingMS(deadlineMs, nowMs int64) int64 {
	if deadlineMs <= nowMs {
		return 0
	}
	return deadlineMs - nowMs
}

// FormatCountdown formats remaining milliseconds as H:MM:SS or M:SS.
func FormatCountdown(remainingMs int64) string {
	if remainingMs <= 0 {
		return "0:00"
	}
	sec := remainingMs / 1000
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return strconv.FormatInt(h, 10) + ":" + two(m) + ":" + two(s)
	}
	return strconv.FormatInt(m, 10) + ":" + two(s)
}

func two(n int64) string {
	return strconv.FormatInt(n/10, 10) + strconv.FormatInt(n%10, 10)
}

// PhaseTimerLine returns a human-readable phase name and live countdown for a CID tile.
func PhaseTimerLine(s CIDSummary, nowMs int64) (phase string, countdown string) {
	status := deriveStatus(s, nowMs)
	switch status {
	case CIDStatusPending:
		left := RemainingMS(s.CurrentEpochEnd, nowMs)
		return "Active epoch", FormatCountdown(left)
	case CIDStatusOfferWindow:
		left := RemainingMS(OfferWindowEndsAtMs(s), nowMs)
		return "Offers open", FormatCountdown(left)
	case CIDStatusNextScheduled:
		left := RemainingMS(s.NextEpochEnd, nowMs)
		return "Next epoch scheduled", FormatCountdown(left)
	case CIDStatusExpired:
		return "Expired", ""
	default:
		return status.String(), ""
	}
}
