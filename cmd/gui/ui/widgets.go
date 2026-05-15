package ui

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
	"github.com/teleconsys/DCS/cmd/gui/ui/nativefile"
)

var hexAddressRe = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)

// validateAddress returns nil for non-empty 0x-prefixed 64-hex strings.
func validateAddress(allowEmpty bool) func(string) error {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			if allowEmpty {
				return nil
			}
			return errors.New("address is required")
		}
		if !hexAddressRe.MatchString(s) {
			return errors.New("expected 0x + 64 hex characters")
		}
		return nil
	}
}

// NewAddressEntry returns an entry that validates Sui/IOTA addresses.
// Pass allowEmpty=true for optional fields.
func NewAddressEntry(allowEmpty bool, placeholder string) *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder(placeholder)
	e.Validator = validateAddress(allowEmpty)
	return e
}

// NewAmountEntry returns an entry constrained to non-negative integers,
// suitable for IOTA-nanos amounts.
func NewAmountEntry(placeholder string, allowZero bool) *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder(placeholder)
	e.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			if allowZero {
				return nil
			}
			return errors.New("amount is required")
		}
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return errors.New("expected an integer")
		}
		if !allowZero && n == 0 {
			return errors.New("amount must be > 0")
		}
		return nil
	}
	return e
}

// ParseUint64 extracts a uint64 from an entry value, returning 0 when
// blank.
func ParseUint64(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	return strconv.ParseUint(s, 10, 64)
}

// NewPrivateKeyEntry returns a password-style entry (built-in reveal control).
func NewPrivateKeyEntry(initial string) (entry *widget.Entry, container fyne.CanvasObject) {
	e := widget.NewPasswordEntry()
	e.SetPlaceHolder("iotaprivkey1… (or hex / base64 keystore)")
	e.SetText(initial)
	return e, e
}

// NewAliasSelect builds a dropdown from ./accounts/<alias>.json filtered
// by the given actor. Selecting an entry calls onChosen with the loaded
// AccountFile.
func NewAliasSelect(actor state.Actor, onChosen func(state.AccountFile)) *widget.Select {
	accounts := state.AccountsFor(actor)
	labels := []string{""}
	byLabel := map[string]state.AccountFile{}
	for _, a := range accounts {
		label := a.Alias
		if strings.TrimSpace(a.Role) != "" {
			label = fmt.Sprintf("%s (%s)", a.Alias, a.Role)
		}
		labels = append(labels, label)
		byLabel[label] = a
	}
	sel := widget.NewSelect(labels, func(s string) {
		if a, ok := byLabel[s]; ok && onChosen != nil {
			onChosen(a)
		}
	})
	sel.PlaceHolder = "(no alias)"
	return sel
}

// FilePickerRow returns an entry that holds a file path and a "Browse…"
// button that opens the OS file picker (Fyne's dialog only as fallback).
func FilePickerRow(win fyne.Window, placeholder string) (entry *widget.Entry, container fyne.CanvasObject) {
	e := widget.NewEntry()
	e.SetPlaceHolder(placeholder)
	btn := widget.NewButton("Browse…", func() {
		nativefile.PickOpenFile(win, e, "Select file")
	})
	return e, container2(e, btn)
}

func container2(left fyne.CanvasObject, right fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil, nil, right, left)
}

// NewCopyButton returns a small button that copies `provider()`'s value
// onto the clipboard.
func NewCopyButton(win fyne.Window, label string, provider func() string) *widget.Button {
	var btn *widget.Button
	btn = widget.NewButton(label, func() {
		v := strings.TrimSpace(provider())
		if v == "" {
			return
		}
		win.Clipboard().SetContent(v)
		feedback.CopyFlash(btn)
	})
	return btn
}
