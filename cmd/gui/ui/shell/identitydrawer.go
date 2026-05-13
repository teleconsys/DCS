package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// OpenIdentity displays the identity panel for the given actor in a
// modal sheet. The panel mutates the live ActorProfile in place; the
// onChange callback runs whenever any field changes so the AppBar can
// refresh actor-specific chrome (for example the Identity caption).
func OpenIdentity(win fyne.Window, app *state.AppState, actor state.Actor, onChange func()) {
	p := app.Registry.Profile(actor)
	panel := ui.NewIdentityPanel(win, p, onChange)
	d := dialog.NewCustom("Identity — "+actor.String(), "Close", panel.CanvasObject(), win)
	d.Resize(fyne.NewSize(520, 560))
	d.Show()
}
