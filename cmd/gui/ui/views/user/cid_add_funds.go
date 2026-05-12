package user

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

// CIDAddFundsView deposits IOTA into a CID's storage budget.
func CIDAddFundsView(vc *ui.ViewContext) fyne.CanvasObject {
	cidID := ui.NewAddressEntry(false, "0x… CID object id")
	amount := ui.NewAmountEntry("amount (nanos)", false)

	form := widget.NewForm(
		widget.NewFormItem("CID object id", cidID),
		widget.NewFormItem("Amount", amount),
	)

	run := ui.RunButton(vc, "Deposit", "cid add-funds",
		func() error {
			if err := cidID.Validate(); err != nil {
				return err
			}
			if err := amount.Validate(); err != nil {
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
			digest, err := service.CIDAddFunds(ctx, snap, cidID.Text, amt, out)
			if err == nil && digest != "" && vc.OnDigest != nil {
				vc.OnDigest(digest)
			}
			return err
		})
	return container.NewVBox(ui.Card("Top up a CID storage budget", form), run)
}
