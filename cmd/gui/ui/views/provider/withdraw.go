package provider

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

// WithdrawView claims payment from a fulfilled offer (Provider action).
func WithdrawView(vc *ui.ViewContext) fyne.CanvasObject {
	cidID := ui.NewAddressEntry(false, "0x… CID object id")
	idx := ui.NewAmountEntry("payment index (0-based)", true)
	debug := widget.NewCheck("debug", nil)

	form := widget.NewForm(
		widget.NewFormItem("CID object id", cidID),
		widget.NewFormItem("Index", idx),
		widget.NewFormItem("", debug),
	)

	run := ui.RunButton(vc, "Withdraw", "withdraw",
		func() error { return cidID.Validate() },
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) error {
			i, _ := ui.ParseUint64(idx.Text)
			_, err := service.Withdraw(ctx, snap, service.OfferIndexForm{
				CIDObjectID: cidID.Text,
				Index:       i,
				Debug:       debug.Checked,
			}, out)
			return err
		})
	return container.NewVBox(ui.Card("Withdraw payment", form), run)
}
