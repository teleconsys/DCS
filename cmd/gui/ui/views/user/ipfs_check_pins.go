package user

import (
	"context"
	"fmt"
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

	result := ui.NewResultLabel()
	run := ui.RunButton(vc, "Check all pins", "ipfs check-pins", nil,
		func(ctx context.Context, out io.Writer, _ state.ActorProfile) (string, error) {
			sum, err := service.CheckPins(ctx, out)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Pins: %d total · %d intact · %d broken", sum.Total, sum.Intact, sum.Broken), nil
		}, ui.RunOpts{ResultLabel: result})
	return container.NewVBox(ui.Card("Verify pin integrity", desc), result, run)
}
