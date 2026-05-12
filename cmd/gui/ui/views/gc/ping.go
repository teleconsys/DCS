package gc

import (
	"context"
	"io"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// PingView is a one-button view that calls iota_sc ping against the
// RPC URL configured on the active profile.
func PingView(vc *ui.ViewContext) fyne.CanvasObject {
	timeoutEntry := widget.NewEntry()
	timeoutEntry.SetText("5s")
	timeoutEntry.SetPlaceHolder("e.g. 5s")

	form := widget.NewForm(widget.NewFormItem("Timeout", timeoutEntry))

	run := ui.RunButton(vc, "Ping", "ping", nil,
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) error {
			to, err := time.ParseDuration(timeoutEntry.Text)
			if err != nil {
				to = 5 * time.Second
			}
			_, err = service.Ping(ctx, snap.RPCURL, to, out)
			return err
		})
	return container.NewVBox(ui.Card("Connectivity", form), run)
}
