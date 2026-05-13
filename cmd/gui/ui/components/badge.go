package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// BadgeState enumerates the supported badge styles. Colors mirror the
// semantic palette used by the theme.
type BadgeState int

const (
	BadgeNeutral BadgeState = iota
	BadgeOK
	BadgeWarn
	BadgeErr
	BadgeInfo
)

// Color returns the dot color for the badge state.
func (s BadgeState) Color() color.Color {
	switch s {
	case BadgeOK:
		return color.NRGBA{R: 0x22, G: 0xC5, B: 0x5E, A: 0xFF}
	case BadgeWarn:
		return color.NRGBA{R: 0xF5, G: 0x9E, B: 0x0B, A: 0xFF}
	case BadgeErr:
		return color.NRGBA{R: 0xEF, G: 0x44, B: 0x44, A: 0xFF}
	case BadgeInfo:
		return color.NRGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF}
	}
	return color.NRGBA{R: 0x64, G: 0x74, B: 0x8B, A: 0xFF}
}

// Badge renders a small colored dot followed by a label. Designed for
// inline use in cards and table rows.
func Badge(state BadgeState, text string) fyne.CanvasObject {
	dot := canvas.NewCircle(state.Color())
	sized := container.NewGridWrap(fyne.NewSize(10, 10), dot)
	return container.NewHBox(container.NewCenter(sized), widget.NewLabel(text))
}
