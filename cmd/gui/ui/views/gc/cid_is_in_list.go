package gc

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

// CIDIsInListView checks whether a CID object id appears on the on-chain list.
func CIDIsInListView(vc *ui.ViewContext) fyne.CanvasObject {
	cidID := ui.NewAddressEntry(false, "0x… CID object id")
	form := widget.NewForm(widget.NewFormItem("CID object id", cidID))

	result := ui.NewResultLabel()
	run := ui.RunButton(vc, "Check", "cid is-in-list",
		func() error { return cidID.Validate() },
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) (string, error) {
			inList, err := service.CIDIsInList(ctx, snap, cidID.Text, out)
			if err != nil {
				return "", err
			}
			if inList {
				return "This CID object id is on the on-chain CID list.", nil
			}
			return "This CID object id is not on the on-chain CID list.", nil
		}, ui.RunOpts{ResultLabel: result})
	return container.NewVBox(ui.Card("On-chain CID list (read-only)", form), result, run)
}
