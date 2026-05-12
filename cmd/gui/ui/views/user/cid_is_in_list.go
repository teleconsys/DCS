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

// CIDIsInListView checks whether a CID object id appears in the list.
func CIDIsInListView(vc *ui.ViewContext) fyne.CanvasObject {
	cidID := ui.NewAddressEntry(false, "0x… CID object id")
	form := widget.NewForm(widget.NewFormItem("CID object id", cidID))

	run := ui.RunButton(vc, "Check", "cid is-in-list",
		func() error { return cidID.Validate() },
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) error {
			_, err := service.CIDIsInList(ctx, snap, cidID.Text, out)
			return err
		})
	return container.NewVBox(ui.Card("Membership check (read-only)", form), run)
}
