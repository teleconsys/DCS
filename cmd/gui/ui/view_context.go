package ui

import (
	"io"

	"fyne.io/fyne/v2"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
)

// ViewContext bundles every dependency a per-action view needs to wire
// itself up. The Profile pointer is always the *live* one (so edits in
// the identity panel are reflected), but each Run call should call
// Profile.Clone() before handing off to a goroutine.
type ViewContext struct {
	Window  fyne.Window
	Output  io.Writer
	Runner  *service.Runner
	Profile *state.ActorProfile
	App     *state.AppState
}

// Snapshot returns an immutable copy of the live profile, suitable for
// hand-off to a goroutine.
func (vc *ViewContext) Snapshot() state.ActorProfile {
	if vc == nil || vc.Profile == nil {
		return state.ActorProfile{}
	}
	return vc.Profile.Clone()
}

// View is a constructor for an action form (IPFS tabs, dialogs, etc.).
type View func(vc *ViewContext) fyne.CanvasObject
