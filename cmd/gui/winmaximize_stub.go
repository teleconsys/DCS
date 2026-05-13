//go:build !windows

package main

import "fyne.io/fyne/v2"

// tryMaximizeInitialWindow is a no-op on non-Windows builds; Fyne does not
// expose a portable “maximize” API yet. Default size + center still apply.
func tryMaximizeInitialWindow(_ fyne.Window) {}
