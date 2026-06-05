package components

import (
	"image"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const (
	zoomStep    = 1.12
	zoomMin     = 0.15
	zoomMax     = 12
	zoomMinView = 320
)

// ZoomImage is a pannable, zoomable image viewer (scroll wheel to zoom, drag to pan).
type ZoomImage struct {
	widget.BaseWidget

	source image.Image
	img    *canvas.Image

	zoom       float32
	offset     fyne.Position
	onZoom     func(zoom float32)
	dragActive bool
}

// NewZoomImage wraps a bitmap in a zoom/pan viewer.
func NewZoomImage(src image.Image) *ZoomImage {
	z := &ZoomImage{
		source: src,
		zoom:   1,
	}
	z.img = canvas.NewImageFromImage(src)
	z.img.FillMode = canvas.ImageFillStretch
	z.ExtendBaseWidget(z)
	return z
}

// OnZoomChanged is called when zoom level changes (1.0 = fit to window).
func (z *ZoomImage) OnZoomChanged(fn func(zoom float32)) {
	z.onZoom = fn
}

// ZoomIn increases magnification around the viewport center.
func (z *ZoomImage) ZoomIn() { z.applyZoom(1) }

// ZoomOut decreases magnification around the viewport center.
func (z *ZoomImage) ZoomOut() { z.applyZoom(-1) }

// ResetZoom fits the image in the viewport, centered.
func (z *ZoomImage) ResetZoom() {
	z.zoom = 1
	z.offset = fyne.NewPos(0, 0)
	z.notifyZoom()
	z.Refresh()
}

// ZoomLevel returns the current zoom factor (1 = fit).
func (z *ZoomImage) ZoomLevel() float32 { return z.zoom }

func (z *ZoomImage) Resize(size fyne.Size) {
	z.BaseWidget.Resize(size)
	z.clampOffset()
}

func (z *ZoomImage) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(diagramViewerBg)
	return &zoomImageRenderer{
		z:       z,
		bg:      bg,
		objects: []fyne.CanvasObject{bg, z.img},
	}
}

func (z *ZoomImage) Scrolled(ev *fyne.ScrollEvent) {
	dy := ev.Scrolled.DY
	if dy == 0 {
		dy = ev.Scrolled.DX
	}
	if dy == 0 {
		return
	}
	// Reversed: scroll down/forward zooms in, scroll up/back zooms out.
	dir := float32(-1)
	if dy > 0 {
		dir = 1
	}
	z.applyZoom(dir)
}

func (z *ZoomImage) Dragged(e *fyne.DragEvent) {
	if !z.dragActive {
		z.dragActive = true
	}
	z.offset = z.offset.Subtract(e.Dragged)
	z.clampOffset()
	z.Refresh()
}

func (z *ZoomImage) DragEnd() { z.dragActive = false }

func (z *ZoomImage) MouseIn(ev *desktop.MouseEvent)   {}
func (z *ZoomImage) MouseMoved(ev *desktop.MouseEvent) {}
func (z *ZoomImage) MouseOut()                        {}

func (z *ZoomImage) Cursor() desktop.Cursor {
	if z.dragActive {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (z *ZoomImage) aspect() float32 {
	b := z.source.Bounds()
	if b.Dy() == 0 {
		return 1
	}
	return float32(b.Dx()) / float32(b.Dy())
}

func (z *ZoomImage) fitSize(viewport fyne.Size) fyne.Size {
	if viewport.Width <= 0 || viewport.Height <= 0 {
		return fyne.NewSize(zoomMinView, zoomMinView/z.aspect())
	}
	a := z.aspect()
	viewAspect := viewport.Width / viewport.Height
	if viewAspect > a {
		h := viewport.Height
		return fyne.NewSize(h*a, h)
	}
	w := viewport.Width
	return fyne.NewSize(w, w/a)
}

func (z *ZoomImage) displaySize(viewport fyne.Size) fyne.Size {
	f := z.fitSize(viewport)
	return fyne.NewSize(f.Width*z.zoom, f.Height*z.zoom)
}

func (z *ZoomImage) viewportCenter() fyne.Position {
	vp := z.Size()
	return fyne.NewPos(vp.Width/2, vp.Height/2)
}

func (z *ZoomImage) centerPad(viewport, content fyne.Size) fyne.Position {
	return fyne.NewPos(
		fyne.Max(0, (viewport.Width-content.Width)/2),
		fyne.Max(0, (viewport.Height-content.Height)/2),
	)
}

func (z *ZoomImage) applyZoom(direction float32) {
	vp := z.Size()
	old := z.displaySize(vp)
	center := z.viewportCenter()
	oldPad := z.centerPad(vp, old)

	if direction > 0 {
		z.zoom *= zoomStep
	} else {
		z.zoom /= zoomStep
	}
	if z.zoom < zoomMin {
		z.zoom = zoomMin
	}
	if z.zoom > zoomMax {
		z.zoom = zoomMax
	}

	newSize := z.displaySize(vp)
	newPad := z.centerPad(vp, newSize)
	if old.Width > 0 && old.Height > 0 {
		imgX := z.offset.X + center.X - oldPad.X
		imgY := z.offset.Y + center.Y - oldPad.Y
		ratioX := imgX / old.Width
		ratioY := imgY / old.Height
		z.offset.X = ratioX*newSize.Width - center.X + newPad.X
		z.offset.Y = ratioY*newSize.Height - center.Y + newPad.Y
	}
	z.clampOffset()
	z.notifyZoom()
	z.Refresh()
}

func (z *ZoomImage) clampOffset() {
	vp := z.Size()
	disp := z.displaySize(vp)
	maxX := fyne.Max(0, disp.Width-vp.Width)
	maxY := fyne.Max(0, disp.Height-vp.Height)
	z.offset.X = clampF32(z.offset.X, 0, maxX)
	z.offset.Y = clampF32(z.offset.Y, 0, maxY)
}

func (z *ZoomImage) notifyZoom() {
	if z.onZoom != nil {
		z.onZoom(z.zoom)
	}
}

func (z *ZoomImage) imagePosition(viewport fyne.Size) fyne.Position {
	disp := z.displaySize(viewport)
	pad := z.centerPad(viewport, disp)
	return fyne.NewPos(pad.X-z.offset.X, pad.Y-z.offset.Y)
}

func clampF32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

var (
	_ fyne.CanvasObject  = (*ZoomImage)(nil)
	_ fyne.Scrollable    = (*ZoomImage)(nil)
	_ fyne.Draggable     = (*ZoomImage)(nil)
	_ desktop.Hoverable  = (*ZoomImage)(nil)
	_ desktop.Cursorable = (*ZoomImage)(nil)
)

type zoomImageRenderer struct {
	z       *ZoomImage
	bg      *canvas.Rectangle
	objects []fyne.CanvasObject
}

func (r *zoomImageRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))

	disp := r.z.displaySize(size)
	r.z.img.Resize(disp)
	pos := r.z.imagePosition(size)
	r.z.img.Move(pos)
}

func (r *zoomImageRenderer) MinSize() fyne.Size {
	return fyne.NewSize(zoomMinView, 240)
}

func (r *zoomImageRenderer) Refresh() {
	r.bg.FillColor = diagramViewerBg
	r.Layout(r.z.Size())
	canvas.Refresh(r.z.img)
}

func (r *zoomImageRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *zoomImageRenderer) Destroy() {}

// Matches lifecycle diagram canvas background.
var diagramViewerBg = color.NRGBA{R: 0x0F, G: 0x0F, B: 0x1A, A: 0xFF}
