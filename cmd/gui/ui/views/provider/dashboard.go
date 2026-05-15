package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	ftheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/theme"
	"github.com/teleconsys/DCS/cmd/gui/ui"
	"github.com/teleconsys/DCS/cmd/gui/ui/components"
	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
	"github.com/teleconsys/DCS/internal/offers"
)

// Dashboard is the Provider workspace: My Offers full width; open offer
// windows open from the "New" button in a modal.
func Dashboard(vc *ui.ViewContext) fyne.CanvasObject {
	var refreshMyOffers func(bool)

	windowsBody, refreshWindows := buildOpenWindowsPanel(vc, func() {
		if refreshMyOffers != nil {
			refreshMyOffers(false)
		}
	})

	openWindowsModal := func() {
		refreshWindows()
		scroll := container.NewScroll(windowsBody)
		scroll.SetMinSize(fyne.NewSize(640, 360))
		var d *dialog.CustomDialog
		closeBtn := widget.NewButton("Close", func() {
			if d != nil {
				d.Hide()
			}
			if refreshMyOffers != nil {
				refreshMyOffers(false)
			}
		})
		footer := container.NewHBox(layout.NewSpacer(), closeBtn)
		body := container.NewBorder(nil, footer, nil, nil, scroll)
		d = dialog.NewCustomWithoutButtons("Open offer windows", body, vc.Window)
		d.Resize(fyne.NewSize(760, 480))
		d.Show()
	}

	card, refreshFn := buildMyOffersCard(vc, openWindowsModal)
	refreshMyOffers = refreshFn
	return container.NewStack(card)
}

// ---- open offer windows (panel for modal) ---------------------------------

func buildOpenWindowsPanel(vc *ui.ViewContext, onOffersChanged func()) (fyne.CanvasObject, func()) {
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
		latest     []offers.OpenOfferCID
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

	refresh := func(showOKDialog bool) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			items, err := service.ListOpenOffers(ctx, vc.Snapshot(), io.Discard)
			if err != nil {
				fyne.Do(func() {
					if vc.Feedback != nil {
						vc.Feedback.Show(feedback.Error, err.Error())
					}
				})
				return
			}
			latest = items
			rebuild()
			if showOKDialog {
				fyne.Do(func() {
					countText := fmt.Sprintf("%d window(s)", len(latest))
					feedback.FlashStatus(statusLbl, "Refreshed", feedback.Success, func() {
						statusLbl.SetText(countText)
					}, 2*time.Second)
				})
			}
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

		okTitle := widget.NewLabelWithStyle("Offer submitted", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		okTitle.Importance = widget.SuccessImportance
		okSub := widget.NewLabel("Your offer is on chain. The list will refresh when you close this dialog.")
		okSub.Wrapping = fyne.TextWrapWord
		okSub.Alignment = fyne.TextAlignCenter
		successPane := container.NewVBox(
			container.NewCenter(okTitle),
			okSub,
		)
		successPane.Hide()

		formPane := container.NewVBox(form)
		content := container.NewStack(formPane, successPane)

		var d *dialog.CustomDialog
		submitBtn := widget.NewButton("Submit", nil)
		submitBtn.Importance = widget.HighImportance
		cancelBtn := widget.NewButton("Cancel", func() {
			if d != nil {
				d.Hide()
			}
		})
		submitBtn.OnTapped = func() {
			amt, _ := ui.ParseUint64(amount.Text)
			if amt == 0 {
				if vc.Feedback != nil {
					vc.Feedback.Show(feedback.Error, "Amount must be > 0.")
				}
				return
			}
			submitBtn.Disable()
			cancelBtn.Disable()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
				defer cancel()
				var buf bytes.Buffer
				out := vc.TeeOutput(&buf)
				_, err := service.SubmitOffer(ctx, vc.Snapshot(), service.SubmitOfferForm{
					CIDObjectID: window.ID,
					Amount:      amt,
					Debug:       debug.Checked,
				}, out)
				fyne.Do(func() {
					if err != nil {
						submitBtn.Enable()
						cancelBtn.Enable()
						if vc.Feedback != nil {
							vc.Feedback.Show(feedback.Error, err.Error())
						}
						return
					}
					formPane.Hide()
					successPane.Show()
					content.Refresh()
					submitBtn.Hide()
					cancelBtn.SetText("Done")
					cancelBtn.Enable()
					cancelBtn.OnTapped = func() {
						if d != nil {
							d.Hide()
						}
					}
					if vc.Feedback != nil {
						vc.Feedback.Show(feedback.Success, "Offer submitted successfully.")
					}
					refresh(false)
					if onOffersChanged != nil {
						onOffersChanged()
					}
				})
			}()
		}

		footer := container.NewHBox(cancelBtn, layout.NewSpacer(), submitBtn)
		body := container.NewBorder(nil, footer, nil, nil, content)
		d = dialog.NewCustomWithoutButtons("Submit offer", body, vc.Window)
		d.Resize(fyne.NewSize(560, 300))
		d.Show()
	}

	refreshBtn := widget.NewButtonWithIcon("Refresh", ftheme.ViewRefreshIcon(), func() { refresh(true) })
	header := container.NewBorder(nil, nil, statusLbl, refreshBtn)
	body := container.NewBorder(header, nil, nil, nil, table.CanvasObject())

	refresh(false)

	// Auto-refresh every 60 s. The goroutine lives for the dashboard
	// lifetime — acceptable because the dashboard is built once per
	// session.
	go func() {
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		for range t.C {
			refresh(false)
		}
	}()

	return body, func() { refresh(false) }
}

// ---- my offers ------------------------------------------------------------

func buildMyOffersCard(vc *ui.ViewContext, onNew func()) (fyne.CanvasObject, func(bool)) {
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

	refresh := func(showOKDialog bool) {
		go func() {
			snap := vc.Snapshot()
			if strings.TrimSpace(snap.Address) == "" {
				fyne.Do(func() {
					if showOKDialog && vc.Feedback != nil {
						vc.Feedback.Show(feedback.Error, "Set your address in Identity first.")
					}
				})
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			ofs, err := service.MyOffersAsProvider(ctx, snap, snap.Address)
			if err != nil {
				fyne.Do(func() {
					if vc.Feedback != nil {
						vc.Feedback.Show(feedback.Error, err.Error())
					}
				})
				return
			}
			allOffers = ofs
			rebuild()
			if showOKDialog {
				fyne.Do(func() {
					countText := fmt.Sprintf("%d offer(s)", len(allOffers))
					feedback.FlashStatus(statusLbl, "Refreshed", feedback.Success, func() {
						statusLbl.SetText(countText)
					}, 2*time.Second)
				})
			}
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
					var buf bytes.Buffer
					out := vc.TeeOutput(&buf)
					_, err := service.Withdraw(ctx, vc.Snapshot(), service.OfferIndexForm{
						CIDObjectID: o.CIDObjectID,
						Index:       uint64(o.Index),
					}, out)
					if err != nil {
						fyne.Do(func() {
							if vc.Feedback != nil {
								vc.Feedback.Show(feedback.Error, err.Error())
							}
						})
						return
					}
					body := ui.RunSuccessBody("withdraw", "Withdraw", buf.String())
					fyne.Do(func() {
						if vc.Feedback != nil {
							vc.Feedback.Show(feedback.Success, feedback.FirstLine(body))
						}
					})
					refresh(false)
				}()
			}, vc.Window)
	}

	newBtn := widget.NewButton("New", onNew)
	newBtn.Importance = widget.HighImportance
	refreshBtn := widget.NewButtonWithIcon("Refresh", ftheme.ViewRefreshIcon(), func() { refresh(true) })
	toolbar := container.NewHBox(newBtn, refreshBtn)
	header := container.NewBorder(nil, nil, statusLbl, toolbar)
	body := container.NewBorder(header, nil, nil, nil, table.CanvasObject())

	refresh(false)

	return components.CardStretch(theme.AccentProvider.Primary,
			"My Offers",
			"",
			body,
		), refresh
}

// ---- helpers --------------------------------------------------------------

func boolMark(b bool) string {
	if b {
		return "✓"
	}
	return "—"
}
