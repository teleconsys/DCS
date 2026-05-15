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

// ApproveOfferView approves an entry from next_epoch_offers.
func ApproveOfferView(vc *ui.ViewContext) fyne.CanvasObject {
	cidID := ui.NewAddressEntry(false, "0x… CID object id")
	idx := ui.NewAmountEntry("offer index (0-based)", true)
	debug := widget.NewCheck("debug", nil)

	form := widget.NewForm(
		widget.NewFormItem("CID object id", cidID),
		widget.NewFormItem("Index", idx),
		widget.NewFormItem("", debug),
	)

	run := ui.RunButton(vc, "Approve offer", "approve_offer",
		func() error { return cidID.Validate() },
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) (string, error) {
			i, _ := ui.ParseUint64(idx.Text)
			d, err := service.ApproveOffer(ctx, snap, service.OfferIndexForm{
				CIDObjectID: cidID.Text,
				Index:       i,
				Debug:       debug.Checked,
			}, out)
			if err != nil {
				return "", err
			}
			return ui.FormatTxDigest(d), nil
		})
	return container.NewVBox(ui.Card("Approve a provider offer", form), run)
}
