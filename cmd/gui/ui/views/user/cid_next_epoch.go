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

// CIDNextEpochView triggers an epoch transition. It performs the same
// pre-flight check (`CheckEpochTransitionAllowed`) the CLI does.
func CIDNextEpochView(vc *ui.ViewContext) fyne.CanvasObject {
	cidID := ui.NewAddressEntry(false, "0x… CID object id")
	form := widget.NewForm(widget.NewFormItem("CID object id", cidID))

	run := ui.RunButton(vc, "Transition", "cid next-epoch",
		func() error { return cidID.Validate() },
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) (string, error) {
			if err := service.CIDTransitionEpoch(ctx, snap, cidID.Text, out); err != nil {
				return "", err
			}
			return "Epoch transition submitted.", nil
		})
	return container.NewVBox(ui.Card("Advance to the next epoch", form), run)
}
