// Package wizards hosts multi-step modals reused across actor
// dashboards. Each wizard is a top-level dialog.Dialog that drives a
// service.* call on completion.
package wizards

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	ftheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// NewCIDWizard launches the "Upload new content" wizard. It walks the
// user through (1) source — upload file or paste CID — (2) epochs (with
// auto default), (3) initial budget, then (4) submits the on-chain
// pipeline via service.CIDCreate. Steps are presented as labeled
// sections in a single scrollable form to keep the implementation
// straightforward across Fyne versions.
//
// onDone runs after a successful creation so the caller can refresh
// the "My CIDs" grid.
func NewCIDWizard(vc *ui.ViewContext, onDone func()) {
	// Step 1 — source
	srcRadio := widget.NewRadioGroup(
		[]string{"Upload a file from disk", "Use an existing IPFS CID"},
		nil,
	)
	srcRadio.SetSelected("Upload a file from disk")

	filePath := widget.NewEntry()
	filePath.SetPlaceHolder("/path/to/file")
	pickBtn := widget.NewButtonWithIcon("Browse…", ftheme.FolderOpenIcon(), func() {
		d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil || r == nil {
				return
			}
			defer r.Close()
			filePath.SetText(r.URI().Path())
		}, vc.Window)
		d.Show()
	})
	pathRow := container.NewBorder(nil, nil, nil, pickBtn, filePath)

	cidEntry := widget.NewEntry()
	cidEntry.SetPlaceHolder("Qm… / bafy… / CIDv1")

	sourceContent := container.NewMax(pathRow)
	srcRadio.OnChanged = func(sel string) {
		if strings.HasPrefix(sel, "Upload") {
			sourceContent.Objects = []fyne.CanvasObject{pathRow}
		} else {
			sourceContent.Objects = []fyne.CanvasObject{cidEntry}
		}
		sourceContent.Refresh()
	}

	// Step 2 — epochs
	autoEpoch := widget.NewCheck("Auto epochs  ·  starts in 2 min, lasts 20 min", nil)
	autoEpoch.SetChecked(true)
	epochStart := widget.NewEntry()
	epochStart.SetPlaceHolder("unix ms")
	epochEnd := widget.NewEntry()
	epochEnd.SetPlaceHolder("unix ms")
	epochStart.Disable()
	epochEnd.Disable()
	autoEpoch.OnChanged = func(b bool) {
		if b {
			epochStart.Disable()
			epochEnd.Disable()
		} else {
			epochStart.Enable()
			epochEnd.Enable()
		}
	}

	// Step 3 — budget
	amount := widget.NewEntry()
	amount.SetText("100000")
	amount.SetPlaceHolder("IOTA nanos")

	stepHdr := func(s string) *widget.Label {
		return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}

	formBody := container.NewVBox(
		stepHdr("Step 1 — Source"),
		srcRadio,
		sourceContent,
		widget.NewSeparator(),
		stepHdr("Step 2 — Epochs"),
		autoEpoch,
		widget.NewForm(
			widget.NewFormItem("Epoch start (ms)", epochStart),
			widget.NewFormItem("Epoch end (ms)", epochEnd),
		),
		widget.NewSeparator(),
		stepHdr("Step 3 — Initial budget"),
		widget.NewForm(widget.NewFormItem("Amount (nanos)", amount)),
		widget.NewSeparator(),
		stepHdr("Step 4 — Review & submit"),
		widget.NewLabel("Click Create to upload (if needed), create the CID object on chain, and register it."),
	)
	scroll := container.NewVScroll(formBody)

	d := dialog.NewCustomConfirm("Upload new content", "Create", "Cancel", scroll,
		func(ok bool) {
			if !ok {
				return
			}
			f, err := collectForm(srcRadio, filePath, cidEntry, autoEpoch, epochStart, epochEnd, amount)
			if err != nil {
				dialog.ShowError(err, vc.Window)
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
				res, err := service.CIDCreate(ctx, vc.Snapshot(), f, vc.Output)
				if err != nil {
					fyne.Do(func() { dialog.ShowError(err, vc.Window) })
					return
				}
				fyne.Do(func() {
					dialog.ShowInformation("CID created",
						fmt.Sprintf("CID %s\nObject %s", res.CIDStr, res.CIDID),
						vc.Window)
				})
				if onDone != nil {
					onDone()
				}
			}()
		}, vc.Window)
	d.Resize(fyne.NewSize(720, 620))
	d.Show()
}

func collectForm(srcRadio *widget.RadioGroup, filePath, cidEntry *widget.Entry,
	autoEpoch *widget.Check, epochStart, epochEnd, amount *widget.Entry) (service.CIDCreateForm, error) {
	var f service.CIDCreateForm
	if strings.HasPrefix(srcRadio.Selected, "Upload") {
		f.FilePath = strings.TrimSpace(filePath.Text)
		if f.FilePath == "" {
			return f, errors.New("file path is required")
		}
	} else {
		f.CID = strings.TrimSpace(cidEntry.Text)
		if f.CID == "" {
			return f, errors.New("CID is required")
		}
	}
	amt, _ := strconv.ParseUint(strings.TrimSpace(amount.Text), 10, 64)
	if amt == 0 {
		return f, errors.New("amount must be > 0")
	}
	f.Amount = amt
	if !autoEpoch.Checked {
		es, err := strconv.ParseUint(strings.TrimSpace(epochStart.Text), 10, 64)
		if err != nil || es == 0 {
			return f, errors.New("epoch start must be a positive unix-ms value")
		}
		ee, err := strconv.ParseUint(strings.TrimSpace(epochEnd.Text), 10, 64)
		if err != nil || ee <= es {
			return f, errors.New("epoch end must be > epoch start")
		}
		f.EpochStart = es
		f.EpochEnd = ee
	}
	return f, nil
}
