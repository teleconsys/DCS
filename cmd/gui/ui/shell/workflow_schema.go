package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	ftheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/assets/diagrams"
	"github.com/teleconsys/DCS/cmd/gui/ui/components"
)

// OpenWorkflowMap shows the embedded CID lifecycle diagram in a zoomable window.
func OpenWorkflowMap(parent fyne.Window) {
	bmp, err := diagrams.LifecycleBitmap()
	if err != nil {
		dialog.ShowError(err, parent)
		return
	}

	viewer := components.NewZoomImage(bmp)
	zoomLbl := widget.NewLabel("100%")
	viewer.OnZoomChanged(func(z float32) {
		zoomLbl.SetText(fmt.Sprintf("%.0f%%", z*100))
	})

	zoomOut := widget.NewButtonWithIcon("", ftheme.ContentRemoveIcon(), viewer.ZoomOut)
	zoomIn := widget.NewButtonWithIcon("", ftheme.ContentAddIcon(), viewer.ZoomIn)
	fitBtn := widget.NewButton("Fit", viewer.ResetZoom)
	hint := widget.NewLabel("Scroll to zoom · drag to pan")
	hint.Importance = widget.LowImportance

	footer := container.NewHBox(
		zoomOut,
		zoomLbl,
		zoomIn,
		fitBtn,
		layout.NewSpacer(),
		hint,
	)
	content := container.NewBorder(nil, footer, nil, nil, viewer)

	win := fyne.CurrentApp().NewWindow("DCS — CID Lifecycle")
	win.SetContent(content)

	size := parent.Canvas().Size()
	w := size.Width * 0.92
	h := size.Height * 0.92
	if w < 960 {
		w = 960
	}
	if h < 600 {
		h = 600
	}
	win.Resize(fyne.NewSize(w, h))
	win.CenterOnScreen()
	win.Show()
}
