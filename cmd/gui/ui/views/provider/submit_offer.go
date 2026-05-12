package provider

import (
	"context"
	"errors"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// SubmitOfferView builds an offer for the next epoch (Provider action).
func SubmitOfferView(vc *ui.ViewContext) fyne.CanvasObject {
	cidID := ui.NewAddressEntry(false, "0x… CID object id")
	amount := ui.NewAmountEntry("amount (nanos)", false)
	debug := widget.NewCheck("debug", nil)

	form := widget.NewForm(
		widget.NewFormItem("CID object id", cidID),
		widget.NewFormItem("Amount", amount),
		widget.NewFormItem("", debug),
	)

	run := ui.RunButton(vc, "Submit offer", "submit_offer",
		func() error {
			if err := cidID.Validate(); err != nil {
				return err
			}
			amt, _ := ui.ParseUint64(amount.Text)
			if amt == 0 {
				return errors.New("amount must be > 0")
			}
			return nil
		},
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) error {
			amt, _ := ui.ParseUint64(amount.Text)
			digest, err := service.SubmitOffer(ctx, snap, service.SubmitOfferForm{
				CIDObjectID: cidID.Text,
				Amount:      amt,
				Debug:       debug.Checked,
			}, out)
			if err == nil && digest != "" && vc.OnDigest != nil {
				vc.OnDigest(digest)
			}
			return err
		})
	return container.NewVBox(ui.Card("Submit a storage offer", form), run)
}
