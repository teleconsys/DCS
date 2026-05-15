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

// WhitelistAddView posts to the GC HTTP API.
func WhitelistAddView(vc *ui.ViewContext) fyne.CanvasObject {
	member := ui.NewAddressEntry(false, "0x… address to add")
	form := widget.NewForm(widget.NewFormItem("Member", member))

	run := ui.RunButton(vc, "Add to whitelist", "whitelist add",
		func() error { return member.Validate() },
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) (string, error) {
			return service.GCWhitelistAdd(ctx, snap, member.Text, out)
		})
	return container.NewVBox(ui.Card("Add member (GC-signed)", form), run)
}
