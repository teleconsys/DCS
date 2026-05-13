package components

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	ftheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// KeyValueRow renders a "label: value" pair, optionally with a copy
// button. Empty values render as an em-dash so the row stays aligned.
func KeyValueRow(win fyne.Window, label, value string, copyable bool) fyne.CanvasObject {
	keyLbl := widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	display := strings.TrimSpace(value)
	if display == "" {
		display = "—"
	}
	valLbl := widget.NewLabel(display)

	if !copyable || strings.TrimSpace(value) == "" || win == nil {
		return container.NewBorder(nil, nil, keyLbl, nil, valLbl)
	}
	copyBtn := widget.NewButtonWithIcon("", ftheme.ContentCopyIcon(), func() {
		win.Clipboard().SetContent(value)
	})
	copyBtn.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, keyLbl, copyBtn, valLbl)
}

// Short truncates a long string with an ellipsis in the middle. Useful
// for displaying 0x… addresses and digests in compact rows.
func Short(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 14 {
		if s == "" {
			return "(empty)"
		}
		return s
	}
	return s[:8] + "…" + s[len(s)-4:]
}
