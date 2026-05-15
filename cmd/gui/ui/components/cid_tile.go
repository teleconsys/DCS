package components

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	ftheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	apptheme "github.com/teleconsys/DCS/cmd/gui/theme"
	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
)

// LabeledField is a compact two-column label / value row for card interiors.
func LabeledField(label, value string) fyne.CanvasObject {
	key := widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	key.Importance = widget.LowImportance
	val := widget.NewLabel(value)
	val.Wrapping = fyne.TextWrapWord
	return container.NewGridWithColumns(2, key, val)
}

// LabeledWidget is LabeledField with a custom value widget (e.g. live timer).
func LabeledWidget(label string, value fyne.CanvasObject) fyne.CanvasObject {
	key := widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	key.Importance = widget.LowImportance
	return container.NewGridWithColumns(2, key, value)
}

// OfferCountStrip shows next / current / prev offer counts in three columns.
func OfferCountStrip(next, current, prev int) fyne.CanvasObject {
	mk := func(title string, n int) fyne.CanvasObject {
		t := widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		t.Importance = widget.LowImportance
		v := widget.NewLabelWithStyle(strconv.Itoa(n), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		return container.NewVBox(t, v)
	}
	return container.NewGridWithColumns(3, mk("Prev", prev), mk("Current", current), mk("Next", next))
}

// CIDHeaderRow is badge + monospace CID with optional copy.
func CIDHeaderRow(win fyne.Window, cid string, state BadgeState, status string) fyne.CanvasObject {
	cidLbl := widget.NewLabel(cid)
	cidLbl.Wrapping = fyne.TextWrapWord
	cidLbl.TextStyle = fyne.TextStyle{Monospace: true}

	var cidCell fyne.CanvasObject = cidLbl
	if win != nil && cid != "" {
		var copyBtn *widget.Button
		copyBtn = widget.NewButtonWithIcon("", ftheme.ContentCopyIcon(), func() {
			win.Clipboard().SetContent(cid)
			feedback.CopyFlash(copyBtn)
		})
		copyBtn.Importance = widget.LowImportance
		cidCell = container.NewBorder(nil, nil, nil, copyBtn, cidLbl)
	}

	return container.NewBorder(nil, nil, Badge(state, status), nil, cidCell)
}

// TileCard is a compact bordered card: body content, separator, then footer actions.
// No empty title row (unlike CardSeparatorStroke with blank title).
func TileCard(body, footer fyne.CanvasObject) fyne.CanvasObject {
	parts := []fyne.CanvasObject{body}
	if footer != nil {
		parts = append(parts, widget.NewSeparator(), footer)
	}
	inner := container.NewPadded(container.NewVBox(parts...))
	return cardBackdrop(apptheme.SeparatorStroke(), inner)
}

// FormatOfferCountsLabel builds a short summary for tooltips or logs.
func FormatOfferCountsLabel(next, current, prev int) string {
	return fmt.Sprintf("%d / %d / %d", next, current, prev)
}
