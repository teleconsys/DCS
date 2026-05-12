package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/state"
)

// buildDemo assembles the "Demo: User ⟷ Provider" side-by-side view.
// Each side is a complete workspace bound to its own actor profile.
// Actions are independent: every service call snapshots its profile at
// invocation time, so a Provider submit_offer and a User approve_offer
// can run in parallel without cross-talk.
func buildDemo(win fyne.Window, appState *state.AppState) fyne.CanvasObject {
	userWS := buildWorkspace(win, appState, state.ActorUser)
	providerWS := buildWorkspace(win, appState, state.ActorProvider)

	wrap := func(title string, ws *workspace) fyne.CanvasObject {
		header := widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		return container.NewBorder(header, nil, nil, nil, ws.canvas)
	}

	split := container.NewHSplit(
		wrap(fmt.Sprintf("◀ %s", state.ActorUser), userWS),
		wrap(fmt.Sprintf("%s ▶", state.ActorProvider), providerWS),
	)
	split.SetOffset(0.5)
	return split
}
