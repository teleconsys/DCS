package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	ftheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
	"github.com/teleconsys/DCS/internal/wallet"
)

// Minimum width for signer identity fields (0x address, gas coin id, private key).
const identityFieldMinWidth float32 = 680

// IdentityPanel renders the active identity for a single actor. The
// panel is bound to a *state.ActorProfile pointer; edits in the form
// mutate the underlying profile so subsequent service calls see the
// new values.
//
// For GC, the panel shows endpoint + token fields. For User and
// Provider it shows alias + private key + derived address + gas coin
// id. Chain RPC is edited in Settings (gear).
type IdentityPanel struct {
	win      fyne.Window
	feedback *feedback.Host
	profile  *state.ActorProfile

	// signing widgets (User / Provider)
	aliasSel   *widget.Select
	privEntry  *widget.Entry
	derivedLbl *widget.Label
	addressEnt *widget.Entry
	gasEntry   *widget.Entry
	verifyBtn  *widget.Button
	verifyLbl  *widget.Label

	// GC widgets
	gcEndpoint *widget.Entry
	gcToken    *widget.Entry

	canvas fyne.CanvasObject

	// onChanged is invoked on every meaningful edit so listeners (e.g.
	// the status bar) can re-render the actor summary.
	onChanged func()
}

// NewIdentityPanel builds the panel bound to the given profile pointer.
func NewIdentityPanel(win fyne.Window, prof *state.ActorProfile, fb *feedback.Host, onChanged func()) *IdentityPanel {
	p := &IdentityPanel{win: win, feedback: fb, profile: prof, onChanged: onChanged}
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
	if p.profile.Actor == state.ActorGC {
		p.canvas = p.buildGC()
	} else {
		p.canvas = p.buildSigner()
	}
}

func (p *IdentityPanel) buildGC() fyne.CanvasObject {
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

	form := widget.NewForm(
		widget.NewFormItem("Endpoint", p.gcEndpoint),
		widget.NewFormItem("Token", p.gcToken),
	)

	return container.NewVBox(
		SectionHeader("Ground Control API", "HTTP endpoint for whitelist and membership calls."),
		form,
	)
}

func (p *IdentityPanel) buildSigner() fyne.CanvasObject {
	p.aliasSel = NewAliasSelect(p.profile.Actor, func(a state.AccountFile) {
		if err := service.LoadAccountIntoProfile(p.profile, a.Alias); err != nil {
			if p.feedback != nil {
				p.feedback.Show(feedback.Error, err.Error())
			}
			return
		}
		p.privEntry.SetText(p.profile.PrivateKey)
		p.addressEnt.SetText(p.profile.Address)
		p.refreshDerived()
		p.notify()
	})

	priv, privBox := NewPrivateKeyEntry(p.profile.PrivateKey)
	p.privEntry = priv
	privBox = widenIdentityField(privBox)
	p.privEntry.OnChanged = func(s string) {
		p.profile.PrivateKey = strings.TrimSpace(s)
		p.refreshDerived()
		p.notify()
	}

	p.derivedLbl = widget.NewLabel("")
	p.derivedLbl.Wrapping = fyne.TextWrapBreak
	p.refreshDerived()

	var copyDerived *widget.Button
	copyDerived = widget.NewButtonWithIcon("", ftheme.ContentCopyIcon(), func() {
		addr := derivedAddressText(p.derivedLbl.Text)
		if addr == "" {
			return
		}
		p.win.Clipboard().SetContent(addr)
		feedback.CopyFlash(copyDerived)
	})
	copyDerived.Importance = widget.LowImportance

	derivedKey := widget.NewLabelWithStyle("Derived", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	derivedRow := container.NewBorder(nil, nil, derivedKey, copyDerived, p.derivedLbl)

	p.addressEnt = NewAddressEntry(true, "leave empty to use derived address")
	p.addressEnt.SetText(p.profile.Address)
	p.addressEnt.OnChanged = func(s string) {
		p.profile.Address = strings.TrimSpace(s)
		p.notify()
	}

	p.gasEntry = NewAddressEntry(true, "0x… gas coin object id")
	p.gasEntry.SetText(p.profile.GasCoinID)
	p.gasEntry.OnChanged = func(s string) {
		p.profile.GasCoinID = strings.TrimSpace(s)
		p.notify()
	}

	p.verifyBtn = widget.NewButton("Verify gas coin", p.verifyGasCoin)
	p.verifyLbl = widget.NewLabel("")
	p.verifyLbl.Wrapping = fyne.TextWrapWord

	signingForm := widget.NewForm(
		widget.NewFormItem("Alias", p.aliasSel),
		widget.NewFormItem("Private key", privBox),
	)

	chainForm := widget.NewForm(
		widget.NewFormItem("Address", widenIdentityField(p.addressEnt)),
		widget.NewFormItem("Gas coin", widenIdentityField(p.gasEntry)),
	)

	return container.NewVBox(
		SectionHeader("Signing", "Pick a saved alias or paste a private key."),
		signingForm,
		derivedRow,
		widget.NewSeparator(),
		SectionHeader("On-chain", "Address and gas coin used when sending transactions."),
		chainForm,
		ActionRow(p.verifyBtn),
		p.verifyLbl,
	)
}

func derivedAddressText(display string) string {
	s := strings.TrimSpace(display)
	switch s {
	case "", "(no key)", "invalid key":
		return ""
	default:
		return s
	}
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
	showVerifyErr := func(msg string) {
		feedback.ApplyLabel(p.verifyLbl, msg, feedback.Error)
		if p.feedback != nil {
			p.feedback.Show(feedback.Error, msg)
		}
	}
	if strings.TrimSpace(prof.GasCoinID) == "" {
		showVerifyErr("Gas coin id is empty.")
		return
	}
	if strings.TrimSpace(prof.PrivateKey) == "" {
		showVerifyErr("Private key is empty.")
		return
	}

	priv, err := wallet.ResolvePrivateKeyStrict(prof.PrivateKey)
	if err != nil {
		showVerifyErr(fmt.Sprintf("Private key: %v", err))
		return
	}
	addr, err := wallet.PrivateKeyToAddress(priv)
	if err != nil {
		showVerifyErr(fmt.Sprintf("Derive address: %v", err))
		return
	}

	p.verifyBtn.Disable()
	feedback.ApplyLabel(p.verifyLbl, "Verifying…", feedback.Info)
	go func() {
		defer fyne.Do(func() { p.verifyBtn.Enable() })
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		gasID, err := wallet.ResolveGasCoinId(ctx, prof.GasCoinID, addr, prof.RPCURL)
		fyne.Do(func() {
			if err != nil {
				showVerifyErr(err.Error())
				return
			}
			msg := fmt.Sprintf("%s belongs to %s", gasID, addr)
			feedback.ApplyLabel(p.verifyLbl, msg, feedback.Success)
		})
	}()
}

func (p *IdentityPanel) notify() {
	if p.onChanged != nil {
		p.onChanged()
	}
}

func widenIdentityField(field fyne.CanvasObject) fyne.CanvasObject {
	if field == nil {
		return field
	}
	return container.New(MinWidthLayout{MinW: identityFieldMinWidth}, field)
}
