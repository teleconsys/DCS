package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

type topBoundLayout struct{}

func (topBoundLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	return objects[0].MinSize()
}

func (topBoundLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	c := objects[0]
	h := c.MinSize().Height
	c.Resize(fyne.NewSize(size.Width, h))
	c.Move(fyne.NewPos(0, 0))
}

// TopBound wraps child so a tall parent (e.g. a scroll viewport) does not
// stretch it vertically: the child keeps its natural height and stays
// top-aligned, avoiding a dead band under the last control.
func TopBound(child fyne.CanvasObject) fyne.CanvasObject {
	return container.New(topBoundLayout{}, child)
}
