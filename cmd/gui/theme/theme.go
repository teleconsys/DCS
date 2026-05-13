// Package theme defines the custom DCS Fyne theme. The theme keeps a
// mutable accent color so a single instance can re-skin across actor
// changes (GC = teal, User = green, Provider = violet).
package theme

import (
	"image/color"
	"sync/atomic"

	"fyne.io/fyne/v2"
	ftheme "fyne.io/fyne/v2/theme"
)

// Accent bundles the primary + hover color used to brand an actor.
type Accent struct {
	Primary color.Color
	Hover   color.Color
}

var (
	// AccentGC: teal — administrative, "control room" feel.
	AccentGC = Accent{
		Primary: color.NRGBA{R: 0x14, G: 0xB8, B: 0xA6, A: 0xFF},
		Hover:   color.NRGBA{R: 0x0F, G: 0x9A, B: 0x8A, A: 0xFF},
	}
	// AccentUser: green — content owners.
	AccentUser = Accent{
		Primary: color.NRGBA{R: 0x22, G: 0xC5, B: 0x5E, A: 0xFF},
		Hover:   color.NRGBA{R: 0x16, G: 0xA3, B: 0x4A, A: 0xFF},
	}
	// AccentProvider: violet — storage providers.
	AccentProvider = Accent{
		Primary: color.NRGBA{R: 0x8B, G: 0x5C, B: 0xF6, A: 0xFF},
		Hover:   color.NRGBA{R: 0x7C, G: 0x3A, B: 0xED, A: 0xFF},
	}
	// AccentDefault: blue — fallback / demo mode.
	AccentDefault = Accent{
		Primary: color.NRGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF},
		Hover:   color.NRGBA{R: 0x25, G: 0x63, B: 0xEB, A: 0xFF},
	}

	colorSuccess = color.NRGBA{R: 0x22, G: 0xC5, B: 0x5E, A: 0xFF}
	colorWarning = color.NRGBA{R: 0xF5, G: 0x9E, B: 0x0B, A: 0xFF}
	colorError   = color.NRGBA{R: 0xEF, G: 0x44, B: 0x44, A: 0xFF}

	// Surface palette — neutral dark canvas with raised surface for cards.
	colorBackground      = color.NRGBA{R: 0x0F, G: 0x17, B: 0x21, A: 0xFF} // slate-900
	colorSurface         = color.NRGBA{R: 0x1E, G: 0x29, B: 0x3B, A: 0xFF} // slate-800
	colorSurfaceRaised   = color.NRGBA{R: 0x17, G: 0x20, B: 0x32, A: 0xFF}
	colorOverlay         = color.NRGBA{R: 0x0B, G: 0x11, B: 0x1A, A: 0xCC}
	colorForeground      = color.NRGBA{R: 0xE2, G: 0xE8, B: 0xF0, A: 0xFF} // slate-200
	colorMutedForeground = color.NRGBA{R: 0x94, G: 0xA3, B: 0xB8, A: 0xFF} // slate-400
	colorDisabled        = color.NRGBA{R: 0x64, G: 0x74, B: 0x8B, A: 0xFF} // slate-500
	colorSeparator       = color.NRGBA{R: 0x33, G: 0x44, B: 0x56, A: 0xFF}
	colorScrollBar       = color.NRGBA{R: 0x47, G: 0x5A, B: 0x6E, A: 0xCC}
	colorShadow          = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x88}
)

// Theme is the DCS Fyne theme. The accent is held as an atomic pointer
// so SetAccent is safe to call from any goroutine; callers must trigger
// a settings refresh (see App.Settings().SetTheme) for live widgets to
// pick the new color up.
type Theme struct {
	accent atomic.Pointer[Accent]
}

// New returns a theme initialized with the default (blue) accent.
func New() *Theme {
	t := &Theme{}
	a := AccentDefault
	t.accent.Store(&a)
	return t
}

// SetAccent swaps the per-actor accent.
func (t *Theme) SetAccent(a Accent) { t.accent.Store(&a) }

// CurrentAccent returns a copy of the live accent.
func (t *Theme) CurrentAccent() Accent { return *t.accent.Load() }

// AccentForRole picks the right palette for a role tag ("gc", "user",
// "provider") — anything else returns AccentDefault.
func AccentForRole(role string) Accent {
	switch role {
	case "gc":
		return AccentGC
	case "user":
		return AccentUser
	case "provider":
		return AccentProvider
	}
	return AccentDefault
}

// Color implements fyne.Theme.
func (t *Theme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case ftheme.ColorNamePrimary, ftheme.ColorNameFocus, ftheme.ColorNameSelection:
		return t.accent.Load().Primary
	case ftheme.ColorNameHover:
		return t.accent.Load().Hover
	case ftheme.ColorNameSuccess:
		return colorSuccess
	case ftheme.ColorNameWarning:
		return colorWarning
	case ftheme.ColorNameError:
		return colorError
	case ftheme.ColorNameBackground:
		return colorBackground
	case ftheme.ColorNameForeground:
		return colorForeground
	case ftheme.ColorNameDisabled:
		return colorDisabled
	case ftheme.ColorNameDisabledButton:
		return colorSurfaceRaised
	case ftheme.ColorNameInputBackground:
		return colorSurface
	case ftheme.ColorNameButton:
		return colorSurface
	case ftheme.ColorNameMenuBackground:
		return colorSurfaceRaised
	case ftheme.ColorNameOverlayBackground:
		return colorOverlay
	case ftheme.ColorNameSeparator:
		return colorSeparator
	case ftheme.ColorNameShadow:
		return colorShadow
	case ftheme.ColorNamePlaceHolder:
		return colorMutedForeground
	case ftheme.ColorNameScrollBar:
		return colorScrollBar
	case ftheme.ColorNameHeaderBackground:
		return colorSurfaceRaised
	}
	return ftheme.DefaultTheme().Color(name, variant)
}

// Font implements fyne.Theme.
func (t *Theme) Font(style fyne.TextStyle) fyne.Resource {
	return ftheme.DefaultTheme().Font(style)
}

// Icon implements fyne.Theme.
func (t *Theme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return ftheme.DefaultTheme().Icon(name)
}

// Size implements fyne.Theme.
func (t *Theme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case ftheme.SizeNamePadding:
		return 6
	case ftheme.SizeNameInlineIcon:
		return 18
	}
	return ftheme.DefaultTheme().Size(name)
}
