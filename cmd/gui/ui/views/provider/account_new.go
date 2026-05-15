package provider

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

// AccountNewView creates a fresh Provider-tagged account.
func AccountNewView(vc *ui.ViewContext) fyne.CanvasObject {
	alias := widget.NewEntry()
	alias.SetPlaceHolder("alias (letters, digits, '-', '_')")

	noFaucet := widget.NewCheck("Skip faucet", nil)

	amount := ui.NewAmountEntry("optional, 0 = server default", true)

	form := widget.NewForm(
		widget.NewFormItem("Alias", alias),
		widget.NewFormItem("", noFaucet),
		widget.NewFormItem("Faucet (nanos)", amount),
	)

	result := ui.NewResultLabel()
	run := ui.RunButton(vc, "Create account", "account new",
		func() error {
			if strings.TrimSpace(alias.Text) == "" {
				return errors.New("alias is required")
			}
			return nil
		},
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) (string, error) {
			amt, _ := ui.ParseUint64(amount.Text)
			acc, err := service.NewAccount(ctx, service.NewAccountForm{
				Alias:        alias.Text,
				NoFaucet:     noFaucet.Checked,
				FaucetAmount: amt,
				Role:         state.ActorProvider,
				FaucetURL:    snap.FaucetURL,
			}, out)
			if err != nil {
				return "", err
			}
			return ui.FormatAccountNewResult(acc), nil
		}, ui.RunOpts{ResultLabel: result})
	run.Importance = widget.HighImportance

	return container.NewVBox(form, result, ui.ActionRow(run))
}
