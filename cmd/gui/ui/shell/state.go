// Package shell hosts the cross-actor chrome (top app bar, identity
// drawer) and the per-actor "shell" struct that bundles each actor's
// live runner together with the profile pointer. Per-actor dashboards
// consume *ActorShell to wire themselves into the shared shell.
package shell

import (
	"io"

	"fyne.io/fyne/v2"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
)

// ActorShell bundles the per-actor live objects used to wire dashboards
// into the shared shell. The Profile pointer is the same instance that
// the registry returns, so identity edits made anywhere in the UI are
// visible everywhere.
type ActorShell struct {
	Actor   state.Actor
	Profile *state.ActorProfile
	Runner  *service.Runner
}

// NewActorShell prepares a shell for the given actor.
func NewActorShell(app *state.AppState, actor state.Actor) *ActorShell {
	return &ActorShell{
		Actor:   actor,
		Profile: app.Registry.Profile(actor),
		Runner:  service.NewRunner(),
	}
}

// NewViewContext returns a ui.ViewContext bound to this shell.
func (s *ActorShell) NewViewContext(win fyne.Window, app *state.AppState, fb *feedback.Host) *ui.ViewContext {
	return &ui.ViewContext{
		Window:   win,
		Output:   io.Discard,
		Runner:   s.Runner,
		Profile:  s.Profile,
		App:      app,
		Feedback: fb,
	}
}
