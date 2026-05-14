// Package components — pill_split.go
//
// Split layout and drag math are derived from fyne.io/fyne/v2/container (BSD).
// The stock divider paints a full-width strip (shadow / hover). Here the track
// stays transparent and only a rounded pill shows; the pill takes the theme
// primary accent on hover.

package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	ftheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var _ fyne.CanvasObject = (*PillSplit)(nil)

// PillSplit is like container.Split: two children with a draggable divider and
// Offset in [0, 1]. The divider hit-area is transparent; the visible grip is a
// rounded pill (foreground when idle, primary accent when hovered).
type PillSplit struct {
	widget.BaseWidget
	Offset      float64
	Horizontal  bool
	Leading     fyne.CanvasObject
	Trailing    fyne.CanvasObject
	offsetUpdated bool
}

// NewPillHSplit is like container.NewHSplit with the pill divider styling.
func NewPillHSplit(leading, trailing fyne.CanvasObject) *PillSplit {
	return newPillSplit(true, leading, trailing)
}

// NewPillVSplit is like container.NewVSplit with the pill divider styling.
func NewPillVSplit(top, bottom fyne.CanvasObject) *PillSplit {
	return newPillSplit(false, top, bottom)
}

func newPillSplit(horizontal bool, leading, trailing fyne.CanvasObject) *PillSplit {
	s := &PillSplit{
		Offset:     0.5,
		Horizontal: horizontal,
		Leading:    leading,
		Trailing:   trailing,
	}
	s.BaseWidget.ExtendBaseWidget(s)
	return s
}

// SetOffset sets the divider position (0 = leading min, 1 = trailing min).
func (s *PillSplit) SetOffset(offset float64) {
	if s.Offset == offset {
		return
	}
	s.Offset = offset
	s.offsetUpdated = true
	s.Refresh()
}

func (s *PillSplit) CreateRenderer() fyne.WidgetRenderer {
	s.BaseWidget.ExtendBaseWidget(s)
	d := newPillDivider(s)
	return &pillSplitRenderer{
		split:   s,
		divider: d,
		objects: []fyne.CanvasObject{s.Leading, d, s.Trailing},
	}
}

var _ fyne.WidgetRenderer = (*pillSplitRenderer)(nil)

type pillSplitRenderer struct {
	split   *PillSplit
	divider *pillDivider
	objects []fyne.CanvasObject
}

func (r *pillSplitRenderer) Destroy() {}

func (r *pillSplitRenderer) Layout(size fyne.Size) {
	var dividerPos, leadingPos, trailingPos fyne.Position
	var dividerSize, leadingSize, trailingSize fyne.Size

	dividerVisible := r.split.Leading.Visible() && r.split.Trailing.Visible()
	if !r.split.Leading.Visible() {
		trailingPos = fyne.NewPos(0, 0)
		trailingSize = size
	} else if !r.split.Trailing.Visible() {
		leadingPos = fyne.NewPos(0, 0)
		leadingSize = size
	} else if dividerVisible {
		if r.split.Horizontal {
			lw, tw := r.computeSplitLengths(size.Width, r.minLeadingWidth(), r.minTrailingWidth())
			leadingPos.X = 0
			leadingSize.Width = lw
			leadingSize.Height = size.Height
			dividerPos.X = lw
			dividerSize.Width = pillDividerThickness(r.divider)
			dividerSize.Height = size.Height
			trailingPos.X = lw + dividerSize.Width
			trailingSize.Width = tw
			trailingSize.Height = size.Height
		} else {
			lh, th := r.computeSplitLengths(size.Height, r.minLeadingHeight(), r.minTrailingHeight())
			leadingPos.Y = 0
			leadingSize.Width = size.Width
			leadingSize.Height = lh
			dividerPos.Y = lh
			dividerSize.Width = size.Width
			dividerSize.Height = pillDividerThickness(r.divider)
			trailingPos.Y = lh + dividerSize.Height
			trailingSize.Width = size.Width
			trailingSize.Height = th
		}
	}

	r.divider.Move(dividerPos)
	r.divider.Resize(dividerSize)
	r.divider.Hidden = !dividerVisible

	r.split.Leading.Move(leadingPos)
	r.split.Leading.Resize(leadingSize)
	r.split.Trailing.Move(trailingPos)
	r.split.Trailing.Resize(trailingSize)
	canvas.Refresh(r.divider)
}

func (r *pillSplitRenderer) MinSize() fyne.Size {
	s := fyne.NewSize(0, 0)
	dividerVisible := r.split.Leading.Visible() && r.split.Trailing.Visible()
	for i, o := range r.objects {
		if (i == 1 && !dividerVisible) || (i != 1 && !o.Visible()) {
			continue
		}
		min := o.MinSize()
		if r.split.Horizontal {
			s.Width += min.Width
			s.Height = fyne.Max(s.Height, min.Height)
		} else {
			s.Width = fyne.Max(s.Width, min.Width)
			s.Height += min.Height
		}
	}
	return s
}

func (r *pillSplitRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *pillSplitRenderer) Refresh() {
	if r.split.offsetUpdated {
		r.Layout(r.split.Size())
		r.split.offsetUpdated = false
		return
	}
	r.objects[0] = r.split.Leading
	r.objects[2] = r.split.Trailing
	r.Layout(r.split.Size())
	r.split.Leading.Refresh()
	r.divider.Refresh()
	r.split.Trailing.Refresh()
	canvas.Refresh(r.split)
}

func (r *pillSplitRenderer) computeSplitLengths(total, lMin, tMin float32) (float32, float32) {
	available := float64(total - pillDividerThickness(r.divider))
	if available <= 0 {
		return 0, 0
	}
	ld := float64(lMin)
	tr := float64(tMin)
	offset := r.split.Offset
	min := ld / available
	max := 1 - tr/available
	if min <= max {
		if offset < min {
			offset = min
		}
		if offset > max {
			offset = max
		}
	} else {
		offset = ld / (ld + tr)
	}
	ld = offset * available
	tr = available - ld
	return float32(ld), float32(tr)
}

func (r *pillSplitRenderer) minLeadingWidth() float32 {
	if r.split.Leading.Visible() {
		return r.split.Leading.MinSize().Width
	}
	return 0
}

func (r *pillSplitRenderer) minLeadingHeight() float32 {
	if r.split.Leading.Visible() {
		return r.split.Leading.MinSize().Height
	}
	return 0
}

func (r *pillSplitRenderer) minTrailingWidth() float32 {
	if r.split.Trailing.Visible() {
		return r.split.Trailing.MinSize().Width
	}
	return 0
}

func (r *pillSplitRenderer) minTrailingHeight() float32 {
	if r.split.Trailing.Visible() {
		return r.split.Trailing.MinSize().Height
	}
	return 0
}

var (
	_ fyne.CanvasObject  = (*pillDivider)(nil)
	_ fyne.Draggable     = (*pillDivider)(nil)
	_ desktop.Cursorable = (*pillDivider)(nil)
	_ desktop.Hoverable  = (*pillDivider)(nil)
)

type pillDivider struct {
	widget.BaseWidget
	split          *PillSplit
	hovered        bool
	startDragOff   *fyne.Position
	currentDragPos fyne.Position
}

func newPillDivider(split *PillSplit) *pillDivider {
	d := &pillDivider{split: split}
	d.ExtendBaseWidget(d)
	return d
}

func (d *pillDivider) CreateRenderer() fyne.WidgetRenderer {
	d.ExtendBaseWidget(d)
	track := canvas.NewRectangle(color.Transparent)
	track.StrokeWidth = 0
	pill := canvas.NewRectangle(color.Transparent)
	pill.StrokeWidth = 0
	return &pillDividerRenderer{
		divider: d,
		track:   track,
		pill:    pill,
		objects: []fyne.CanvasObject{track, pill},
	}
}

func (d *pillDivider) Cursor() desktop.Cursor {
	if d.split.Horizontal {
		return desktop.HResizeCursor
	}
	return desktop.VResizeCursor
}

func (d *pillDivider) DragEnd() { d.startDragOff = nil }

func (d *pillDivider) Dragged(e *fyne.DragEvent) {
	if d.startDragOff == nil {
		d.currentDragPos = d.Position().Add(e.Position)
		start := e.Position.Subtract(e.Dragged)
		d.startDragOff = &start
	} else {
		d.currentDragPos = d.currentDragPos.Add(e.Dragged)
	}
	x, y := d.currentDragPos.Components()
	var offset, leadingRatio, trailingRatio float64
	if d.split.Horizontal {
		widthFree := float64(d.split.Size().Width - pillDividerThickness(d))
		leadingRatio = float64(d.split.Leading.MinSize().Width) / widthFree
		trailingRatio = 1. - (float64(d.split.Trailing.MinSize().Width) / widthFree)
		offset = float64(x-d.startDragOff.X) / widthFree
	} else {
		heightFree := float64(d.split.Size().Height - pillDividerThickness(d))
		leadingRatio = float64(d.split.Leading.MinSize().Height) / heightFree
		trailingRatio = 1. - (float64(d.split.Trailing.MinSize().Height) / heightFree)
		offset = float64(y-d.startDragOff.Y) / heightFree
	}
	if offset < leadingRatio {
		offset = leadingRatio
	}
	if offset > trailingRatio {
		offset = trailingRatio
	}
	d.split.SetOffset(offset)
}

func (d *pillDivider) MouseIn(event *desktop.MouseEvent) {
	d.hovered = true
	d.Refresh()
}

func (d *pillDivider) MouseMoved(event *desktop.MouseEvent) {}

func (d *pillDivider) MouseOut() {
	d.hovered = false
	d.Refresh()
}

var _ fyne.WidgetRenderer = (*pillDividerRenderer)(nil)

type pillDividerRenderer struct {
	divider *pillDivider
	track   *canvas.Rectangle
	pill    *canvas.Rectangle
	objects []fyne.CanvasObject
}

func (r *pillDividerRenderer) Destroy() {}

func (r *pillDividerRenderer) Layout(size fyne.Size) {
	r.track.Resize(size)
	var x, y, w, h float32
	if r.divider.split.Horizontal {
		x = (pillDividerThickness(r.divider) - pillHandleThickness(r.divider)) / 2
		y = (size.Height - pillHandleLength(r.divider)) / 2
		w = pillHandleThickness(r.divider)
		h = pillHandleLength(r.divider)
	} else {
		x = (size.Width - pillHandleLength(r.divider)) / 2
		y = (pillDividerThickness(r.divider) - pillHandleThickness(r.divider)) / 2
		w = pillHandleLength(r.divider)
		h = pillHandleThickness(r.divider)
	}
	r.pill.Move(fyne.NewPos(x, y))
	r.pill.Resize(fyne.NewSize(w, h))
	radius := fyne.Min(w, h) / 2
	if radius > 0 {
		r.pill.CornerRadius = radius
	}
}

func (r *pillDividerRenderer) MinSize() fyne.Size {
	if r.divider.split.Horizontal {
		return fyne.NewSize(pillDividerThickness(r.divider), pillDividerLength(r.divider))
	}
	return fyne.NewSize(pillDividerLength(r.divider), pillDividerThickness(r.divider))
}

func (r *pillDividerRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *pillDividerRenderer) Refresh() {
	th := r.divider.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	r.track.FillColor = color.Transparent
	r.track.Refresh()
	if r.divider.hovered {
		r.pill.FillColor = th.Color(ftheme.ColorNamePrimary, v)
	} else {
		r.pill.FillColor = th.Color(ftheme.ColorNameForeground, v)
	}
	r.pill.Refresh()
	r.Layout(r.divider.Size())
}

func pillDividerTheme(d *pillDivider) fyne.Theme {
	if d == nil {
		return ftheme.Current()
	}
	return d.Theme()
}

func pillDividerThickness(d *pillDivider) float32 {
	th := pillDividerTheme(d)
	return th.Size(ftheme.SizeNamePadding) * 2
}

func pillDividerLength(d *pillDivider) float32 {
	th := pillDividerTheme(d)
	return th.Size(ftheme.SizeNamePadding) * 6
}

func pillHandleThickness(d *pillDivider) float32 {
	th := pillDividerTheme(d)
	return th.Size(ftheme.SizeNamePadding) / 2
}

func pillHandleLength(d *pillDivider) float32 {
	th := pillDividerTheme(d)
	return th.Size(ftheme.SizeNamePadding) * 4
}
