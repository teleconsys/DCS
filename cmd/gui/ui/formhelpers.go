package ui

import (
	"context"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
)

// Action is the canonical signature for an asynchronous action launched
// by a view. It receives a goroutine-local context, the live OutputView
// (as an io.Writer), and a snapshot of the actor profile taken at the
// moment the Run button was clicked.
type Action func(ctx context.Context, out io.Writer, snap state.ActorProfile) error

// RunButton builds a "Run" button that:
//  1. invokes `validate` synchronously; if it returns an error, the
//     error is shown to the user and the runner is not touched.
//  2. snapshots the actor profile.
//  3. runs `action` via vc.Runner, streaming output into vc.Output.
//
// The button is disabled while the action is in flight.
func RunButton(vc *ViewContext, label, title string,
	validate func() error,
	action Action,
) *widget.Button {
	var btn *widget.Button
	btn = widget.NewButton(label, func() {
		if validate != nil {
			if err := validate(); err != nil {
				dialog.ShowError(err, vc.Window)
				return
			}
		}
		snap := vc.Snapshot()
		btn.Disable()
		vc.Runner.Run(context.Background(), vc.Output, service.Job{
			Title: title,
			Fn: func(ctx context.Context, out io.Writer) error {
				return action(ctx, out, snap)
			},
		}, func(err error) {
			// Re-enable the button on the UI thread. Views that
			// produce a digest call vc.OnDigest themselves; the
			// runner doesn't try to be clever about it.
			_ = err
			fyne.Do(func() { btn.Enable() })
		})
	})
	return btn
}

// Card wraps a bold title above a vertical group of children.
func Card(title string, children ...fyne.CanvasObject) fyne.CanvasObject {
	header := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.NewVBox(append([]fyne.CanvasObject{header, widget.NewSeparator()}, children...)...)
}
