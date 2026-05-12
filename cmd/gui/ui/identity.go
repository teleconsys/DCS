package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/internal/wallet"
)

// IdentityPanel renders the active identity for a single actor. The
// panel is bound to a *state.ActorProfile pointer; edits in the form
// mutate the underlying profile so subsequent service calls see the
// new values.
//
// For GC, the panel shows endpoint + token fields. For User and
// Provider it shows alias + private key + derived address + gas coin
// id.
type IdentityPanel struct {
	win     fyne.Window
	profile *state.ActorProfile

	// signing widgets (User / Provider)
	aliasSel   *widget.Select
	privEntry  *widget.Entry
	derivedLbl *widget.Label
	addressEnt *widget.Entry
	gasEntry   *widget.Entry
	verifyBtn  *widget.Button

	// GC widgets
	gcEndpoint *widget.Entry
	gcToken    *widget.Entry

	// network widgets (shared)
	rpcEntry *widget.Entry

	canvas fyne.CanvasObject

	// onChanged is invoked on every meaningful edit so listeners (e.g.
	// the status bar) can re-render the actor summary.
	onChanged func()
}

// NewIdentityPanel builds the panel bound to the given profile pointer.
func NewIdentityPanel(win fyne.Window, prof *state.ActorProfile, onChanged func()) *IdentityPanel {
	p := &IdentityPanel{win: win, profile: prof, onChanged: onChanged}
	p.build()
	return p
}

// CanvasObject lets the panel be embedded.
func (p *IdentityPanel) CanvasObject() fyne.CanvasObject { return p.canvas }

// Profile returns the underlying profile pointer.
func (p *IdentityPanel) Profile() *state.ActorProfile { return p.profile }

// Snapshot returns an immutable copy of the current profile, ideal for
// hand-off to a background goroutine.
func (p *IdentityPanel) Snapshot() state.ActorProfile { return p.profile.Clone() }

func (p *IdentityPanel) build() {
	title := widget.NewLabelWithStyle(
		fmt.Sprintf("Identity — %s", p.profile.Actor),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	p.rpcEntry = widget.NewEntry()
	p.rpcEntry.SetText(p.profile.RPCURL)
	p.rpcEntry.OnChanged = func(s string) {
		p.profile.RPCURL = strings.TrimSpace(s)
		p.notify()
	}

	form := widget.NewForm()
	form.Append("RPC URL", p.rpcEntry)

	if p.profile.Actor == state.ActorGC {
		p.buildGC(form)
	} else {
		p.buildSigner(form)
	}

	p.canvas = container.NewVBox(title, form)
}

func (p *IdentityPanel) buildGC(form *widget.Form) {
	p.gcEndpoint = widget.NewEntry()
	p.gcEndpoint.SetPlaceHolder("http://host:port")
	p.gcEndpoint.SetText(p.profile.GCEndpoint)
	p.gcEndpoint.OnChanged = func(s string) {
		p.profile.GCEndpoint = strings.TrimSpace(s)
		p.notify()
	}

	p.gcToken = widget.NewPasswordEntry()
	p.gcToken.SetPlaceHolder("optional bearer token")
	p.gcToken.SetText(p.profile.GCToken)
	p.gcToken.OnChanged = func(s string) {
		p.profile.GCToken = strings.TrimSpace(s)
	}

	form.Append("GC endpoint", p.gcEndpoint)
	form.Append("GC token", p.gcToken)
}

func (p *IdentityPanel) buildSigner(form *widget.Form) {
	p.aliasSel = NewAliasSelect(p.profile.Actor, func(a state.AccountFile) {
		if err := service.LoadAccountIntoProfile(p.profile, a.Alias); err != nil {
			dialog.ShowError(err, p.win)
			return
		}
		p.privEntry.SetText(p.profile.PrivateKey)
		p.addressEnt.SetText(p.profile.Address)
		p.refreshDerived()
		p.notify()
	})

	priv, privBox := NewPrivateKeyEntry(p.profile.PrivateKey)
	p.privEntry = priv
	p.privEntry.OnChanged = func(s string) {
		p.profile.PrivateKey = strings.TrimSpace(s)
		p.refreshDerived()
		p.notify()
	}

	p.derivedLbl = widget.NewLabel("")
	p.refreshDerived()

	p.addressEnt = NewAddressEntry(true, "0x… (overrides derived)")
	p.addressEnt.SetText(p.profile.Address)
	p.addressEnt.OnChanged = func(s string) {
		p.profile.Address = strings.TrimSpace(s)
		p.notify()
	}

	p.gasEntry = NewAddressEntry(true, "0x… (signer's gas coin)")
	p.gasEntry.SetText(p.profile.GasCoinID)
	p.gasEntry.OnChanged = func(s string) {
		p.profile.GasCoinID = strings.TrimSpace(s)
		p.notify()
	}

	p.verifyBtn = widget.NewButton("Verify gas coin", p.verifyGasCoin)

	form.Append("Alias", p.aliasSel)
	form.Append("Private key", privBox)
	form.Append("Derived addr", p.derivedLbl)
	form.Append("Address override", p.addressEnt)
	form.Append("Gas coin id", p.gasEntry)
	form.Append("", p.verifyBtn)
}

func (p *IdentityPanel) refreshDerived() {
	pk := strings.TrimSpace(p.profile.PrivateKey)
	if pk == "" {
		p.derivedLbl.SetText("(no key)")
		return
	}
	priv, err := wallet.ResolvePrivateKeyStrict(pk)
	if err != nil {
		p.derivedLbl.SetText("invalid key")
		return
	}
	addr, err := wallet.PrivateKeyToAddress(priv)
	if err != nil {
		p.derivedLbl.SetText("invalid key")
		return
	}
	p.derivedLbl.SetText(addr)
}

func (p *IdentityPanel) verifyGasCoin() {
	prof := p.Snapshot()
	if strings.TrimSpace(prof.GasCoinID) == "" {
		dialog.ShowError(fmt.Errorf("gas coin id is empty"), p.win)
		return
	}
	if strings.TrimSpace(prof.PrivateKey) == "" {
		dialog.ShowError(fmt.Errorf("private key is empty"), p.win)
		return
	}

	// Derive a fresh address (don't trust manual override here).
	priv, err := wallet.ResolvePrivateKeyStrict(prof.PrivateKey)
	if err != nil {
		dialog.ShowError(fmt.Errorf("private key: %w", err), p.win)
		return
	}
	addr, err := wallet.PrivateKeyToAddress(priv)
	if err != nil {
		dialog.ShowError(fmt.Errorf("derive address: %w", err), p.win)
		return
	}

	p.verifyBtn.Disable()
	go func() {
		defer fyne.Do(func() { p.verifyBtn.Enable() })
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		gasID, err := wallet.ResolveGasCoinId(ctx, prof.GasCoinID, addr, prof.RPCURL)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, p.win)
				return
			}
			dialog.ShowInformation("gas coin verified",
				fmt.Sprintf("%s belongs to %s", gasID, addr), p.win)
		})
	}()
}

func (p *IdentityPanel) notify() {
	if p.onChanged != nil {
		p.onChanged()
	}
}
