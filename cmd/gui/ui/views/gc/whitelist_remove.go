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

// WhitelistRemoveView posts to the GC HTTP API.
func WhitelistRemoveView(vc *ui.ViewContext) fyne.CanvasObject {
	member := ui.NewAddressEntry(false, "0x… address to remove")
	form := widget.NewForm(widget.NewFormItem("Member", member))

	run := ui.RunButton(vc, "Remove from whitelist", "whitelist remove",
		func() error { return member.Validate() },
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) error {
			return service.GCWhitelistRemove(ctx, snap, member.Text, out)
		})
	return container.NewVBox(ui.Card("Remove member (GC-signed)", form), run)
}
