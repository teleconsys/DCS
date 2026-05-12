package user

import (
	"context"
	"errors"
	"io"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// AccountNewView creates a fresh User-tagged account.
func AccountNewView(vc *ui.ViewContext) fyne.CanvasObject {
	alias := widget.NewEntry()
	alias.SetPlaceHolder("alias (letters, digits, '-', '_')")

	noFaucet := widget.NewCheck("Skip faucet", nil)

	amount := ui.NewAmountEntry("faucet amount (optional, 0 = server default)", true)

	form := widget.NewForm(
		widget.NewFormItem("Alias", alias),
		widget.NewFormItem("", noFaucet),
		widget.NewFormItem("Faucet amount", amount),
	)

	run := ui.RunButton(vc, "Create account", "account new",
		func() error {
			if strings.TrimSpace(alias.Text) == "" {
				return errors.New("alias is required")
			}
			return nil
		},
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) error {
			amt, _ := ui.ParseUint64(amount.Text)
			_, err := service.NewAccount(ctx, service.NewAccountForm{
				Alias:        alias.Text,
				NoFaucet:     noFaucet.Checked,
				FaucetAmount: amt,
				Role:         state.ActorUser,
				FaucetURL:    snap.FaucetURL,
			}, out)
			return err
		})
	return container.NewVBox(ui.Card("Generate a new User account", form), run)
}
