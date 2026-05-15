package user

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"

	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// OpenIPFSModal shows Upload, Check CID, and Check pins in a single dialog.
func OpenIPFSModal(vc *ui.ViewContext) {
	makeTab := func(label string, builder ui.View) *container.TabItem {
		body := container.NewVScroll(ui.TopBound(builder(vc)))
		return container.NewTabItem(label, body)
	}
	tabs := container.NewAppTabs(
		makeTab("Upload", IPFSLoadView),
		makeTab("Check CID", IPFSCheckCIDView),
		makeTab("Check pins", IPFSCheckPinsView),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	d := dialog.NewCustom("IPFS", "Close", tabs, vc.Window)
	d.Resize(fyne.NewSize(720, 520))
	d.Show()
}
