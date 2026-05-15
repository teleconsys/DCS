package components

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// NewLiveGatedButton shows label when enabledFn is true, otherwise disabledLbl.
// enabledFn is re-evaluated every second.
func NewLiveGatedButton(label, disabledLbl string, enabledFn func() bool, onTap func()) fyne.CanvasObject {
	btn := widget.NewButton(label, onTap)
	btn.Importance = widget.HighImportance
	wait := widget.NewLabel(disabledLbl)
	host := container.NewStack(btn, wait)

	apply := func() {
		if enabledFn() {
			btn.Show()
			btn.Enable()
			wait.Hide()
		} else {
			btn.Hide()
			wait.Show()
		}
		host.Refresh()
	}
	apply()
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for range t.C {
			fyne.Do(apply)
		}
	}()
	return host
}
