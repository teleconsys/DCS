package components

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
)

// NewLivePhaseTimer returns a label that updates every second with phase + countdown.
func NewLivePhaseTimer(sum service.CIDSummary) *widget.Label {
	lbl := widget.NewLabel("")
	lbl.Wrapping = fyne.TextWrapWord
	update := func() {
		now := time.Now().UnixMilli()
		phase, cd := service.PhaseTimerLine(sum, now)
		if cd == "" {
			lbl.SetText(phase)
		} else {
			lbl.SetText(phase + " · " + cd)
		}
	}
	update()
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for range t.C {
			fyne.Do(update)
		}
	}()
	return lbl
}

// NewLiveCountdownLabel ticks every second; textFn should return the full line to display.
func NewLiveCountdownLabel(textFn func() string) *widget.Label {
	lbl := widget.NewLabel("")
	lbl.Wrapping = fyne.TextWrapWord
	refresh := func() {
		lbl.SetText(textFn())
		lbl.Refresh()
	}
	refresh()
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for range t.C {
			fyne.Do(refresh)
		}
	}()
	return lbl
}
