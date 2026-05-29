// Package dcserrors maps DCS Move module abort codes to human-readable messages.
package dcserrors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	suitypes "github.com/coming-chat/go-sui/v2/types"
)

var (
	reMoveAbortCode = regexp.MustCompile(`MoveAbort\([^,]+,\s*(\d+)\)`)
	reAbortCode     = regexp.MustCompile(`(?i)abort\s+code[:\s]+(\d+)`)
	reFunctionName  = regexp.MustCompile(`function_name:\s*Some\("([^"]+)"\)`)
	reFunctionJSON  = regexp.MustCompile(`"function_name"\s*:\s*"([^"]+)"`)
	reModuleCall    = regexp.MustCompile(`dcs::([a-z_]+)`)
)

// messages maps "function:code" to a user-facing explanation.
var messages = map[string]string{
	"add_id_to_whitelist:0":      "Caller must be @DCSGroundControl",
	"remove_id_from_whitelist:0": "Caller must be @DCSGroundControl",
	"add_to_cidlist:0":           "Caller must be the CID owner",
	"remove_from_cidlist:0":      "Caller must be the CID owner",

	"create_offer:1": "Provider's address must be on the whitelist",
	"create_offer:2": "Current epoch has ended; offers are only accepted for the next epoch",
	"create_offer:3": "Still within 10 minutes after next epoch start — wait before submitting offers",
	"create_offer:4": "Offer amount must be greater than zero",

	"deposit_funds:1": "Caller must be the CID owner",

	"approve_offer:5": "Caller must be the CID owner",
	"approve_offer:6": "At least 10 minutes must have passed after next epoch start before approving offers",
	"approve_offer:7": "Invalid offer index (out of range for next epoch offers)",
	"approve_offer:8": "Offer is already approved",

	"transition_epoch:9":  "Caller must be the CID owner",
	"transition_epoch:10": "Transition is only allowed after next epoch has started",

	"honor_offer:11": "Caller must be the CID owner",
	"honor_offer:12": "Must honor offers before the current epoch ends",
	"honor_offer:13": "Invalid offer index (out of range for current epoch offers)",
	"honor_offer:14": "Offer must be approved before it can be honored",
	"honor_offer:15": "Offer has already been honored",

	"withdraw_payment:16": "Withdraw is only allowed after the previous epoch ended",
	"withdraw_payment:17": "Invalid offer index (out of range for previous epoch offers)",
	"withdraw_payment:18": "Caller must be the offer's provider",
	"withdraw_payment:19": "Offer must be honored before withdrawing payment",
	"withdraw_payment:20": "Offer payment has already been withdrawn",
	"withdraw_payment:21": "CID funds balance is insufficient for this offer amount",
}

var displayNames = map[string]string{
	"add_id_to_whitelist":      "Add to whitelist",
	"remove_id_from_whitelist": "Remove from whitelist",
	"add_to_cidlist":           "Add to CID list",
	"remove_from_cidlist":      "Remove from CID list",
	"create_cid":               "Create CID",
	"create_offer":             "Create offer",
	"deposit_funds":            "Deposit funds",
	"approve_offer":            "Approve offer",
	"transition_epoch":         "Transition epoch",
	"honor_offer":              "Honor offer",
	"withdraw_payment":         "Withdraw payment",
}

// FormatFailure turns a raw on-chain failure reason into a readable message.
// function is the DCS entry point that was called (e.g. "create_offer").
func FormatFailure(function, reason string) string {
	function = strings.TrimSpace(function)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fmt.Sprintf("%s failed: transaction execution returned failure status", DisplayName(function))
	}

	fn, code, ok := parseAbort(reason, function)
	if !ok {
		return fmt.Sprintf("%s failed: %s", DisplayName(function), reason)
	}

	key := fmt.Sprintf("%s:%d", fn, code)
	if msg, found := messages[key]; found {
		return fmt.Sprintf("%s failed: %s (abort code %d)", DisplayName(fn), msg, code)
	}
	return fmt.Sprintf("%s failed: abort code %d (%s)", DisplayName(fn), code, reason)
}

// TxError builds an error for a failed transaction response.
func TxError(function string, resp *suitypes.SuiTransactionBlockResponse) error {
	ok, reason := txStatusOK(resp)
	if ok {
		return nil
	}
	return fmt.Errorf("%s", FormatFailure(function, reason))
}

// Wrap decorates a build/execute/sign error when the message contains an abort code.
func Wrap(function string, err error) error {
	if err == nil {
		return nil
	}
	raw := err.Error()
	if _, _, ok := parseAbort(raw, function); ok {
		return fmt.Errorf("%s", FormatFailure(function, raw))
	}
	return err
}

// DisplayName returns a user-facing label for a Move function.
func DisplayName(function string) string {
	if name, ok := displayNames[function]; ok {
		return name
	}
	if function == "" {
		return "Transaction"
	}
	return function
}

func parseAbort(reason, defaultFunction string) (function string, code int, ok bool) {
	function = strings.TrimSpace(defaultFunction)
	if m := reFunctionName.FindStringSubmatch(reason); len(m) == 2 {
		function = m[1]
	} else if m := reFunctionJSON.FindStringSubmatch(reason); len(m) == 2 {
		function = m[1]
	} else if m := reModuleCall.FindStringSubmatch(reason); len(m) == 2 {
		function = m[1]
	}

	var codeStr string
	switch {
	case reMoveAbortCode.MatchString(reason):
		codeStr = reMoveAbortCode.FindStringSubmatch(reason)[1]
	case reAbortCode.MatchString(reason):
		codeStr = reAbortCode.FindStringSubmatch(reason)[1]
	default:
		return function, 0, false
	}

	code, err := strconv.Atoi(codeStr)
	if err != nil {
		return function, 0, false
	}
	return function, code, true
}

// txStatusOK inspects effects status across SDK JSON shapes.
func txStatusOK(resp *suitypes.SuiTransactionBlockResponse) (bool, string) {
	if resp == nil || resp.Effects == nil {
		return false, "no effects in response"
	}
	b, _ := json.Marshal(resp.Effects)

	var a struct {
		Data struct {
			V1 struct {
				Status struct {
					Status string `json:"status"`
					Error  string `json:"error"`
				} `json:"status"`
			} `json:"v1"`
		} `json:"Data"`
	}
	if json.Unmarshal(b, &a) == nil && a.Data.V1.Status.Status != "" {
		return a.Data.V1.Status.Status == "success", a.Data.V1.Status.Error
	}

	var bshape struct {
		Data struct {
			Status struct {
				Status string `json:"status"`
				Error  string `json:"error"`
			} `json:"status"`
		} `json:"data"`
	}
	if json.Unmarshal(b, &bshape) == nil && bshape.Data.Status.Status != "" {
		return bshape.Data.Status.Status == "success", bshape.Data.Status.Error
	}

	var c struct {
		Status struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"status"`
	}
	if json.Unmarshal(b, &c) == nil && c.Status.Status != "" {
		return c.Status.Status == "success", c.Status.Error
	}

	return true, ""
}
