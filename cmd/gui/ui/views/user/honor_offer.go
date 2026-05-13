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

// HonorOfferView honors a confirmed offer in the current epoch.
func HonorOfferView(vc *ui.ViewContext) fyne.CanvasObject {
	cidID := ui.NewAddressEntry(false, "0x… CID object id")
	idx := ui.NewAmountEntry("offer index (0-based)", true)
	debug := widget.NewCheck("debug", nil)

	form := widget.NewForm(
		widget.NewFormItem("CID object id", cidID),
		widget.NewFormItem("Index", idx),
		widget.NewFormItem("", debug),
	)

	run := ui.RunButton(vc, "Honor offer", "honor_offer",
		func() error { return cidID.Validate() },
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) error {
			i, _ := ui.ParseUint64(idx.Text)
			_, err := service.HonorOffer(ctx, snap, service.OfferIndexForm{
				CIDObjectID: cidID.Text,
				Index:       i,
				Debug:       debug.Checked,
			}, out)
			return err
		})
	return container.NewVBox(ui.Card("Honor a confirmed offer", form), run)
}
