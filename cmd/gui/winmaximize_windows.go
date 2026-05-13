//go:build windows

package main

import (
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

// SW_SHOWMAXIMIZED: activate and maximize (fills work area; taskbar stays visible).
const swShowMaximized = 3

// tryMaximizeInitialWindow applies the same state as Windows “snap to top”
// maximize — not exclusive fullscreen.
func tryMaximizeInitialWindow(w fyne.Window) {
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	nw.RunNative(func(ctx any) {
		c, ok := ctx.(driver.WindowsWindowContext)
		if !ok || c.HWND == 0 {
			return
		}
		user32 := syscall.NewLazyDLL("user32.dll")
		showWindow := user32.NewProc("ShowWindow")
		_, _, _ = showWindow.Call(c.HWND, uintptr(swShowMaximized))
	})
}
