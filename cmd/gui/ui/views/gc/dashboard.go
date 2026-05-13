package gc

import (
	"context"
	"fmt"
	"io"
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
)

// Dashboard is the redesigned Ground Control workspace: a connection
// card on top (GC API + RPC reachability) and a whitelist console
// below with inline Add/Remove actions per row.
func Dashboard(vc *ui.ViewContext) fyne.CanvasObject {
	conn := buildConnectionCard(vc)
	wl := buildWhitelistCard(vc)
	return container.NewBorder(conn, nil, nil, nil, wl)
}

// ---- connection card -------------------------------------------------------

func buildConnectionCard(vc *ui.ViewContext) fyne.CanvasObject {
	gcDot := canvas.NewCircle(components.BadgeNeutral.Color())
	gcDotBox := container.NewGridWrap(fyne.NewSize(12, 12), gcDot)
	gcLbl := widget.NewLabel("unknown")

	rpcDot := canvas.NewCircle(components.BadgeNeutral.Color())
	rpcDotBox := container.NewGridWrap(fyne.NewSize(12, 12), rpcDot)
	rpcLbl := widget.NewLabel("unknown")

	tokenLabel := "not set"
	if strings.TrimSpace(vc.Profile.GCToken) != "" {
		tokenLabel = "set (hidden)"
	}

	rows := container.NewVBox(
		components.KeyValueRow(vc.Window, "GC endpoint", vc.Profile.GCEndpoint, true),
		components.KeyValueRow(vc.Window, "GC token", tokenLabel, false),
		components.KeyValueRow(vc.Window, "RPC URL", vc.Profile.RPCURL, true),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabelWithStyle("GC API:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), gcDotBox, gcLbl),
		container.NewHBox(widget.NewLabelWithStyle("RPC:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), rpcDotBox, rpcLbl),
	)

	pingAll := func() {
		go func() {
			snap := vc.Snapshot()
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			defer cancel()
			// GC health probe.
			gcErr := service.GCHealth(ctx, snap)
			fyne.Do(func() {
				if gcErr != nil {
					gcDot.FillColor = components.BadgeErr.Color()
					gcLbl.SetText(gcErr.Error())
				} else {
					gcDot.FillColor = components.BadgeOK.Color()
					gcLbl.SetText("reachable")
				}
				gcDot.Refresh()
			})

			// RPC ping (writes to io.Discard so we don't spam the log).
			res, pingErr := service.Ping(ctx, snap.RPCURL, 5*time.Second, io.Discard)
			fyne.Do(func() {
				if pingErr != nil {
					rpcDot.FillColor = components.BadgeErr.Color()
					rpcLbl.SetText(pingErr.Error())
				} else {
					rpcDot.FillColor = components.BadgeOK.Color()
					rpcLbl.SetText(fmt.Sprintf("checkpoint %d", res.Checkpoint))
				}
				rpcDot.Refresh()
			})
		}()
	}

	pingBtn := widget.NewButtonWithIcon("Check", ftheme.ViewRefreshIcon(), pingAll)
	pingBtn.Importance = widget.HighImportance

	pingAll() // initial probe
	return components.Card(theme.AccentGC.Primary,
		"Connection",
		"GC API + IOTA RPC reachability",
		rows,
		pingBtn,
	)
}

// ---- whitelist card --------------------------------------------------------

func buildWhitelistCard(vc *ui.ViewContext) fyne.CanvasObject {
	table := components.NewDataTable(
		[]components.DataColumn{{Header: "Member address"}},
		"Whitelist is empty.",
	)

	var (
		all    []string
		filter string
	)
	statusLbl := widget.NewLabel("0 members")

	var openRemove func(member string)

	rebuild := func() {
		f := strings.ToLower(strings.TrimSpace(filter))
		rows := make([]components.DataRow, 0, len(all))
		for _, m := range all {
			if f != "" && !strings.Contains(strings.ToLower(m), f) {
				continue
			}
			mCopy := m
			rmBtn := widget.NewButtonWithIcon("Remove", ftheme.DeleteIcon(), func() { openRemove(mCopy) })
			rmBtn.Importance = widget.DangerImportance
			rows = append(rows, components.DataRow{
				Cells:   []string{mCopy},
				Actions: []fyne.CanvasObject{rmBtn},
			})
		}
		fyne.Do(func() {
			table.SetRows(rows)
			statusLbl.SetText(fmt.Sprintf("%d member(s)%s",
				len(all),
				visibleSuffix(len(all), len(rows))))
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

	search := widget.NewEntry()
	search.SetPlaceHolder("filter by address substring…")
	search.OnChanged = func(s string) {
		filter = s
		rebuild()
	}

	addBtn := widget.NewButtonWithIcon("Add member", ftheme.ContentAddIcon(), openAdd)
	addBtn.Importance = widget.HighImportance
	refreshBtn := widget.NewButtonWithIcon("Refresh", ftheme.ViewRefreshIcon(), refresh)

	toolbar := container.NewBorder(nil, nil, nil,
		container.NewHBox(refreshBtn, addBtn),
		search,
	)
	top := container.NewVBox(toolbar, statusLbl)

	body := container.NewBorder(top, nil, nil, nil, table.CanvasObject())
	refresh() // initial load

	return components.CardStretch(theme.AccentGC.Primary,
		"Whitelist members",
		"On-chain registry of approved addresses (GC-signed mutations)",
		body,
	)
}

func visibleSuffix(total, visible int) string {
	if total == visible {
		return ""
	}
	return fmt.Sprintf("  ·  %d visible", visible)
}
