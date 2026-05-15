package gc

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

// WhitelistHasView reads the whitelist via RPC (no signature needed).
func WhitelistHasView(vc *ui.ViewContext) fyne.CanvasObject {
	member := ui.NewAddressEntry(false, "0x… member address")
	form := widget.NewForm(widget.NewFormItem("Member", member))

	result := ui.NewResultLabel()
	run := ui.RunButton(vc, "Check", "whitelist has",
		func() error { return member.Validate() },
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) (string, error) {
			found, err := service.WhitelistHas(ctx, snap, member.Text, out)
			if err != nil {
				return "", err
			}
			if found {
				return "This address is on the whitelist.", nil
			}
			return "This address is not on the whitelist.", nil
		}, ui.RunOpts{ResultLabel: result})
	return container.NewVBox(ui.Card("Check whitelist membership", form), result, run)
}
