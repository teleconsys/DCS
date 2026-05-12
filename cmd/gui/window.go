package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/state"
)

const (
	modeGC       = "Ground Control"
	modeUser     = "User"
	modeProvider = "Provider"
	modeDemo     = "Demo: User ⟷ Provider"
)

// buildMainWindow assembles the top-level window: actor switcher,
// content pane (single actor or demo), and a status bar.
func buildMainWindow(win fyne.Window, appState *state.AppState) {
	statusBar := widget.NewLabel("")
	updateStatus := func(mode string) {
		switch mode {
		case modeGC:
			p := appState.Registry.Profile(state.ActorGC)
			ep := p.GCEndpoint
			if ep == "" {
				ep = "(no endpoint)"
			}
			statusBar.SetText(fmt.Sprintf("Ground Control · %s · rpc=%s", ep, p.RPCURL))
		case modeDemo:
			u := appState.Registry.Profile(state.ActorUser)
			pv := appState.Registry.Profile(state.ActorProvider)
			statusBar.SetText(fmt.Sprintf(
				"Demo · User=%s · Provider=%s",
				short(u.Address), short(pv.Address),
			))
		default:
			actor := state.ActorUser
			if mode == modeProvider {
				actor = state.ActorProvider
			}
			p := appState.Registry.Profile(actor)
			statusBar.SetText(fmt.Sprintf(
				"%s · %s · gas=%s · rpc=%s",
				p.Actor, short(p.Address), short(p.GasCoinID), p.RPCURL,
			))
		}
	}

	content := container.NewMax()

	swap := func(mode string) {
		var obj fyne.CanvasObject
		switch mode {
		case modeGC:
			ws := buildWorkspace(win, appState, state.ActorGC)
			obj = ws.canvas
			appState.Registry.SetCurrent(state.ActorGC)
		case modeUser:
			ws := buildWorkspace(win, appState, state.ActorUser)
			obj = ws.canvas
			appState.Registry.SetCurrent(state.ActorUser)
		case modeProvider:
			ws := buildWorkspace(win, appState, state.ActorProvider)
			obj = ws.canvas
			appState.Registry.SetCurrent(state.ActorProvider)
		case modeDemo:
			obj = buildDemo(win, appState)
		default:
			obj = widget.NewLabel("(unknown mode)")
		}
		content.Objects = []fyne.CanvasObject{obj}
		content.Refresh()
		updateStatus(mode)
	}

	switcher := widget.NewRadioGroup(
		[]string{modeGC, modeUser, modeProvider, modeDemo},
		swap,
	)
	switcher.Horizontal = true
	switcher.Required = true
	switcher.SetSelected(modeUser)

	topBar := container.NewVBox(
		container.NewHBox(
			widget.NewLabelWithStyle("Actor:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			switcher,
		),
		widget.NewSeparator(),
	)
	statusBox := container.NewBorder(widget.NewSeparator(), nil, nil, nil, statusBar)

	win.SetContent(container.NewBorder(topBar, statusBox, nil, nil, content))
}

func short(s string) string {
	if len(s) <= 14 {
		if s == "" {
			return "(empty)"
		}
		return s
	}
	return s[:8] + "…" + s[len(s)-4:]
}
