// Package shell hosts the cross-actor chrome (top app bar, identity
// drawer, bottom output drawer) and the per-actor "shell" struct that
// bundles each actor's live runner + output view together with the
// profile pointer. Per-actor dashboards consume *ActorShell to wire
// themselves into the shared shell.
package shell

import (
	"fyne.io/fyne/v2"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// ActorShell bundles the per-actor live objects used to wire dashboards
// into the shared shell. The Profile pointer is the same instance that
// the registry returns, so identity edits made anywhere in the UI are
// visible everywhere.
type ActorShell struct {
	Actor   state.Actor
	Profile *state.ActorProfile
	Output  *ui.OutputView
	Runner  *service.Runner
}

// NewActorShell prepares a shell for the given actor.
func NewActorShell(app *state.AppState, actor state.Actor) *ActorShell {
	return &ActorShell{
		Actor:   actor,
		Profile: app.Registry.Profile(actor),
		Output:  ui.NewOutputView(actor.String() + " log"),
		Runner:  service.NewRunner(),
	}
}

// NewViewContext returns a ui.ViewContext bound to this shell. Every
// digest is forwarded to app.SetLastDigest plus the optional onDigest
// hook so the output drawer can render a consolidated indicator.
func (s *ActorShell) NewViewContext(win fyne.Window, app *state.AppState, onDigest func(digest string)) *ui.ViewContext {
	return &ui.ViewContext{
		Window:  win,
		Output:  s.Output,
		Runner:  s.Runner,
		Profile: s.Profile,
		App:     app,
		OnDigest: func(digest string) {
			app.SetLastDigest(s.Actor, digest)
			if onDigest != nil {
				onDigest(digest)
			}
		},
	}
}
