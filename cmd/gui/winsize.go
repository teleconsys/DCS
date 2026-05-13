package main

import "fyne.io/fyne/v2"

// initialWindowSize is a fixed logical (DPI-aware) size. We intentionally
// do not read Win32 GetSystemMetrics here: those values are in physical
// pixels and do not match Fyne's coordinate space on scaled displays, which
// produced windows larger than the visible work area (borders off-screen).
// A modest default plus CenterOnScreen matches typical desktop app startup.
func initialWindowSize() fyne.Size {
	return fyne.NewSize(1024, 640)
}
