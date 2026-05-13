package gc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	ftheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/theme"
	"github.com/teleconsys/DCS/cmd/gui/ui"
	"github.com/teleconsys/DCS/cmd/gui/ui/components"
)

// Dashboard is the Ground Control whitelist: members table with delete per row,
// status and Add, inside a Whitelist section card.
func Dashboard(vc *ui.ViewContext) fyne.CanvasObject {
	return container.NewStack(buildWhitelistView(vc))
}

func buildWhitelistView(vc *ui.ViewContext) fyne.CanvasObject {
	table := components.NewDataTable(
		[]components.DataColumn{{Header: "Member address"}},
		"No addresses on the whitelist yet.",
	)

	var all []string
	statusLbl := widget.NewLabel("0 members")
	statusLbl.TextStyle = fyne.TextStyle{Italic: true}

	var openRemove func(member string)

	rebuild := func() {
		rows := make([]components.DataRow, 0, len(all))
		for _, m := range all {
			mCopy := m
			del := widget.NewButtonWithIcon("Delete", ftheme.DeleteIcon(), func() { openRemove(mCopy) })
			del.Importance = widget.DangerImportance
			rows = append(rows, components.DataRow{
				Cells:   []string{mCopy},
				Actions: []fyne.CanvasObject{del},
			})
		}
		fyne.Do(func() {
			table.SetRows(rows)
			statusLbl.SetText(fmt.Sprintf("%d member(s)", len(all)))
		})
	}

	refresh := func() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			members, err := service.WhitelistMembers(ctx, vc.Snapshot())
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("list members: %w", err), vc.Window)
				})
				return
			}
			all = members
			rebuild()
		}()
	}

	openAdd := func() {
		addr := ui.NewAddressEntry(false, "0x… address to add")
		form := widget.NewForm(widget.NewFormItem("Address", addr))
		d := dialog.NewCustomConfirm("Add member", "Add", "Cancel", form,
			func(ok bool) {
				if !ok {
					return
				}
				if err := addr.Validate(); err != nil {
					dialog.ShowError(err, vc.Window)
					return
				}
				member := strings.TrimSpace(addr.Text)
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
					defer cancel()
					if err := service.GCWhitelistAdd(ctx, vc.Snapshot(), member, vc.Output); err != nil {
						fyne.Do(func() { dialog.ShowError(err, vc.Window) })
						return
					}
					refresh()
				}()
			}, vc.Window)
		d.Resize(fyne.NewSize(480, 200))
		d.Show()
	}

	openRemove = func(member string) {
		dialog.ShowConfirm("Remove member",
			"Remove "+member+" from the whitelist?",
			func(ok bool) {
				if !ok {
					return
				}
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
					defer cancel()
					if err := service.GCWhitelistRemove(ctx, vc.Snapshot(), member, vc.Output); err != nil {
						fyne.Do(func() { dialog.ShowError(err, vc.Window) })
						return
					}
					refresh()
				}()
			}, vc.Window)
	}

	addBtn := widget.NewButtonWithIcon("Add", ftheme.ContentAddIcon(), openAdd)
	addBtn.Importance = widget.HighImportance

	header := container.NewBorder(nil, nil, statusLbl, addBtn, nil)
	top := container.NewVBox(header)
	body := container.NewBorder(top, nil, nil, nil, table.CanvasObject())
	refresh()

	return components.CardStretch(theme.AccentGC.Primary, "Whitelist", "", body)
}
