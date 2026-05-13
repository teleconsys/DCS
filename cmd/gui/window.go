package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	ftheme "fyne.io/fyne/v2/theme"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/theme"
	"github.com/teleconsys/DCS/cmd/gui/ui"
	"github.com/teleconsys/DCS/cmd/gui/ui/shell"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/gc"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/provider"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/user"
)

// buildMainWindow assembles the shared shell (top app bar, output
// drawer) plus an AppTabs container that hosts one body per actor +
// Demo. Each tab body is constructed by buildActorBody / buildDemo.
// Tab selection rebrands the theme and refreshes the appbar.
func buildMainWindow(win fyne.Window, app *state.AppState, t *theme.Theme) {
	// Per-actor shells (each owns its OutputView and Runner).
	shells := map[state.Actor]*shell.ActorShell{
		state.ActorGC:       shell.NewActorShell(app, state.ActorGC),
		state.ActorUser:     shell.NewActorShell(app, state.ActorUser),
		state.ActorProvider: shell.NewActorShell(app, state.ActorProvider),
	}
	outputs := map[state.Actor]*ui.OutputView{}
	for a, s := range shells {
		outputs[a] = s.Output
	}

	// Output drawer (sits at the bottom).
	outDrawer := shell.NewOutputDrawer(app, outputs)

	// App bar (top chrome).
	var bar *shell.AppBar
	bar = shell.NewAppBar(win, app, t,
		func() {
			shell.OpenIdentity(win, app, app.Registry.Current(), func() {
				bar.Refresh(app.Registry.Current())
			})
		},
		outDrawer.Toggle,
	)

	// Build per-actor content bodies — every actor now uses its
	// dashboard.
	digestCB := func(string) { outDrawer.RefreshDigest() }
	bodies := make(map[state.Actor]fyne.CanvasObject, 3)
	bodies[state.ActorGC] = container.NewPadded(
		gc.Dashboard(shells[state.ActorGC].NewViewContext(win, app, digestCB)),
	)
	bodies[state.ActorProvider] = container.NewPadded(
		provider.Dashboard(shells[state.ActorProvider].NewViewContext(win, app, digestCB)),
	)
	bodies[state.ActorUser] = container.NewPadded(
		user.Dashboard(shells[state.ActorUser].NewViewContext(win, app, digestCB)),
	)
	demoBody := buildDemo(win, app, shells, outDrawer)

	gcTab := container.NewTabItemWithIcon("Ground Control", ftheme.SettingsIcon(), bodies[state.ActorGC])
	userTab := container.NewTabItemWithIcon("User", ftheme.AccountIcon(), bodies[state.ActorUser])
	provTab := container.NewTabItemWithIcon("Provider", ftheme.StorageIcon(), bodies[state.ActorProvider])
	demoTab := container.NewTabItemWithIcon("Demo", ftheme.ComputerIcon(), demoBody)

	tabs := container.NewAppTabs(gcTab, userTab, provTab, demoTab)
	tabs.SetTabLocation(container.TabLocationTop)

	tabs.OnSelected = func(ti *container.TabItem) {
		var a state.Actor
		switch ti {
		case gcTab:
			a = state.ActorGC
			t.SetAccent(theme.AccentGC)
		case userTab:
			a = state.ActorUser
			t.SetAccent(theme.AccentUser)
		case provTab:
			a = state.ActorProvider
			t.SetAccent(theme.AccentProvider)
		case demoTab:
			a = state.ActorUser
			t.SetAccent(theme.AccentDefault)
		default:
			return
		}
		app.Registry.SetCurrent(a)
		bar.Refresh(a)
		outDrawer.SetActor(a)
		fyne.CurrentApp().Settings().SetTheme(t)
	}

	// Initial state: open on User.
	tabs.Select(userTab)
	app.Registry.SetCurrent(state.ActorUser)
	t.SetAccent(theme.AccentUser)
	bar.Refresh(state.ActorUser)
	outDrawer.SetActor(state.ActorUser)

	root := container.NewBorder(
		bar.CanvasObject(),       // top
		outDrawer.CanvasObject(), // bottom
		nil, nil,
		tabs,
	)
	win.SetContent(root)
}
