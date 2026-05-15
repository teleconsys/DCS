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

// CIDCreateView mirrors the CLI's two-mode `cid create` command:
// either upload a local file to IPFS first (mode = "path"), or supply
// an existing CID directly (mode = "cid").
func CIDCreateView(vc *ui.ViewContext) fyne.CanvasObject {
	// File mode
	path, pathRow := ui.FilePickerRow(vc.Window, "/path/to/file (mode: path)")

	// CID mode
	cidEntry := widget.NewEntry()
	cidEntry.SetPlaceHolder("Qm… or bafy… (mode: cid)")

	mode := widget.NewRadioGroup([]string{"path", "cid"}, nil)
	mode.Horizontal = true
	mode.SetSelected("path")
	mode.OnChanged = func(s string) {
		switch s {
		case "path":
			path.Enable()
			cidEntry.Disable()
		case "cid":
			path.Disable()
			cidEntry.Enable()
		}
	}
	cidEntry.Disable()

	epochStart := ui.NewAmountEntry("epoch start (ms; 0 = auto)", true)
	epochEnd := ui.NewAmountEntry("epoch end (ms; 0 = auto)", true)
	amount := ui.NewAmountEntry("initial coin amount (nanos; default 100000)", true)

	form := widget.NewForm(
		widget.NewFormItem("Mode", mode),
		widget.NewFormItem("File path", pathRow),
		widget.NewFormItem("CID", cidEntry),
		widget.NewFormItem("Epoch start", epochStart),
		widget.NewFormItem("Epoch end", epochEnd),
		widget.NewFormItem("Initial amount", amount),
	)

	run := ui.RunButton(vc, "Create CID", "cid create",
		func() error {
			switch mode.Selected {
			case "path":
				if strings.TrimSpace(path.Text) == "" {
					return errors.New("file path is required when mode=path")
				}
			case "cid":
				if strings.TrimSpace(cidEntry.Text) == "" {
					return errors.New("CID is required when mode=cid")
				}
			default:
				return errors.New("mode must be path or cid")
			}
			return nil
		},
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) (string, error) {
			es, _ := ui.ParseUint64(epochStart.Text)
			ee, _ := ui.ParseUint64(epochEnd.Text)
			amt, _ := ui.ParseUint64(amount.Text)

			form := service.CIDCreateForm{
				EpochStart: es,
				EpochEnd:   ee,
				Amount:     amt,
			}
			if mode.Selected == "path" {
				form.FilePath = path.Text
			} else {
				form.CID = cidEntry.Text
			}
			res, err := service.CIDCreate(ctx, snap, form, out)
			if err != nil {
				return "", err
			}
			return ui.FormatCIDCreateResult(res), nil
		})
	return container.NewVBox(ui.Card("Create a new CID object", form), run)
}
