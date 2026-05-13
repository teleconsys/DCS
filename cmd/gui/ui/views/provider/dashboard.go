package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
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
	"github.com/teleconsys/DCS/internal/offers"
)

// Dashboard is the Provider workspace: "Open offer windows" and "My offers"
// side by side with a draggable split between them.
func Dashboard(vc *ui.ViewContext) fyne.CanvasObject {
	windows := buildOpenWindowsCard(vc)
	mine := buildMyOffersCard(vc)

	split := container.NewHSplit(windows, mine)
	split.SetOffset(0.52)

	return container.NewStack(split)
}

// ---- open offer windows ---------------------------------------------------

func buildOpenWindowsCard(vc *ui.ViewContext) fyne.CanvasObject {
	table := components.NewDataTable(
		[]components.DataColumn{
			{Header: "CID"},
			{Header: "Owner"},
			{Header: "Closes in"},
			{Header: "Offers"},
		},
		"No open offer windows right now.",
	)
	statusLbl := widget.NewLabel("0 windows")

	var (
		latest    []offers.OpenOfferCID
		openSubmit func(window offers.OpenOfferCID)
	)

	rebuild := func() {
		dataRows := make([]components.DataRow, 0, len(latest))
		for _, w := range latest {
			wCopy := w
			submit := widget.NewButtonWithIcon("Submit offer", ftheme.UploadIcon(), func() { openSubmit(wCopy) })
			submit.Importance = widget.HighImportance
			dataRows = append(dataRows, components.DataRow{
				Cells: []string{
					components.Short(w.CID),
					components.Short(w.Owner),
					fmt.Sprintf("%d min", w.ClosesInMinutes),
					strconv.Itoa(w.NextOffers),
				},
				Actions: []fyne.CanvasObject{submit},
			})
		}
		fyne.Do(func() {
			table.SetRows(dataRows)
			statusLbl.SetText(fmt.Sprintf("%d window(s)", len(latest)))
		})
	}

	refresh := func() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			items, err := service.ListOpenOffers(ctx, vc.Snapshot(), io.Discard)
			if err != nil {
				fyne.Do(func() { dialog.ShowError(err, vc.Window) })
				return
			}
			latest = items
			rebuild()
		}()
	}

	openSubmit = func(window offers.OpenOfferCID) {
		amount := ui.NewAmountEntry("amount (IOTA nanos)", false)
		debug := widget.NewCheck("debug", nil)
		info := widget.NewLabel(fmt.Sprintf("CID %s  ·  owner %s  ·  closes in %d min",
			components.Short(window.CID),
			components.Short(window.Owner),
			window.ClosesInMinutes,
		))
		info.Wrapping = fyne.TextWrapWord
		form := widget.NewForm(
			widget.NewFormItem("", info),
			widget.NewFormItem("Amount", amount),
			widget.NewFormItem("", debug),
		)
		d := dialog.NewCustomConfirm("Submit offer", "Submit", "Cancel", form,
			func(ok bool) {
				if !ok {
					return
				}
				amt, _ := ui.ParseUint64(amount.Text)
				if amt == 0 {
					dialog.ShowError(errors.New("amount must be > 0"), vc.Window)
					return
				}
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
					defer cancel()
					_, err := service.SubmitOffer(ctx, vc.Snapshot(), service.SubmitOfferForm{
						CIDObjectID: window.ID,
						Amount:      amt,
						Debug:       debug.Checked,
					}, vc.Output)
					if err != nil {
						fyne.Do(func() { dialog.ShowError(err, vc.Window) })
						return
					}
					refresh()
				}()
			}, vc.Window)
		d.Resize(fyne.NewSize(560, 260))
		d.Show()
	}

	refreshBtn := widget.NewButtonWithIcon("Refresh", ftheme.ViewRefreshIcon(), refresh)
	header := container.NewBorder(nil, nil, statusLbl, refreshBtn)
	body := container.NewBorder(header, nil, nil, nil, table.CanvasObject())

	refresh()

	// Auto-refresh every 60 s. The goroutine lives for the dashboard
	// lifetime — acceptable because the dashboard is built once per
	// session.
	go func() {
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		for range t.C {
			refresh()
		}
	}()

	return components.CardStretch(theme.AccentProvider.Primary,
		"Open offer windows",
		"",
		body,
	)
}

// ---- my offers ------------------------------------------------------------

func buildMyOffersCard(vc *ui.ViewContext) fyne.CanvasObject {
	table := components.NewDataTable(
		[]components.DataColumn{
			{Header: "CID"},
			{Header: "Amount (nanos)"},
			{Header: "Approved"},
			{Header: "Honored"},
			{Header: "Withdrawn"},
		},
		"No active offers as Provider.",
	)
	statusLbl := widget.NewLabel("0 offers")

	var (
		allOffers   []service.OfferRow
		openWithdraw func(o service.OfferRow)
	)

	rebuild := func() {
		dataRows := make([]components.DataRow, 0, len(allOffers))
		for _, o := range allOffers {
			oCopy := o
			var actions []fyne.CanvasObject
			switch {
			case o.Withdrawn:
				actions = append(actions, widget.NewLabel("paid"))
			case o.Honored:
				wd := widget.NewButtonWithIcon("Withdraw", ftheme.DownloadIcon(), func() { openWithdraw(oCopy) })
				wd.Importance = widget.HighImportance
				actions = append(actions, wd)
			case o.Approved:
				actions = append(actions, widget.NewLabel("awaiting honor"))
			default:
				actions = append(actions, widget.NewLabel("pending"))
			}
			dataRows = append(dataRows, components.DataRow{
				Cells: []string{
					components.Short(o.CIDStr),
					strconv.FormatInt(o.Amount, 10),
					boolMark(o.Approved),
					boolMark(o.Honored),
					boolMark(o.Withdrawn),
				},
				Actions: actions,
			})
		}
		fyne.Do(func() {
			table.SetRows(dataRows)
			statusLbl.SetText(fmt.Sprintf("%d offer(s)", len(allOffers)))
		})
	}

	refresh := func() {
		go func() {
			snap := vc.Snapshot()
			if strings.TrimSpace(snap.Address) == "" {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			ofs, err := service.MyOffersAsProvider(ctx, snap, snap.Address)
			if err != nil {
				fyne.Do(func() { dialog.ShowError(err, vc.Window) })
				return
			}
			allOffers = ofs
			rebuild()
		}()
	}

	openWithdraw = func(o service.OfferRow) {
		dialog.ShowConfirm("Withdraw payment",
			fmt.Sprintf("Withdraw payment from offer #%d on CID %s?", o.Index, components.Short(o.CIDStr)),
			func(ok bool) {
				if !ok {
					return
				}
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
					defer cancel()
					_, err := service.Withdraw(ctx, vc.Snapshot(), service.OfferIndexForm{
						CIDObjectID: o.CIDObjectID,
						Index:       uint64(o.Index),
					}, vc.Output)
					if err != nil {
						fyne.Do(func() { dialog.ShowError(err, vc.Window) })
						return
					}
					refresh()
				}()
			}, vc.Window)
	}

	refreshBtn := widget.NewButtonWithIcon("Refresh", ftheme.ViewRefreshIcon(), refresh)
	header := container.NewBorder(nil, nil, statusLbl, refreshBtn)
	body := container.NewBorder(header, nil, nil, nil, table.CanvasObject())

	refresh()

	return components.CardStretch(theme.AccentProvider.Primary,
		"My offers",
		"",
		body,
	)
}

// ---- helpers --------------------------------------------------------------

func boolMark(b bool) string {
	if b {
		return "✓"
	}
	return "—"
}
