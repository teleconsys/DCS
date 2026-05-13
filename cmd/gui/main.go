// Command dcs-gui is a Fyne desktop front-end for the DCS CLI.
//
// The window is organized around the three DCS actors defined in
// testsheet.md § 2 (Ground Control, User, Provider) plus a side-by-side
// "Demo" mode that hosts both User and Provider workspaces at once for
// end-to-end demos. All on-chain operations call the same internal/*
// packages used by the CLI.
package main

import (
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/theme"
	"github.com/teleconsys/DCS/internal/config"
)

// ensureLocaleForFyne fixes WSL/minimal shells where LANG is "C" or
// "POSIX". Fyne parses LANG with golang.org/x/text/language and logs
// "tag is not well-formed" for those values.
func ensureLocaleForFyne() {
	const fallback = "en_US.UTF-8"
	lang := strings.TrimSpace(os.Getenv("LANG"))
	if lang == "" || lang == "C" || strings.EqualFold(lang, "POSIX") {
		_ = os.Setenv("LANG", fallback)
	}
	// LC_ALL=C overrides LANG and is equally invalid for Fyne.
	lc := strings.TrimSpace(os.Getenv("LC_ALL"))
	if lc == "C" || strings.EqualFold(lc, "POSIX") {
		_ = os.Unsetenv("LC_ALL")
		if strings.TrimSpace(os.Getenv("LANG")) == "" {
			_ = os.Setenv("LANG", fallback)
		}
	}
}

func main() {
	// Load .env before constructing the actor registry: defaults pulled
	// from environment variables would otherwise be empty.
	config.LoadEnv()
	ensureLocaleForFyne()

	a := app.NewWithID("io.teleconsys.dcs.gui")
	t := theme.New()
	a.Settings().SetTheme(t)

	w := a.NewWindow("DCS — Decentralised Content Security")
	w.Resize(fyne.NewSize(1400, 900))

	st := state.NewAppState()
	buildMainWindow(w, st, t)

	w.ShowAndRun()
}
