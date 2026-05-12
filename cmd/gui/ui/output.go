// Package ui contains the Fyne widgets, identity panels and per-action
// views that make up the DCS GUI. The package depends on `state` and
// `service` but never the other way around.
package ui

import (
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// OutputView is a scrollable, read-only log surface that doubles as an
// io.Writer. Writes from background goroutines are safe because every
// UI mutation is dispatched onto the main thread via fyne.Do.
type OutputView struct {
	entry     *widget.Entry
	scroll    *container.Scroll
	clearBtn  *widget.Button
	titleLbl  *widget.Label
	container *fyne.Container

	mu  sync.Mutex
	buf strings.Builder
}

// NewOutputView returns an empty log view labelled `title`.
func NewOutputView(title string) *OutputView {
	entry := widget.NewMultiLineEntry()
	entry.Wrapping = fyne.TextWrapBreak
	entry.SetMinRowsVisible(8)
	entry.Disable() // read-only but still selectable / copyable

	scroll := container.NewScroll(entry)
	scroll.SetMinSize(fyne.NewSize(320, 180))

	out := &OutputView{entry: entry, scroll: scroll, titleLbl: widget.NewLabel(title)}
	out.clearBtn = widget.NewButton("Clear", func() { out.Clear() })

	header := container.NewBorder(nil, nil, out.titleLbl, out.clearBtn)
	out.container = container.NewBorder(header, nil, nil, nil, scroll)
	return out
}

// CanvasObject lets the view be embedded in other containers.
func (o *OutputView) CanvasObject() fyne.CanvasObject { return o.container }

// Write implements io.Writer. Safe to call from any goroutine.
func (o *OutputView) Write(p []byte) (int, error) {
	o.mu.Lock()
	o.buf.Write(p)
	text := o.buf.String()
	o.mu.Unlock()

	fyne.Do(func() {
		o.entry.SetText(text)
		o.entry.CursorRow = len(strings.Split(text, "\n")) - 1
		o.entry.Refresh()
		o.scroll.ScrollToBottom()
	})
	return len(p), nil
}

// Clear empties the log on the UI thread.
func (o *OutputView) Clear() {
	o.mu.Lock()
	o.buf.Reset()
	o.mu.Unlock()
	fyne.Do(func() {
		o.entry.SetText("")
		o.entry.Refresh()
	})
}

// Println appends a line followed by a newline.
func (o *OutputView) Println(s string) {
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	_, _ = o.Write([]byte(s))
}
