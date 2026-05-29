package shell

import (
	"fmt"
	"image/color"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service/ipfsdaemon"
	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
)

// newIPFSDaemonRow returns an IPFS daemon status row (dot + label + toggle) and
// a cleanup func that must run when the settings dialog closes.
func newIPFSDaemonRow(fb *feedback.Host) (fyne.CanvasObject, func()) {
	var alive atomic.Bool
	alive.Store(true)

	okDot := color.NRGBA{R: 0x22, G: 0xC5, B: 0x5E, A: 0xFF}
	badDot := color.NRGBA{R: 0xEF, G: 0x44, B: 0x44, A: 0xFF}

	dot := canvas.NewCircle(badDot)
	dotCell := container.NewGridWrap(fyne.NewSize(10, 10), dot)
	statusLbl := widget.NewLabel(ipfsStatusStopped)
	statusGroup := container.NewHBox(container.NewCenter(dotCell), statusLbl)

	var toggleBtn *widget.Button

	refresh := func() {
		if !alive.Load() {
			return
		}
		up := ipfsdaemon.Reachable()
		if up {
			dot.FillColor = okDot
			statusLbl.SetText(ipfsStatusRunning)
			if ipfsdaemon.Default.Managed() {
				toggleBtn.SetText("Turn off")
				toggleBtn.Importance = widget.DangerImportance
				toggleBtn.Enable()
			} else {
				toggleBtn.SetText("Running")
				toggleBtn.Importance = widget.LowImportance
				toggleBtn.Disable()
				dot.Refresh()
				toggleBtn.Refresh()
				return
			}
		} else {
			dot.FillColor = badDot
			statusLbl.SetText(ipfsStatusStopped)
			toggleBtn.SetText("Turn on")
			toggleBtn.Importance = widget.SuccessImportance
		}
		dot.Refresh()
		toggleBtn.Enable()
		toggleBtn.Refresh()
	}

	scheduleRefresh := func() {
		fyne.Do(func() {
			refresh()
		})
	}

	cleanup := func() {
		alive.Store(false)
		ipfsdaemon.Default.SetOnExit(nil)
	}

	ipfsdaemon.Default.SetOnExit(func(err error) {
		_ = err
		if !alive.Load() {
			return
		}
		scheduleRefresh()
	})

	toggleBtn = widget.NewButton("Turn on", func() {
		toggleBtn.Disable()
		go func() {
			var err error
			if ipfsdaemon.Reachable() {
				if ipfsdaemon.Default.Managed() {
					err = ipfsdaemon.Default.Stop()
					if err != nil && !ipfsdaemon.Reachable() {
						err = nil
					}
				} else {
					err = fmt.Errorf("IPFS is already running outside DCS — stop it in your terminal first")
				}
			} else {
				err = ipfsdaemon.Default.Start()
				if err == nil {
					for i := 0; i < 30; i++ {
						if ipfsdaemon.Reachable() {
							break
						}
						time.Sleep(200 * time.Millisecond)
					}
					if !ipfsdaemon.Reachable() {
						err = fmt.Errorf("ipfs daemon started but API is not reachable yet")
					}
				}
			}
			if err != nil && fb != nil {
				fyne.Do(func() {
					if alive.Load() {
						fb.Show(feedback.Error, err.Error())
					}
				})
			}
			scheduleRefresh()
		}()
	})

	scheduleRefresh()

	row := container.NewBorder(
		nil, nil,
		widget.NewLabel("IPFS daemon"),
		toggleBtn,
		statusGroup,
	)
	return row, cleanup
}
