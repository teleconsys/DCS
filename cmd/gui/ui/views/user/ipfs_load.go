package user

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// IPFSLoadView uploads a local file to the running IPFS node.
func IPFSLoadView(vc *ui.ViewContext) fyne.CanvasObject {
	path, pathRow := ui.FilePickerRow(vc.Window, "/path/to/file")
	form := widget.NewForm(widget.NewFormItem("File", pathRow))

	result := ui.NewResultLabel()
	run := ui.RunButton(vc, "Upload to IPFS", "ipfs load-file",
		func() error {
			if strings.TrimSpace(path.Text) == "" {
				return errors.New("file path is required")
			}
			return nil
		},
		func(ctx context.Context, out io.Writer, _ state.ActorProfile) (string, error) {
			cid, err := service.LoadFile(ctx, path.Text, out)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("File pinned on IPFS.\n\nCID:\n%s", cid), nil
		}, ui.RunOpts{ResultLabel: result})
	return container.NewVBox(ui.Card("Upload + pin a file on the local IPFS node", form), result, run)
}
