package diagrams

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
	"sync"
)

//go:embed lifecycle.png
var lifecyclePNG []byte

var (
	lifecycleOnce sync.Once
	lifecycleImg  image.Image
	lifecycleErr  error
)

// LifecycleBitmap returns the embedded lifecycle diagram (PNG, full SVG fidelity).
func LifecycleBitmap() (image.Image, error) {
	lifecycleOnce.Do(func() {
		lifecycleImg, _, lifecycleErr = image.Decode(bytes.NewReader(lifecyclePNG))
	})
	return lifecycleImg, lifecycleErr
}
