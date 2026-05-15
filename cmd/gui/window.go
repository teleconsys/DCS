package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	ftheme "fyne.io/fyne/v2/theme"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/theme"
	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
	"github.com/teleconsys/DCS/cmd/gui/ui/shell"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/gc"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/provider"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/user"
)

// buildMainWindow assembles the shared shell (top app bar) plus an
// AppTabs container that hosts one body per actor + Demo. Tab selection
// rebrands the theme and refreshes the app bar.
func buildMainWindow(win fyne.Window, app *state.AppState, t *theme.Theme) {
	shells := map[state.Actor]*shell.ActorShell{
		state.ActorGC:       shell.NewActorShell(app, state.ActorGC),
		state.ActorUser:     shell.NewActorShell(app, state.ActorUser),
		state.ActorProvider: shell.NewActorShell(app, state.ActorProvider),
	}

	fb := feedback.NewHost()

	var bar *shell.AppBar
	bar = shell.NewAppBar(win, app, t, fb,
		func() {
			act := app.Registry.Current()
			vc := shells[act].NewViewContext(win, app, fb)
			shell.OpenIdentity(win, app, act, vc, func() {
				bar.Refresh(act)
			})
		},
	)

	bodies := make(map[state.Actor]fyne.CanvasObject, 3)
	bodies[state.ActorGC] = container.NewPadded(
		gc.Dashboard(shells[state.ActorGC].NewViewContext(win, app, fb)),
	)
	bodies[state.ActorProvider] = container.NewPadded(
		provider.Dashboard(shells[state.ActorProvider].NewViewContext(win, app, fb)),
	)
	bodies[state.ActorUser] = container.NewPadded(
		user.Dashboard(shells[state.ActorUser].NewViewContext(win, app, fb)),
	)
	demoBody := buildDemo(win, app, shells, fb)

	gcTab := container.NewTabItemWithIcon("Admin", ftheme.SettingsIcon(), bodies[state.ActorGC])
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
		fyne.CurrentApp().Settings().SetTheme(t)
	}

	tabs.Select(userTab)
	app.Registry.SetCurrent(state.ActorUser)
	t.SetAccent(theme.AccentUser)
	bar.Refresh(state.ActorUser)

	root := container.NewBorder(
		bar.CanvasObject(),
		fb.CanvasObject(),
		nil, nil,
		tabs,
	)
	win.SetContent(container.NewStack(root))
}
