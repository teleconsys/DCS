package user

import (
	"bytes"
	"context"
	"fmt"
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
	"github.com/teleconsys/DCS/cmd/gui/ui/wizards"
)

// userDashboardOpts tweaks the User dashboard layout (Demo uses compact scroll mins).
type userDashboardOpts struct {
	compactSplitMins bool
}

// Dashboard is the User workspace: My CIDs full width; IPFS tools open from the header.
func Dashboard(vc *ui.ViewContext) fyne.CanvasObject {
	return buildUserDashboard(vc, userDashboardOpts{})
}

// DashboardForDemo is like Dashboard but uses compact scroll minimums for the Demo tab.
func DashboardForDemo(vc *ui.ViewContext) fyne.CanvasObject {
	return buildUserDashboard(vc, userDashboardOpts{compactSplitMins: true})
}

func buildUserDashboard(vc *ui.ViewContext, opts userDashboardOpts) fyne.CanvasObject {
	var cidScrollMin fyne.Size
	if opts.compactSplitMins {
		cidScrollMin = fyne.NewSize(120, 96)
	}
	grid, _ := buildMyCIDsCard(vc, cidScrollMin)
	return container.NewStack(components.CardStretch(theme.AccentUser.Primary, "My CIDs", "", grid))
}

// ---- My CIDs grid ---------------------------------------------------------

func buildMyCIDsCard(vc *ui.ViewContext, cidScrollMin fyne.Size) (fyne.CanvasObject, func()) {
	if cidScrollMin.Width <= 0 || cidScrollMin.Height <= 0 {
		cidScrollMin = fyne.NewSize(220, 160)
	}
	// RowWrapLayout: each CID card keeps its natural MinSize width; tiles wrap
	// into additional rows as the My CIDs pane widens or narrows (no stretching
	// to fill equal-width columns).
	grid := container.New(layout.NewRowWrapLayout())
	cidScroll := container.NewVScroll(grid)
	cidScroll.SetMinSize(cidScrollMin)

	statusLbl := widget.NewLabel("0 CIDs")
	emptyLbl := components.EmptyState(
		ftheme.ContentAddIcon(),
		"No CIDs yet",
		"Use Add (above) to create your first CID.",
		"", nil,
	)
	host := container.NewMax(emptyLbl)

	var refresh func(showOKDialog bool)
	refresh = func(showOKDialog bool) {
		go func() {
			snap := vc.Snapshot()
			if strings.TrimSpace(snap.Address) == "" {
				fyne.Do(func() {
					statusLbl.SetText("(no address)")
					if showOKDialog && vc.Feedback != nil {
						vc.Feedback.Show(feedback.Error, "Set your address in Identity first.")
					}
				})
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			cids, err := service.MyCIDs(ctx, snap, snap.Address)
			if err != nil {
				fyne.Do(func() {
					statusLbl.SetText("Refresh failed")
					if vc.Feedback != nil {
						vc.Feedback.Show(feedback.Error, err.Error())
					}
				})
				return
			}
			fyne.Do(func() {
				grid.RemoveAll()
				for _, c := range cids {
					grid.Add(buildCIDTile(vc, c, refresh))
				}
				grid.Refresh()
				if len(cids) == 0 {
					host.Objects = []fyne.CanvasObject{emptyLbl}
				} else {
					host.Objects = []fyne.CanvasObject{cidScroll}
				}
				host.Refresh()
				countText := fmt.Sprintf("%d CID(s)", len(cids))
				statusLbl.SetText(countText)
				if showOKDialog {
					feedback.FlashStatus(statusLbl, "Refreshed", feedback.Success, func() {
						statusLbl.SetText(countText)
					}, 2*time.Second)
				}
			})
		}()
	}

	addBtn := widget.NewButtonWithIcon("Add", ftheme.ContentAddIcon(), func() {
		wizards.NewCIDWizard(vc, func() { refresh(false) })
	})
	addBtn.Importance = widget.HighImportance

	refreshBtn := widget.NewButtonWithIcon("Refresh", ftheme.ViewRefreshIcon(), func() { refresh(true) })
	ipfsBtn := widget.NewButtonWithIcon("IPFS", ftheme.FolderIcon(), func() { OpenIPFSModal(vc) })
	headerRight := container.NewHBox(ipfsBtn, addBtn, refreshBtn)
	header := container.NewBorder(nil, nil, statusLbl, headerRight)
	body := container.NewBorder(header, nil, nil, nil, host)

	refresh(false)

	return body, func() { refresh(false) }
}

func buildCIDTile(vc *ui.ViewContext, sum service.CIDSummary, refresh func(showOKDialog bool)) fyne.CanvasObject {
	state := badgeStateForCID(sum.Status)
	phaseTimer := components.NewLivePhaseTimer(sum)
	now := time.Now().UnixMilli()

	var balReadable string
	if sum.Balance >= 0 {
		balReadable = formatIOTA(uint64(sum.Balance)) + " IOTA"
	} else {
		balReadable = strconv.FormatInt(sum.Balance, 10)
	}

	header := components.CIDHeaderRow(vc.Window, sum.CIDStr, state, sum.Status.String())
	stats := container.NewVBox(
		components.LabeledWidget("Phase", phaseTimer),
		components.LabeledField("Balance", balReadable),
		components.LabeledWidget("Offers", components.OfferCountStrip(sum.NextOffers, sum.CurrentOffers, sum.PrevOffers)),
	)
	body := container.NewVBox(header, widget.NewSeparator(), stats)

	addFunds := widget.NewButtonWithIcon("Add funds", ftheme.ContentAddIcon(), func() {
		openAddFundsDialog(vc, sum, refresh)
	})
	offersBtn := widget.NewButtonWithIcon("Offers", ftheme.ListIcon(), func() {
		openOffersDialog(vc, sum, refresh)
	})
	nextEp := widget.NewButtonWithIcon("Next epoch", ftheme.MediaSkipNextIcon(), func() {
		openNextEpochDialog(vc, sum, refresh)
	})
	if sum.NextEpochStart == 0 || now < sum.NextEpochStart {
		nextEp.Disable()
	}
	remove := widget.NewButtonWithIcon("Remove", ftheme.DeleteIcon(), func() {
		openRemoveDialog(vc, sum, refresh)
	})
	remove.Importance = widget.DangerImportance

	actions := container.NewGridWithColumns(2, addFunds, offersBtn, nextEp, remove)
	return components.TileCard(body, actions)
}

func badgeStateForCID(s service.CIDStatus) components.BadgeState {
	switch s {
	case service.CIDStatusPending:
		return components.BadgeOK
	case service.CIDStatusOfferWindow:
		return components.BadgeWarn
	case service.CIDStatusNextScheduled:
		return components.BadgeInfo
	case service.CIDStatusExpired:
		return components.BadgeErr
	}
	return components.BadgeNeutral
}

// ---- per-tile dialogs -----------------------------------------------------

func openAddFundsDialog(vc *ui.ViewContext, sum service.CIDSummary, refresh func(showOKDialog bool)) {
	amount := ui.NewAmountEntry("amount (IOTA nanos)", false)
	form := widget.NewForm(
		widget.NewFormItem("CID", widget.NewLabel(components.Short(sum.CIDStr))),
		widget.NewFormItem("Amount", amount),
	)
	var d dialog.Dialog
	d = dialog.NewCustomConfirm("Add funds", "Deposit", "Cancel", form,
		func(ok bool) {
			if !ok {
				return
			}
			amt, _ := ui.ParseUint64(amount.Text)
			if amt == 0 {
				if vc.Feedback != nil {
					vc.Feedback.Show(feedback.Error, "Amount must be > 0.")
				}
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				var buf bytes.Buffer
				out := vc.TeeOutput(&buf)
				_, err := service.CIDAddFunds(ctx, vc.Snapshot(), sum.ID, amt, out)
				if err != nil {
					fyne.Do(func() {
						if vc.Feedback != nil {
							vc.Feedback.Show(feedback.Error, err.Error())
						}
					})
					return
				}
				body := ui.RunSuccessBody("cid add-funds", "Add funds", buf.String())
				fyne.Do(func() {
					if d != nil {
						d.Hide()
					}
					if vc.Feedback != nil {
						vc.Feedback.Show(feedback.Success, feedback.FirstLine(body))
					}
				})
				refresh(false)
			}()
		}, vc.Window)
	d.Resize(fyne.NewSize(520, 240))
	d.Show()
}

func openNextEpochDialog(vc *ui.ViewContext, sum service.CIDSummary, refresh func(showOKDialog bool)) {
	dialog.ShowConfirm("Transition to next epoch",
		fmt.Sprintf("Move CID %s into its next epoch?", components.Short(sum.CIDStr)),
		func(ok bool) {
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				var buf bytes.Buffer
				out := vc.TeeOutput(&buf)
				if err := service.CIDTransitionEpoch(ctx, vc.Snapshot(), sum.ID, out); err != nil {
					fyne.Do(func() {
						if vc.Feedback != nil {
							vc.Feedback.Show(feedback.Error, err.Error())
						}
					})
					return
				}
				body := ui.RunSuccessBody("cid next-epoch", "Next epoch", buf.String())
				fyne.Do(func() {
					if vc.Feedback != nil {
						vc.Feedback.Show(feedback.Success, feedback.FirstLine(body))
					}
				})
				refresh(false)
			}()
		}, vc.Window)
}

func openRemoveDialog(vc *ui.ViewContext, sum service.CIDSummary, refresh func(showOKDialog bool)) {
	dialog.ShowConfirm("Remove CID",
		fmt.Sprintf("Remove %s from the CID list? This cannot be undone.", components.Short(sum.CIDStr)),
		func(ok bool) {
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				var buf bytes.Buffer
				out := vc.TeeOutput(&buf)
				_, err := service.CIDRemove(ctx, vc.Snapshot(), sum.ID, out)
				if err != nil {
					fyne.Do(func() {
						if vc.Feedback != nil {
							vc.Feedback.Show(feedback.Error, err.Error())
						}
					})
					return
				}
				fyne.Do(func() { refresh(false) })
			}()
		}, vc.Window)
}

// openOffersDialog shows next/current/prev offer arrays for a CID in
// separate tabs, with Approve / Honor row actions where appropriate.
func openOffersDialog(vc *ui.ViewContext, sum service.CIDSummary, refresh func(showOKDialog bool)) {
	openOffersDialogWithTab(vc, sum, refresh, "")
}

func openOffersDialogWithTab(vc *ui.ViewContext, sum service.CIDSummary, refresh func(showOKDialog bool), selectTab string) {
	loading := widget.NewLabel("Loading offers…")
	root := container.NewStack(loading)

	var (
		d    dialog.Dialog
		tabs *container.AppTabs
	)
	reload := func() {
		sel := selectTab
		if tabs != nil && tabs.Selected() != nil {
			sel = offersTabEpoch(tabs.Selected().Text)
		}
		if d != nil {
			d.Hide()
		}
		openOffersDialogWithTab(vc, sum, refresh, sel)
	}

	loadFn := func() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			next, current, prev, err := service.OffersForCID(ctx, vc.Snapshot(), sum.ID)
			fyne.Do(func() {
				if err != nil {
					errLbl := widget.NewLabel("Error: " + err.Error())
					errLbl.Wrapping = fyne.TextWrapWord
					errLbl.Importance = widget.DangerImportance
					root.Objects = []fyne.CanvasObject{errLbl}
					root.Refresh()
					return
				}
				tabs = container.NewAppTabs(
					container.NewTabItem(offersTabLabel("Previous", len(prev)),
						buildOffersEpochTab(vc, sum, prev, "", "Settled offers from past epochs.", refresh, nil)),
					container.NewTabItem(offersTabLabel("Current", len(current)),
						buildOffersEpochTab(vc, sum, current, "honor", "", refresh, reload)),
					container.NewTabItem(offersTabLabel("Next", len(next)),
						buildOffersEpochTab(vc, sum, next, "approve", "", refresh, reload)),
				)
				tabs.SetTabLocation(container.TabLocationTop)
				root.Objects = []fyne.CanvasObject{tabs}
				root.Refresh()

				pick := selectTab
				if pick == "" {
					pick = "Current"
				}
				for _, item := range tabs.Items {
					if offersTabEpoch(item.Text) == pick {
						tabs.Select(item)
						return
					}
				}
			})
		}()
	}

	d = dialog.NewCustom("Offers — "+components.Short(sum.CIDStr), "Close", root, vc.Window)
	d.Resize(fyne.NewSize(720, 600))
	d.Show()
	loadFn()
}

func offersTabLabel(epoch string, n int) string {
	if n == 0 {
		return epoch
	}
	return fmt.Sprintf("%s (%d)", epoch, n)
}

func offersTabEpoch(label string) string {
	if i := strings.Index(label, " ("); i >= 0 {
		return label[:i]
	}
	return label
}

func buildOffersEpochTab(vc *ui.ViewContext, sum service.CIDSummary, rows []service.OfferRow,
	mode, hint string, refresh func(showOKDialog bool), reload func()) fyne.CanvasObject {
	body := container.NewVBox()
	if banner := offersTabBanner(sum, mode); banner != nil {
		body.Add(banner)
	}
	if hint != "" {
		h := widget.NewLabel(hint)
		h.Wrapping = fyne.TextWrapWord
		body.Add(h)
	}
	body.Add(buildOffersSection(vc, sum, rows, mode, refresh, reload))
	return container.NewVScroll(body)
}

func offersTabBanner(sum service.CIDSummary, mode string) fyne.CanvasObject {
	switch mode {
	case "approve":
		return components.NewLiveCountdownLabel(func() string {
			now := time.Now().UnixMilli()
			if service.CanApproveOffers(sum, now) {
				return "Offer phase ended — you can approve offers for the next epoch."
			}
			if service.IsOfferWindowOpen(sum, now) {
				left := service.FormatCountdown(service.RemainingMS(service.OfferWindowEndsAtMs(sum), now))
				return "Offer phase still open — approvals unlock in " + left
			}
			return "Approvals are not available for this CID yet."
		})
	case "honor":
		return components.NewLiveCountdownLabel(func() string {
			now := time.Now().UnixMilli()
			left := service.FormatCountdown(service.RemainingMS(service.HonorDeadlineMs(sum, now), now))
			return "Time left to honor offers: " + left
		})
	default:
		return nil
	}
}

func buildOffersSection(vc *ui.ViewContext, sum service.CIDSummary,
	rows []service.OfferRow, mode string, refresh func(showOKDialog bool), reload func()) fyne.CanvasObject {
	if len(rows) == 0 {
		l := widget.NewLabel("  (none)")
		l.TextStyle = fyne.TextStyle{Italic: true}
		return l
	}
	out := container.NewVBox()
	for _, o := range rows {
		oCopy := o
		stateLbl := widget.NewLabel(offerStateLabel(o))
		action := buildOfferAction(vc, sum, oCopy, mode, refresh, reload, stateLbl)
		row := container.NewGridWithColumns(4,
			widget.NewLabel("#"+strconv.Itoa(o.Index)+"  "+components.Short(o.Provider)),
			widget.NewLabel(strconv.FormatInt(o.Amount, 10)+" nanos"),
			stateLbl,
			action,
		)
		out.Add(row)
	}
	return out
}

func offerStateLabel(o service.OfferRow) string {
	switch {
	case o.Withdrawn:
		return "paid"
	case o.Honored:
		return "honored"
	case o.Approved:
		return "approved"
	}
	return "pending"
}

func buildOfferAction(vc *ui.ViewContext, sum service.CIDSummary, o service.OfferRow,
	mode string, refresh func(showOKDialog bool), reload func(), stateLbl *widget.Label) fyne.CanvasObject {
	switch mode {
	case "approve":
		if o.Approved {
			return widget.NewLabel("approved")
		}
		oCopy := o
		return components.NewLiveGatedButton("Approve", "offer phase", func() bool {
			return service.CanApproveOffers(sum, time.Now().UnixMilli())
		}, func() {
			now := time.Now().UnixMilli()
			if service.IsOfferWindowOpen(sum, now) {
				left := service.FormatCountdown(service.RemainingMS(service.OfferWindowEndsAtMs(sum), now))
				if vc.Feedback != nil {
					vc.Feedback.Show(feedback.Error, "Offer phase not over — try again in "+left+".")
				}
				return
			}
			runOfferIndex(vc, sum, oCopy, "approve", refresh, reload, nil, stateLbl)
		})
	case "honor":
		if o.Honored {
			return widget.NewLabel("honored")
		}
		if !o.Approved {
			return widget.NewLabel("not approved")
		}
		var btn *widget.Button
		btn = widget.NewButton("Honor", func() {
			runOfferIndex(vc, sum, o, "honor", refresh, reload, btn, stateLbl)
		})
		btn.Importance = widget.HighImportance
		return btn
	}
	return widget.NewLabel("—")
}

func runOfferIndex(vc *ui.ViewContext, sum service.CIDSummary, o service.OfferRow,
	mode string, refresh func(showOKDialog bool), reload func(),
	btn *widget.Button, stateLbl *widget.Label) {
	if btn != nil {
		fyne.Do(func() { btn.Disable() })
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		var buf bytes.Buffer
		out := vc.TeeOutput(&buf)
		var err error
		switch mode {
		case "approve":
			_, err = service.ApproveOffer(ctx, vc.Snapshot(), service.OfferIndexForm{
				CIDObjectID: sum.ID,
				Index:       uint64(o.Index),
			}, out)
		case "honor":
			_, err = service.HonorOffer(ctx, vc.Snapshot(), service.OfferIndexForm{
				CIDObjectID: sum.ID,
				Index:       uint64(o.Index),
			}, out)
		}
		if err != nil {
			fyne.Do(func() {
				if btn != nil {
					btn.Enable()
				}
				if stateLbl != nil {
					feedback.ApplyLabel(stateLbl, "failed", feedback.Error)
				}
				if vc.Feedback != nil {
					vc.Feedback.Show(feedback.Error, err.Error())
				}
			})
			return
		}
		fyne.Do(func() {
			if btn != nil {
				btn.Enable()
			}
		})
		refresh(false)
		if reload != nil {
			reload()
		}
	}()
}

// ---- helpers --------------------------------------------------------------

func formatIOTA(nanos uint64) string {
	whole := nanos / 1_000_000_000
	frac := nanos % 1_000_000_000
	s := fmt.Sprintf("%d.%09d", whole, frac)
	s = strings.TrimRight(s, "0")
	if strings.HasSuffix(s, ".") {
		s += "0"
	}
	return s
}
