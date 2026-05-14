package shell

import (
	"context"
	"errors"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
)

// OpenAppSettings shows a modal with application-wide settings. For now
// this is the chain RPC URL. Save writes it to all actor profiles, refreshes
// the connection indicator (via onRPCSaved), and closes the dialog.
// Cancel closes without applying edits.
// "Ping RPC" tests the URL currently in the RPC field (saved or not).
func OpenAppSettings(win fyne.Window, app *state.AppState, onRPCSaved func()) {
	if app == nil {
		return
	}
	reg := app.Registry
	rpc := widget.NewEntry()
	rpc.SetPlaceHolder("https://…")
	rpc.SetText(strings.TrimSpace(reg.Profile(state.ActorGC).RPCURL))

	timeout := widget.NewEntry()
	timeout.SetText("5s")

	pingLbl := widget.NewLabel("")
	pingLbl.Wrapping = fyne.TextWrapWord

	rpcForm := widget.NewForm(widget.NewFormItem("RPC URL", rpc))

	timeoutLbl := widget.NewLabel("Timeout")

	var pingBtn *widget.Button
	pingBtn = widget.NewButton("Ping RPC", func() {
		url := strings.TrimSpace(rpc.Text)
		if url == "" {
			dialog.ShowError(errors.New("enter an RPC URL"), win)
			return
		}
		to, err := time.ParseDuration(strings.TrimSpace(timeout.Text))
		if err != nil || to <= 0 {
			to = 5 * time.Second
		}
		pingBtn.Disable()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var b strings.Builder
			_, err := service.Ping(ctx, url, to, &b)
			msg := strings.TrimSpace(b.String())
			fyne.Do(func() {
				pingBtn.Enable()
				if err != nil {
					dialog.ShowError(err, win)
					pingLbl.SetText("")
					return
				}
				if msg != "" {
					pingLbl.SetText(msg)
				} else {
					pingLbl.SetText("OK")
				}
			})
		}()
	})

	var d dialog.Dialog
	saveBtn := widget.NewButton("Save", func() {
		reg.SyncRPCURL(rpc.Text)
		if onRPCSaved != nil {
			fyne.Do(onRPCSaved)
		}
		if d != nil {
			d.Hide()
		}
	})
	saveBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButton("Cancel", func() {
		if d != nil {
			d.Hide()
		}
	})

	pair := container.NewHBox(saveBtn, cancelBtn)
	buttons := container.NewHBox(layout.NewSpacer(), pair, layout.NewSpacer())

	pingRow := container.NewHBox(timeoutLbl, timeout, layout.NewSpacer(), pingBtn)

	body := container.NewVBox(
		rpcForm,
		pingRow,
		pingLbl,
		widget.NewSeparator(),
		buttons,
	)

	d = dialog.NewCustomWithoutButtons("Settings", body, win)
	d.Resize(fyne.NewSize(480, 320))
	d.Show()
}
