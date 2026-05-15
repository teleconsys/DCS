package user

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// AccountCoinsView lists coin objects owned by an address. The address
// is filled from the active User profile by default.
func AccountCoinsView(vc *ui.ViewContext) fyne.CanvasObject {
	address := ui.NewAddressEntry(false, "0x… owner address")
	address.SetText(vc.Profile.Address)

	form := widget.NewForm(widget.NewFormItem("Address", address))

	result := ui.NewResultLabel()
	run := ui.RunButton(vc, "List coins", "account coins",
		func() error {
			if strings.TrimSpace(address.Text) == "" {
				return errors.New("address is required")
			}
			return address.Validate()
		},
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) (string, error) {
			res, err := service.AccountCoins(ctx, snap.RPCURL, strings.TrimSpace(address.Text), 25*time.Second, out)
			if err != nil {
				return "", err
			}
			return ui.FormatAccountCoinsResult(res), nil
		}, ui.RunOpts{ResultLabel: result})

	return container.NewVBox(form, result, ui.ActionRow(run))
}
