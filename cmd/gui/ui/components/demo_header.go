package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// DemoPaneHeader is a slim title strip for Demo split panes (icon + title + accent line).
func DemoPaneHeader(title string, accent color.Color, icon fyne.Resource) fyne.CanvasObject {
	titleLbl := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	var row fyne.CanvasObject = titleLbl
	if icon != nil {
		row = container.NewHBox(widget.NewIcon(icon), titleLbl)
	}
	return container.NewVBox(
		container.NewPadded(row),
		AccentSeparator(accent),
	)
}
