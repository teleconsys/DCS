package user

import (
	"context"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// IPFSCheckPinsView iterates pins on the local IPFS node.
func IPFSCheckPinsView(vc *ui.ViewContext) fyne.CanvasObject {
	desc := widget.NewLabel("Walks every pin on the local IPFS node and reports its status.")
	desc.Wrapping = fyne.TextWrapWord

	run := ui.RunButton(vc, "Check all pins", "ipfs check-pins", nil,
		func(ctx context.Context, out io.Writer, _ state.ActorProfile) error {
			_, err := service.CheckPins(ctx, out)
			return err
		})
	return container.NewVBox(ui.Card("Verify pin integrity", desc), run)
}
