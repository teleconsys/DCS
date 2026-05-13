package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	ftheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// OutputDrawer is the collapsible bottom panel that surfaces the
// currently-active actor's OutputView along with a "Last digest" pill.
// Inactive actors keep accumulating into their own OutputView; the
// drawer just swaps which view is visible when SetActor is called.
type OutputDrawer struct {
	app   *state.AppState
	views map[state.Actor]*ui.OutputView

	host        *fyne.Container // current view goes here
	digestLbl   *widget.Label
	toggle      *widget.Button
	root        fyne.CanvasObject
	collapsed   bool
	current     state.Actor
}

// NewOutputDrawer constructs the drawer. By default it starts collapsed
// (only the header bar visible) to maximize content vertical space.
func NewOutputDrawer(app *state.AppState, views map[state.Actor]*ui.OutputView) *OutputDrawer {
	d := &OutputDrawer{
		app:       app,
		views:     views,
		host:      container.NewMax(),
		digestLbl: widget.NewLabel("(no digest yet)"),
	}
	d.digestLbl.TextStyle = fyne.TextStyle{Monospace: true}
	d.toggle = widget.NewButtonWithIcon("Show log", ftheme.MenuExpandIcon(), d.Toggle)
	d.toggle.Importance = widget.LowImportance

	digestRow := container.NewHBox(
		widget.NewLabelWithStyle("Last digest:", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		d.digestLbl,
	)
	header := container.NewBorder(nil, nil, d.toggle, nil, digestRow)
	d.root = container.NewBorder(widget.NewSeparator(), nil, nil, nil, container.NewBorder(header, nil, nil, nil, d.host))
	d.SetActor(app.Registry.Current())
	d.Collapse()
	return d
}

// CanvasObject returns the drawer's root canvas object.
func (d *OutputDrawer) CanvasObject() fyne.CanvasObject { return d.root }

// SetActor swaps which actor's OutputView is on display and refreshes
// the digest label.
func (d *OutputDrawer) SetActor(a state.Actor) {
	d.current = a
	view, ok := d.views[a]
	if ok {
		d.host.Objects = []fyne.CanvasObject{view.CanvasObject()}
		d.host.Refresh()
	}
	d.RefreshDigest()
}

// RefreshDigest pulls the current actor's last digest from AppState.
// Safe to call from any goroutine.
func (d *OutputDrawer) RefreshDigest() {
	digest := d.app.LastDigest(d.current)
	if digest == "" {
		digest = "(no digest yet)"
	}
	fyne.Do(func() { d.digestLbl.SetText(digest) })
}

// Collapse hides the output content (header stays visible).
func (d *OutputDrawer) Collapse() {
	d.collapsed = true
	d.host.Hide()
	d.toggle.SetIcon(ftheme.MenuExpandIcon())
	d.toggle.SetText("Show log")
}

// Expand reveals the output content.
func (d *OutputDrawer) Expand() {
	d.collapsed = false
	d.host.Show()
	d.toggle.SetIcon(ftheme.MenuDropDownIcon())
	d.toggle.SetText("Hide log")
}

// Toggle flips collapsed/expanded.
func (d *OutputDrawer) Toggle() {
	if d.collapsed {
		d.Expand()
	} else {
		d.Collapse()
	}
}
