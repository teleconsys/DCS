package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui/shell"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/provider"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/user"
)

// demoSideClamp caps the minimum width reported by each Demo pane (User or
// Provider) so the outer HSplit can move; content is still laid out at the
// full width the split assigns.
type demoSideClamp struct {
	MinW float32
}

func (d demoSideClamp) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	ms := objects[0].MinSize()
	const minH = 200
	return fyne.NewSize(d.MinW, fyne.Max(minH, ms.Height))
}

func (d demoSideClamp) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Resize(size)
	objects[0].Move(fyne.NewPos(0, 0))
}

// demoHostLayout wraps the User|Provider HSplit so AppTabs does not inherit
// the split's huge intrinsic width (sum of both panels), while every
// layout pass still gives the split the full allocated size. A plain
// HScroll uses max(contentMin, viewport) width, which kept the inner
// HSplit wider than the visible area and broke divider / nested resizing.
type demoHostLayout struct{}

func (demoHostLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	ms := objects[0].MinSize()
	const minW, minH = 520, 280
	return fyne.NewSize(minW, fyne.Max(minH, ms.Height))
}

func (demoHostLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	o := objects[0]
	o.Resize(size)
	o.Move(fyne.NewPos(0, 0))
}

// buildDemo lays out User and Provider side-by-side. The User side uses
// DashboardForDemo: single-column CID tiles and IPFS/Tools stacked vertically,
// while keeping the usual rail | My CIDs split.
func buildDemo(win fyne.Window, app *state.AppState, shells map[state.Actor]*shell.ActorShell) fyne.CanvasObject {
	userVC := shells[state.ActorUser].NewViewContext(win, app)
	provVC := shells[state.ActorProvider].NewViewContext(win, app)

	userBody := user.DashboardForDemo(userVC)
	provBody := provider.Dashboard(provVC)

	leftHdr := widget.NewLabelWithStyle("◀ User", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	rightHdr := widget.NewLabelWithStyle("Provider ▶", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	left := container.New(demoSideClamp{MinW: 240}, container.NewBorder(leftHdr, nil, nil, nil, container.NewPadded(userBody)))
	right := container.New(demoSideClamp{MinW: 240}, container.NewBorder(rightHdr, nil, nil, nil, container.NewPadded(provBody)))
	split := container.NewHSplit(left, right)
	split.SetOffset(0.5)
	return container.New(demoHostLayout{}, split)
}
