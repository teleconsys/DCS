package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/provider"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/user"
)

// Identity modal sizes: wide enough for full 0x addresses, gas coin IDs, and long private keys.
var (
	identityModalSignerSize = fyne.NewSize(820, 720)
	identityModalGCSize     = fyne.NewSize(560, 360)
)

const identityModalMinWidth float32 = 760

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
	var fb *feedback.Host
	if vc != nil {
		fb = vc.Feedback
	}
	panel := ui.NewIdentityPanel(win, p, fb, onChange)

	sections := []fyne.CanvasObject{panel.CanvasObject()}

	if vc != nil && (actor == state.ActorUser || actor == state.ActorProvider) {
		var walletTools fyne.CanvasObject
		switch actor {
		case state.ActorUser:
			walletTools = buildWalletSection(
				user.AccountNewView(vc),
				user.AccountCoinsView(vc),
			)
		case state.ActorProvider:
			walletTools = buildWalletSection(
				provider.AccountNewView(vc),
				provider.AccountCoinsView(vc),
			)
		}
		sections = append(sections,
			widget.NewSeparator(),
			ui.SectionHeader("Wallet", "Create an account file or list coin objects for an address."),
			walletTools,
		)
	}

	content := container.NewVBox(sections...)
	wide := container.New(ui.MinWidthLayout{MinW: identityModalMinWidth}, content)
	body := ui.ModalScroll(wide)

	d := dialog.NewCustom(identityModalTitle(actor), "Close", body, win)
	if vc != nil && (actor == state.ActorUser || actor == state.ActorProvider) {
		d.Resize(identityModalSignerSize)
	} else {
		d.Resize(identityModalGCSize)
	}
	d.Show()
}

func buildWalletSection(newAccount, listCoins fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(
		ui.SubsectionTitle("New account"),
		newAccount,
		widget.NewSeparator(),
		ui.SubsectionTitle("List coins"),
		listCoins,
	)
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
