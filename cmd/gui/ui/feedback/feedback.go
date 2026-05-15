// Package feedback provides transient, non-modal UI feedback (banner bar,
// status flashes, copy confirmations).
package feedback

import (
	"image/color"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Kind classifies feedback severity for styling.
type Kind int

const (
	Info Kind = iota
	Success
	Error
)

const defaultDismiss = 5 * time.Second

var (
	bgInfo    = color.NRGBA{R: 0x1E, G: 0x3A, B: 0x5F, A: 0xFF}
	bgSuccess = color.NRGBA{R: 0x14, G: 0x3D, B: 0x2E, A: 0xFF}
	bgError   = color.NRGBA{R: 0x45, G: 0x1A, B: 0x1A, A: 0xFF}
	fgInfo    = color.NRGBA{R: 0xBF, G: 0xDB, B: 0xFE, A: 0xFF}
	fgSuccess = color.NRGBA{R: 0x86, G: 0xEF, B: 0xAC, A: 0xFF}
	fgError   = color.NRGBA{R: 0xFC, G: 0xA5, B: 0xA5, A: 0xFF}
)

// Host is a slim banner under the app bar. Hidden when empty.
type Host struct {
	mu      sync.Mutex
	bg      *canvas.Rectangle
	txt     *canvas.Text
	root    fyne.CanvasObject
	dismiss *time.Timer
}

// NewHost builds a feedback banner host (initially hidden).
func NewHost() *Host {
	h := &Host{
		bg:  canvas.NewRectangle(color.Transparent),
		txt: canvas.NewText("", fgInfo),
	}
	h.txt.TextSize = 14
	h.txt.Hide()
	h.bg.Hide()
	inner := container.NewStack(h.bg, container.NewPadded(h.txt))
	h.root = inner
	return h
}

// CanvasObject returns the banner widget for embedding in a Border layout.
func (h *Host) CanvasObject() fyne.CanvasObject { return h.root }

// Show displays a transient message and auto-clears after defaultDismiss.
func (h *Host) Show(kind Kind, message string) {
	if h == nil {
		return
	}
	msg := trimMessage(message)
	if msg == "" {
		h.Clear()
		return
	}
	fyne.Do(func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.dismiss != nil {
			h.dismiss.Stop()
		}
		bg, fg := colorsFor(kind)
		h.bg.FillColor = bg
		h.bg.Show()
		h.txt.Text = msg
		h.txt.Color = fg
		h.txt.Show()
		h.bg.Refresh()
		h.txt.Refresh()
		h.root.Refresh()
		h.dismiss = time.AfterFunc(defaultDismiss, func() {
			fyne.Do(h.Clear)
		})
	})
}

// Clear hides the banner.
func (h *Host) Clear() {
	if h == nil {
		return
	}
	fyne.Do(func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.dismiss != nil {
			h.dismiss.Stop()
			h.dismiss = nil
		}
		h.txt.Text = ""
		h.txt.Hide()
		h.bg.Hide()
		h.root.Refresh()
	})
}

func colorsFor(kind Kind) (bg, fg color.Color) {
	switch kind {
	case Success:
		return bgSuccess, fgSuccess
	case Error:
		return bgError, fgError
	default:
		return bgInfo, fgInfo
	}
}

func trimMessage(s string) string {
	const max = 280
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

// ApplyLabel sets label text and Fyne importance for inline status.
func ApplyLabel(lbl *widget.Label, message string, kind Kind) {
	if lbl == nil {
		return
	}
	lbl.SetText(message)
	lbl.Importance = labelImportance(kind)
	lbl.Show()
	lbl.Refresh()
}

func labelImportance(kind Kind) widget.Importance {
	switch kind {
	case Success:
		return widget.SuccessImportance
	case Error:
		return widget.DangerImportance
	default:
		return widget.MediumImportance
	}
}

// FlashStatus briefly sets lbl text and importance, then calls revert after dur.
func FlashStatus(lbl *widget.Label, message string, kind Kind, revert func(), dur time.Duration) {
	if lbl == nil {
		return
	}
	if dur <= 0 {
		dur = 2 * time.Second
	}
	fyne.Do(func() {
		ApplyLabel(lbl, message, kind)
	})
	time.AfterFunc(dur, func() {
		fyne.Do(func() {
			if revert != nil {
				revert()
			}
			lbl.Importance = widget.MediumImportance
			lbl.Refresh()
		})
	})
}

// CopyFlash briefly changes a copy button to show confirmation.
func CopyFlash(btn *widget.Button) {
	if btn == nil {
		return
	}
	orig := btn.Text
	fyne.Do(func() {
		btn.SetText("Copied")
		btn.Refresh()
	})
	time.AfterFunc(1500*time.Millisecond, func() {
		fyne.Do(func() {
			btn.SetText(orig)
			btn.Refresh()
		})
	})
}

// FirstLine returns the first non-empty line of a multi-line message.
func FirstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
