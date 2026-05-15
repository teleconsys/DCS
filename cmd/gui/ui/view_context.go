package ui

import (
	"bytes"
	"io"

	"fyne.io/fyne/v2"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
)

// ViewContext bundles every dependency a per-action view needs to wire
// itself up. The Profile pointer is always the *live* one (so edits in
// the identity panel are reflected), but each Run call should call
// Profile.Clone() before handing off to a goroutine.
type ViewContext struct {
	Window   fyne.Window
	Output   io.Writer
	Runner   *service.Runner
	Profile  *state.ActorProfile
	App      *state.AppState
	Feedback *feedback.Host
}

// Snapshot returns an immutable copy of the live profile, suitable for
// hand-off to a goroutine.
func (vc *ViewContext) Snapshot() state.ActorProfile {
	if vc == nil || vc.Profile == nil {
		return state.ActorProfile{}
	}
	return vc.Profile.Clone()
}

// TeeOutput returns a writer that duplicates writes to vc.Output and to
// capture, so UI code can show the same text the CLI prints (CIDs, digests,
// pin checks, etc.) while still honoring the global output sink.
func (vc *ViewContext) TeeOutput(capture *bytes.Buffer) io.Writer {
	if capture == nil {
		if vc == nil {
			return io.Discard
		}
		return vc.Output
	}
	if vc == nil {
		return capture
	}
	return io.MultiWriter(vc.Output, capture)
}

// View is a constructor for an action form (IPFS tabs, dialogs, etc.).
type View func(vc *ViewContext) fyne.CanvasObject
