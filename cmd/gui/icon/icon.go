package icon

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed Icon.png
var png []byte

// Resource is the application window / taskbar icon.
func Resource() fyne.Resource {
	return fyne.NewStaticResource("Icon.png", png)
}
