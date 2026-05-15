package ui

import "fyne.io/fyne/v2"

// MinWidthLayout lays out a single child at least MinW wide (fills allocated width).
type MinWidthLayout struct {
	MinW float32
}

func (l MinWidthLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	ms := objects[0].MinSize()
	return fyne.NewSize(fyne.Max(l.MinW, ms.Width), ms.Height)
}

func (l MinWidthLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Resize(size)
	objects[0].Move(fyne.NewPos(0, 0))
}
