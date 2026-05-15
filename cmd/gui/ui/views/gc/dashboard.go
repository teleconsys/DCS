package gc

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	ftheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/theme"
	"github.com/teleconsys/DCS/cmd/gui/ui"
	"github.com/teleconsys/DCS/cmd/gui/ui/components"
	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
)

// Dashboard is the Admin workspace: whitelist, membership lookup, and CID list tools.
func Dashboard(vc *ui.ViewContext) fyne.CanvasObject {
	members := tabContentWithTopGap(buildWhitelistMembers(vc))
	lookup := tabContentWithTopGap(container.NewVScroll(ui.TopBound(WhitelistHasView(vc))))
	cidList := tabContentWithTopGap(container.NewVScroll(ui.TopBound(CIDIsInListView(vc))))
	tabs := container.NewAppTabs(
		container.NewTabItem("Members", members),
		container.NewTabItem("Lookup", lookup),
		container.NewTabItem("CID list", cidList),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return container.NewStack(components.CardStretch(theme.AccentGC.Primary, "Admin", "", tabs))
}

// tabContentWithTopGap keeps a fixed gap under the AppTabs bar, then gives
// everything below that line to inner (via Max) so the members table /
// lookup scroll actually receives the tab's remaining height. A trailing
// Spacer in a VBox would absorb that height and crop the list to one row.
func tabContentWithTopGap(inner fyne.CanvasObject) fyne.CanvasObject {
	gap := canvas.NewRectangle(color.Transparent)
	gap.SetMinSize(fyne.NewSize(0, 14))
	return container.NewBorder(gap, nil, nil, nil, container.NewMax(inner))
}

func buildWhitelistMembers(vc *ui.ViewContext) fyne.CanvasObject {
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
					statusLbl.SetText("List failed")
					if vc.Feedback != nil {
						vc.Feedback.Show(feedback.Error, err.Error())
					}
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
					if vc.Feedback != nil {
						vc.Feedback.Show(feedback.Error, err.Error())
					}
					return
				}
				member := strings.TrimSpace(addr.Text)
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
					defer cancel()
					out := vc.TeeOutput(nil)
					_, err := service.GCWhitelistAdd(ctx, vc.Snapshot(), member, out)
					if err != nil {
						fyne.Do(func() {
							if vc.Feedback != nil {
								vc.Feedback.Show(feedback.Error, err.Error())
							}
						})
						return
					}
					fyne.Do(func() {
						feedback.FlashStatus(statusLbl, "Member added", feedback.Success, nil, 2*time.Second)
						refresh()
					})
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
					out := vc.TeeOutput(nil)
					_, err := service.GCWhitelistRemove(ctx, vc.Snapshot(), member, out)
					if err != nil {
						fyne.Do(func() {
							if vc.Feedback != nil {
								vc.Feedback.Show(feedback.Error, err.Error())
							}
						})
						return
					}
					fyne.Do(func() {
						feedback.FlashStatus(statusLbl, "Member removed", feedback.Success, nil, 2*time.Second)
						refresh()
					})
				}()
			}, vc.Window)
	}

	addBtn := widget.NewButtonWithIcon("Add", ftheme.ContentAddIcon(), openAdd)
	addBtn.Importance = widget.HighImportance

	header := container.NewBorder(nil, nil, statusLbl, addBtn, nil)
	top := container.NewVBox(header)
	body := container.NewBorder(top, nil, nil, nil, table.CanvasObject())
	refresh()

	return body
}
