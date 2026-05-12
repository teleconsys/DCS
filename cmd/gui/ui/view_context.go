package ui

import (
	"fyne.io/fyne/v2"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
)

// ViewContext bundles every dependency a per-action view needs to wire
// itself up. The Profile pointer is always the *live* one (so edits in
// the identity panel are reflected), but each Run call should call
// Profile.Clone() before handing off to a goroutine.
type ViewContext struct {
	Window   fyne.Window
	Output   *OutputView
	Runner   *service.Runner
	Profile  *state.ActorProfile
	App      *state.AppState
	OnDigest func(digest string)
}

// Snapshot returns an immutable copy of the live profile, suitable for
// hand-off to a goroutine.
func (vc *ViewContext) Snapshot() state.ActorProfile {
	if vc == nil || vc.Profile == nil {
		return state.ActorProfile{}
	}
	return vc.Profile.Clone()
}

// View is a constructor for an action's form. Returning a fyne object
// lets the workspace assembler embed it under a content stack.
type View func(vc *ViewContext) fyne.CanvasObject

// NamedView pairs a sidebar label with its View builder. View
// subpackages expose an `All()` returning a slice of these so the
// workspace can wire them up.
type NamedView struct {
	Name    string
	Builder View
}
