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

// CIDRemoveView removes a CID from the on-chain CID list.
func CIDRemoveView(vc *ui.ViewContext) fyne.CanvasObject {
	cidID := ui.NewAddressEntry(false, "0x… CID object id")
	form := widget.NewForm(widget.NewFormItem("CID object id", cidID))

	run := ui.RunButton(vc, "Remove", "cid remove",
		func() error { return cidID.Validate() },
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) error {
			digest, err := service.CIDRemove(ctx, snap, cidID.Text, out)
			if err == nil && digest != "" && vc.OnDigest != nil {
				vc.OnDigest(digest)
			}
			return err
		})
	return container.NewVBox(ui.Card("Remove a CID from the list", form), run)
}
