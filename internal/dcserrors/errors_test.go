package dcserrors

import (
	"strings"
	"testing"
)

func TestFormatFailure_knownCode(t *testing.T) {
	raw := `MoveAbort(MoveLocation { module: ModuleId { address: 0x1, name: Identifier("dcs") }, function: 3, instruction: 10, function_name: Some("create_offer") }, 1)`
	got := FormatFailure("create_offer", raw)
	if !strings.Contains(got, "Provider's address must be on the whitelist") {
		t.Fatalf("unexpected message: %q", got)
	}
	if !strings.Contains(got, "abort code 1") {
		t.Fatalf("expected code in message: %q", got)
	}
}

func TestFormatFailure_collisionCodeUsesFunction(t *testing.T) {
	raw := `MoveAbort(..., 0)`
	got := FormatFailure("add_to_cidlist", raw)
	if !strings.Contains(got, "CID owner") {
		t.Fatalf("unexpected message: %q", got)
	}
	got = FormatFailure("add_id_to_whitelist", raw)
	if !strings.Contains(got, "DCSGroundControl") {
		t.Fatalf("unexpected message: %q", got)
	}
}

func TestFormatFailure_unknownReason(t *testing.T) {
	got := FormatFailure("approve_offer", "network timeout")
	if !strings.Contains(got, "network timeout") {
		t.Fatalf("unexpected message: %q", got)
	}
}

func TestFormatFailure_emptyReason(t *testing.T) {
	got := FormatFailure("transition_epoch", "")
	if !strings.Contains(got, "failure status") {
		t.Fatalf("unexpected message: %q", got)
	}
}

func TestWrap_executeError(t *testing.T) {
	raw := `MoveAbort(MoveLocation { function_name: Some("deposit_funds") }, 1)`
	err := Wrap("deposit_funds", fmtError(raw))
	if err == nil || !strings.Contains(err.Error(), "CID owner") {
		t.Fatalf("unexpected wrap: %v", err)
	}
}

type fmtError string

func (e fmtError) Error() string { return string(e) }
