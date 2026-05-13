package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// EmptyState renders a centered placeholder (icon + title + hint +
// optional CTA) used when a list or grid has no rows yet.
func EmptyState(icon fyne.Resource, title, hint, ctaLabel string, ctaFn func()) fyne.CanvasObject {
	parts := make([]fyne.CanvasObject, 0, 4)
	if icon != nil {
		img := widget.NewIcon(icon)
		parts = append(parts, container.NewCenter(img))
	}
	t := widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	parts = append(parts, t)
	if hint != "" {
		h := widget.NewLabel(hint)
		h.Alignment = fyne.TextAlignCenter
		h.Wrapping = fyne.TextWrapWord
		parts = append(parts, h)
	}
	if ctaLabel != "" && ctaFn != nil {
		btn := widget.NewButton(ctaLabel, ctaFn)
		btn.Importance = widget.HighImportance
		parts = append(parts, container.NewCenter(btn))
	}
	return container.NewCenter(container.NewVBox(parts...))
}
