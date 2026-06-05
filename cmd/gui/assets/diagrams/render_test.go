package diagrams

import (
	"testing"
)

func TestLifecyclePNGLoads(t *testing.T) {
	bmp, err := LifecycleBitmap()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	b := bmp.Bounds()
	if b.Dx() < 2000 || b.Dy() < 1100 {
		t.Fatalf("unexpected size: %dx%d", b.Dx(), b.Dy())
	}

	nonBg := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := bmp.At(x, y).RGBA()
			if uint8(r>>8) != 0x0F || uint8(g>>8) != 0x0F || uint8(bl>>8) != 0x1A {
				nonBg++
			}
		}
	}
	if nonBg < 100_000 {
		t.Fatalf("render looks empty: only %d non-background pixels", nonBg)
	}
}
