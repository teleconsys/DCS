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

// IPFSCheckCIDView verifies that a CID is pinned locally.
func IPFSCheckCIDView(vc *ui.ViewContext) fyne.CanvasObject {
	cidEntry := widget.NewEntry()
	cidEntry.SetPlaceHolder("Qm… or bafy…")

	form := widget.NewForm(widget.NewFormItem("CID", cidEntry))

	result := ui.NewResultLabel()
	run := ui.RunButton(vc, "Check CID", "ipfs check-cid",
		func() error {
			if strings.TrimSpace(cidEntry.Text) == "" {
				return errors.New("CID is required")
			}
			return nil
		},
		func(ctx context.Context, out io.Writer, _ state.ActorProfile) (string, error) {
			pinned, err := service.CheckCid(ctx, cidEntry.Text, out)
			if err != nil {
				return "", err
			}
			if pinned {
				return "This CID is pinned on the local IPFS node.", nil
			}
			return "This CID is not pinned on the local IPFS node.", nil
		}, ui.RunOpts{ResultLabel: result})
	return container.NewVBox(ui.Card("Is a CID pinned locally?", form), result, run)
}
