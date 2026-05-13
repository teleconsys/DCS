package shell

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	ftheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/theme"
	"github.com/teleconsys/DCS/internal/rebased"
)

// AppBar is the persistent top chrome: brand, connection dot, wallet
// pill, identity button and output-drawer toggle. The actor switcher
// itself lives in the AppTabs container directly below the bar; the
// AppBar exposes Refresh(actor) so callers can re-render whenever the
// active tab changes.
type AppBar struct {
	win   fyne.Window
	app   *state.AppState
	dcs   *theme.Theme
	mu    sync.Mutex
	stop  chan struct{}

	brand     *widget.Label
	connDot   *canvas.Circle
	connLbl   *widget.Label
	walletLbl *widget.Label
	idBtn     *widget.Button
	outBtn    *widget.Button

	root fyne.CanvasObject
}

// NewAppBar constructs the bar and starts a background ticker that
// re-pings the RPC every 30 s. Call Stop on app shutdown to stop the
// ticker cleanly (not strictly required: the goroutine is bounded by
// the app lifetime).
func NewAppBar(win fyne.Window, app *state.AppState, t *theme.Theme,
	onIdentity, onToggleOutput func()) *AppBar {
	b := &AppBar{
		win:  win,
		app:  app,
		dcs:  t,
		stop: make(chan struct{}),
	}

	b.brand = widget.NewLabelWithStyle("DCS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	b.connDot = canvas.NewCircle(neutralDot)
	dotBox := container.NewGridWrap(fyne.NewSize(12, 12), b.connDot)
	b.connLbl = widget.NewLabel("connecting…")

	b.walletLbl = widget.NewLabel("")
	b.walletLbl.TextStyle = fyne.TextStyle{Monospace: true}

	b.idBtn = widget.NewButtonWithIcon("Identity", ftheme.AccountIcon(), onIdentity)
	b.idBtn.Importance = widget.LowImportance
	b.outBtn = widget.NewButtonWithIcon("Log", ftheme.DocumentIcon(), onToggleOutput)
	b.outBtn.Importance = widget.LowImportance

	left := container.NewHBox(b.brand)
	right := container.NewHBox(
		dotBox, b.connLbl,
		widget.NewSeparator(),
		b.walletLbl,
		b.idBtn, b.outBtn,
	)
	b.root = container.NewBorder(nil, widget.NewSeparator(), left, right, nil)

	go b.pingLoop()
	return b
}

// CanvasObject returns the bar's root for embedding in a Border layout.
func (b *AppBar) CanvasObject() fyne.CanvasObject { return b.root }

// Stop terminates the background ping ticker.
func (b *AppBar) Stop() {
	select {
	case <-b.stop:
	default:
		close(b.stop)
	}
}

// Refresh re-reads the registry view for the given actor and refreshes
// the wallet pill + identity button caption. Triggers an immediate
// ping so connection state lines up with the visible actor.
func (b *AppBar) Refresh(actor state.Actor) {
	p := b.app.Registry.Profile(actor)
	wallet := actorWallet(p)
	fyne.Do(func() {
		b.walletLbl.SetText(wallet)
		b.idBtn.SetText("Identity — " + actor.String())
	})
	go b.pingNow(p.RPCURL)
}

func (b *AppBar) pingLoop() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-b.stop:
			return
		case <-t.C:
			p := b.app.Registry.Profile(b.app.Registry.Current())
			b.pingNow(p.RPCURL)
		}
	}
}

func (b *AppBar) pingNow(rpcURL string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	rpcURL = strings.TrimSpace(rpcURL)
	if rpcURL == "" {
		fyne.Do(func() { b.setConn(stateBad, "no RPC") })
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	w, err := rebased.Dial(rpcURL)
	if err != nil {
		fyne.Do(func() { b.setConn(stateBad, "dial failed") })
		return
	}
	seq, err := w.Ping(ctx)
	if err != nil {
		fyne.Do(func() { b.setConn(stateBad, "unreachable") })
		return
	}
	fyne.Do(func() { b.setConn(stateOK, fmt.Sprintf("checkpoint %d", seq)) })
}

type connState int

const (
	stateNeutral connState = iota
	stateOK
	stateBad
)

var (
	neutralDot = color.NRGBA{R: 0x64, G: 0x74, B: 0x8B, A: 0xFF}
	okDot      = color.NRGBA{R: 0x22, G: 0xC5, B: 0x5E, A: 0xFF}
	badDot     = color.NRGBA{R: 0xEF, G: 0x44, B: 0x44, A: 0xFF}
)

func (b *AppBar) setConn(state connState, msg string) {
	switch state {
	case stateOK:
		b.connDot.FillColor = okDot
	case stateBad:
		b.connDot.FillColor = badDot
	default:
		b.connDot.FillColor = neutralDot
	}
	b.connDot.Refresh()
	b.connLbl.SetText(msg)
}

func actorWallet(p *state.ActorProfile) string {
	switch p.Actor {
	case state.ActorGC:
		ep := strings.TrimSpace(p.GCEndpoint)
		if ep == "" {
			ep = "(no endpoint)"
		}
		return "GC · " + ep
	default:
		addr := strings.TrimSpace(p.Address)
		if addr == "" {
			addr = "(no address)"
		} else if len(addr) > 14 {
			addr = addr[:8] + "…" + addr[len(addr)-4:]
		}
		return fmt.Sprintf("%s · %s", p.Actor, addr)
	}
}
