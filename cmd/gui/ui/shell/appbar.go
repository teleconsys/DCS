package shell

import (
	"context"
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

// Connection status labels (shown next to the indicator dot in the app bar).
const (
	connLabelConnecting  = "Connecting…"
	connLabelConnected   = "Connected"
	connLabelNoRPC       = "No RPC"
	connLabelDialFailed  = "Dial failed"
	connLabelUnreachable = "Unreachable"
)

// AppBar is the persistent top chrome: brand, connection dot, settings and
// identity buttons. The actor switcher lives in the AppTabs container directly
// below the bar; AppBar exposes Refresh(actor) so callers can re-render
// whenever the active tab changes.
type AppBar struct {
	win  fyne.Window
	app  *state.AppState
	dcs  *theme.Theme
	mu   sync.Mutex
	stop chan struct{}

	brand   *widget.Label
	connDot *canvas.Circle
	connLbl *widget.Label
	idBtn   *widget.Button

	root fyne.CanvasObject
}

// NewAppBar constructs the bar and starts a background ticker that
// re-pings the RPC every 30 s. Call Stop on app shutdown to stop the
// ticker cleanly (not strictly required: the goroutine is bounded by
// the app lifetime).
func NewAppBar(win fyne.Window, app *state.AppState, t *theme.Theme, onIdentity func()) *AppBar {
	b := &AppBar{
		win:  win,
		app:  app,
		dcs:  t,
		stop: make(chan struct{}),
	}

	b.brand = widget.NewLabelWithStyle("DCS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	b.connDot = canvas.NewCircle(neutralDot)
	dotCell := container.NewGridWrap(fyne.NewSize(10, 10), b.connDot)
	dotRow := container.NewCenter(dotCell)
	b.connLbl = widget.NewLabel(connLabelConnecting)

	connGroup := container.NewBorder(nil, nil, dotRow, nil, b.connLbl)

	b.idBtn = widget.NewButtonWithIcon("", ftheme.AccountIcon(), onIdentity)
	b.idBtn.Importance = widget.LowImportance

	settingsBtn := widget.NewButtonWithIcon("", ftheme.SettingsIcon(), func() {
		OpenAppSettings(win, app, func() {
			b.Refresh(app.Registry.Current())
		})
	})
	settingsBtn.Importance = widget.LowImportance

	left := container.NewHBox(b.brand)
	right := container.NewHBox(connGroup, widget.NewSeparator(), settingsBtn, b.idBtn)
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

// Refresh triggers an immediate ping so connection state lines up with
// the visible actor.
func (b *AppBar) Refresh(actor state.Actor) {
	p := b.app.Registry.Profile(actor)
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
		b.setConn(stateBad, connLabelNoRPC)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	w, err := rebased.Dial(rpcURL)
	if err != nil {
		b.setConn(stateBad, connLabelDialFailed)
		return
	}
	if _, err := w.Ping(ctx); err != nil {
		b.setConn(stateBad, connLabelUnreachable)
		return
	}
	b.setConn(stateOK, connLabelConnected)
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
	fyne.Do(func() {
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
	})
}
