// Package components hosts reusable presentational widgets (cards,
// badges, key-value rows, empty states, simple data tables) used to
// compose the per-actor dashboards.
package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	apptheme "github.com/teleconsys/DCS/cmd/gui/theme"
)

func cardBackdrop(stroke color.Color, content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(apptheme.CardInteriorFill())
	bg.StrokeColor = stroke
	bg.StrokeWidth = 2
	bg.CornerRadius = 8
	return container.NewStack(bg, content)
}

func cardBackdropWithAccent(accent color.Color, content fyne.CanvasObject) fyne.CanvasObject {
	return cardBackdrop(accent, content)
}

// Card renders a visual card: rounded panel with an accent-colored border,
// bold title (optional subtitle), separator, body content, then a row of
// action buttons. Pass nil for body or empty actions to omit either.
func Card(accent color.Color, title, subtitle string, body fyne.CanvasObject, actions ...fyne.CanvasObject) fyne.CanvasObject {
	titleLbl := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	var head fyne.CanvasObject = titleLbl
	if subtitle != "" {
		sub := widget.NewLabel(subtitle)
		sub.TextStyle = fyne.TextStyle{Italic: true}
		head = container.NewVBox(titleLbl, sub)
	}

	parts := []fyne.CanvasObject{head}
	if body != nil {
		parts = append(parts, widget.NewSeparator(), body)
	}
	if len(actions) > 0 {
		parts = append(parts, widget.NewSeparator(), container.NewHBox(actions...))
	}

	inner := container.NewPadded(container.NewVBox(parts...))
	return cardBackdropWithAccent(accent, inner)
}

// CardSeparatorStroke is like Card but uses the same stroke tone as UI
// separators instead of the actor accent (for tiles where accent + badge is noisy).
func CardSeparatorStroke(title, subtitle string, body fyne.CanvasObject, actions ...fyne.CanvasObject) fyne.CanvasObject {
	var footer fyne.CanvasObject
	if len(actions) > 0 {
		footer = container.NewHBox(actions...)
	}
	if title == "" && subtitle == "" {
		return TileCard(body, footer)
	}

	titleLbl := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	var head fyne.CanvasObject = titleLbl
	if subtitle != "" {
		sub := widget.NewLabel(subtitle)
		sub.TextStyle = fyne.TextStyle{Italic: true}
		head = container.NewVBox(titleLbl, sub)
	}

	parts := []fyne.CanvasObject{head}
	if body != nil {
		parts = append(parts, widget.NewSeparator(), body)
	}
	if footer != nil {
		parts = append(parts, widget.NewSeparator(), footer)
	}

	inner := container.NewPadded(container.NewVBox(parts...))
	return cardBackdrop(apptheme.SeparatorStroke(), inner)
}

// CardStretch is like Card but lays the body inside a Border layout so
// it expands to fill all available vertical space. Use it for a card
// that hosts a table or grid.
//
// If both title and subtitle are empty, no heading row is shown — only the
// body (and optional footer actions), useful for a titleless macro panel.
func CardStretch(accent color.Color, title, subtitle string, body fyne.CanvasObject, actions ...fyne.CanvasObject) fyne.CanvasObject {
	var footer fyne.CanvasObject
	if len(actions) > 0 {
		footer = container.NewVBox(widget.NewSeparator(), container.NewHBox(actions...))
	}
	center := body
	if body != nil {
		center = container.NewStack(body)
	}

	if title == "" && subtitle == "" {
		inner := container.NewBorder(nil, footer, nil, nil, center)
		return cardBackdropWithAccent(accent, container.NewPadded(inner))
	}

	titleLbl := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	var head fyne.CanvasObject = titleLbl
	if subtitle != "" {
		sub := widget.NewLabel(subtitle)
		sub.TextStyle = fyne.TextStyle{Italic: true}
		head = container.NewVBox(titleLbl, sub)
	}
	header := container.NewVBox(head, widget.NewSeparator())
	inner := container.NewBorder(header, footer, nil, nil, center)
	return cardBackdropWithAccent(accent, container.NewPadded(inner))
}

// SectionTitle returns a bold heading suitable for a dashboard section.
func SectionTitle(text string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

// AccentSeparator returns a 3px-tall colored bar useful as a visual
// section divider above grouped content.
func AccentSeparator(c color.Color) fyne.CanvasObject {
	bar := canvas.NewRectangle(c)
	bar.SetMinSize(fyne.NewSize(0, 3))
	return bar
}
