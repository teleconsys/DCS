package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/provider"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/user"
)

// OpenIdentity displays the identity panel for the given actor in a
// modal sheet. The panel mutates the live ActorProfile in place; the
// onChange callback runs whenever any field changes so the AppBar can
// refresh actor-specific chrome (for example the Identity caption).
//
// When vc is non-nil and the actor is User or Provider, the modal also
// includes wallet helpers (new account, list coins) wired to that actor's
// ViewContext.
func OpenIdentity(win fyne.Window, app *state.AppState, actor state.Actor, vc *ui.ViewContext, onChange func()) {
	p := app.Registry.Profile(actor)
	panel := ui.NewIdentityPanel(win, p, onChange)

	base := panel.CanvasObject()
	var body fyne.CanvasObject = base
	if vc != nil && (actor == state.ActorUser || actor == state.ActorProvider) {
		walletHdr := widget.NewLabelWithStyle("Wallet", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		var walletBlock fyne.CanvasObject
		switch actor {
		case state.ActorUser:
			walletBlock = container.NewVBox(
				walletHdr,
				user.AccountNewView(vc),
				user.AccountCoinsView(vc),
			)
		case state.ActorProvider:
			walletBlock = container.NewVBox(
				walletHdr,
				provider.AccountNewView(vc),
				provider.AccountCoinsView(vc),
			)
		}
		body = container.NewVScroll(container.NewVBox(
			base,
			widget.NewSeparator(),
			walletBlock,
		))
	}

	d := dialog.NewCustom(identityModalTitle(actor), "Close", body, win)
	if vc != nil && (actor == state.ActorUser || actor == state.ActorProvider) {
		d.Resize(fyne.NewSize(520, 720))
	} else {
		d.Resize(fyne.NewSize(520, 560))
	}
	d.Show()
}

func identityModalTitle(a state.Actor) string {
	switch a {
	case state.ActorUser:
		return "User info"
	case state.ActorProvider:
		return "Provider info"
	case state.ActorGC:
		return "Admin info"
	default:
		return "Identity"
	}
}
