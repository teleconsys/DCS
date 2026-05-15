package nativefile

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
)

var errCancelled = errors.New("cancelled")

var errUseFyne = errors.New("use fyne file dialog")

// PickOpenFile opens the OS file picker when the platform supports it; otherwise
// it falls back to Fyne's built-in file dialog. The chosen path is written to entry.
func PickOpenFile(win fyne.Window, entry *widget.Entry, title string, fb ...*feedback.Host) {
	var host *feedback.Host
	if len(fb) > 0 {
		host = fb[0]
	}
	if title == "" {
		title = "Select file"
	}
	path, err := pickNativePath(title)
	switch {
	case err == nil && path != "":
		fyne.Do(func() { entry.SetText(path) })
	case errors.Is(err, errUseFyne):
		showFyneFileOpen(win, entry, title, host)
	case errors.Is(err, errCancelled):
		return
	default:
		if err != nil {
			fyne.Do(func() {
				if host != nil {
					host.Show(feedback.Error, err.Error())
				}
			})
		}
	}
}

func showFyneFileOpen(win fyne.Window, entry *widget.Entry, title string, fb *feedback.Host) {
	dlg := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			if fb != nil {
				fb.Show(feedback.Error, err.Error())
			}
			return
		}
		if r == nil {
			return
		}
		defer r.Close()
		u := r.URI()
		path := u.Path()
		if path == "" {
			path = u.String()
		}
		entry.SetText(path)
	}, win)
	dlg.SetTitleText(title)
	dlg.Show()
}
