package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui/shell"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/provider"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/user"
)

// buildDemo lays out the User and Provider dashboards side-by-side so
// the User↔Provider story (upload → offer window opens → submit → approve
// → honor → withdraw) can be demoed in a single window. Each side
// drives its own actor shell (independent runner).
func buildDemo(win fyne.Window, app *state.AppState, shells map[state.Actor]*shell.ActorShell) fyne.CanvasObject {
	userVC := shells[state.ActorUser].NewViewContext(win, app)
	provVC := shells[state.ActorProvider].NewViewContext(win, app)

	userBody := user.Dashboard(userVC)
	provBody := provider.Dashboard(provVC)

	leftHdr := widget.NewLabelWithStyle("◀ User", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	rightHdr := widget.NewLabelWithStyle("Provider ▶", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	left := container.NewStack(container.NewBorder(leftHdr, nil, nil, nil, container.NewPadded(userBody)))
	right := container.NewStack(container.NewBorder(rightHdr, nil, nil, nil, container.NewPadded(provBody)))
	split := container.NewHSplit(left, right)
	split.SetOffset(0.5)
	// HSplit MinSize width is leading+trailing (both full dashboards), which forced the
	// whole window to ~2× a normal tab width on startup. Horizontal scroll keeps a sane
	// minimum while still allowing the full side-by-side demo when the pane is wide.
	demo := container.NewHScroll(split)
	demo.SetMinSize(fyne.NewSize(560, 320))
	return container.NewStack(demo)
}
