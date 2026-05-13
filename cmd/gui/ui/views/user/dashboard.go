package user

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
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
	"github.com/teleconsys/DCS/cmd/gui/ui/shell"
	"github.com/teleconsys/DCS/cmd/gui/ui/wizards"
)

// Dashboard is the redesigned User workspace: wallet + whitelisted
// strip on top, hero "+ Upload" CTA, then a tile grid of "My CIDs"
// with per-tile actions, and a collapsed "Tools" accordion at the
// bottom holding the residual utility actions.
func Dashboard(vc *ui.ViewContext) fyne.CanvasObject {
	// Top strip cards.
	wallet, refreshWallet := buildWalletCard(vc)
	// Grid + refresh handle (also fires when the wizard returns).
	grid, refreshGrid := buildMyCIDsCard(vc)

	hero := buildHero(vc, func() {
		refreshGrid()
		refreshWallet()
	})

	topStrip := container.NewGridWithColumns(1, wallet)
	header := container.NewVBox(topStrip, hero)

	tools := buildTools(vc)

	body := container.NewBorder(header, tools, nil, nil, grid)
	return body
}

// ---- wallet card ----------------------------------------------------------

func buildWalletCard(vc *ui.ViewContext) (fyne.CanvasObject, func()) {
	balLbl := widget.NewLabel("…")
	wlDot := canvas.NewCircle(components.BadgeNeutral.Color())
	wlDotBox := container.NewGridWrap(fyne.NewSize(12, 12), wlDot)
	wlLbl := widget.NewLabel("checking…")

	rows := container.NewVBox(
		components.KeyValueRow(vc.Window, "Address", vc.Profile.Address, true),
		components.KeyValueRow(vc.Window, "Gas coin", vc.Profile.GasCoinID, true),
		container.NewHBox(
			widget.NewLabelWithStyle("IOTA balance:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			balLbl,
		),
		container.NewHBox(
			widget.NewLabelWithStyle("Whitelisted:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			wlDotBox,
			wlLbl,
		),
	)

	refresh := func() {
		go func() {
			snap := vc.Snapshot()
			if strings.TrimSpace(snap.Address) == "" {
				fyne.Do(func() {
					balLbl.SetText("(no address — open Identity)")
					wlDot.FillColor = components.BadgeNeutral.Color()
					wlDot.Refresh()
					wlLbl.SetText("unknown")
				})
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()

			bal, balErr := service.IOTABalance(ctx, snap.RPCURL, snap.Address)
			fyne.Do(func() {
				if balErr != nil {
					balLbl.SetText("error: " + balErr.Error())
				} else {
					balLbl.SetText(fmt.Sprintf("%s IOTA  (%s nanos)", formatIOTA(bal), strconv.FormatUint(bal, 10)))
				}
			})

			found, wlErr := service.WhitelistHas(ctx, snap, snap.Address, io.Discard)
			fyne.Do(func() {
				if wlErr != nil {
					wlDot.FillColor = components.BadgeErr.Color()
					wlLbl.SetText(wlErr.Error())
				} else if found {
					wlDot.FillColor = components.BadgeOK.Color()
					wlLbl.SetText("yes — you can upload content")
				} else {
					wlDot.FillColor = components.BadgeWarn.Color()
					wlLbl.SetText("no — ask GC to add you")
				}
				wlDot.Refresh()
			})
		}()
	}

	refreshBtn := widget.NewButtonWithIcon("Refresh", ftheme.ViewRefreshIcon(), refresh)
	manageBtn := widget.NewButtonWithIcon("Manage identity", ftheme.AccountIcon(), func() {
		shell.OpenIdentity(vc.Window, vc.App, vc.Profile.Actor, refresh)
	})

	refresh()
	subtitle := "address " + components.Short(vc.Profile.Address)
	return components.Card(theme.AccentUser.Primary, "My wallet", subtitle, rows, manageBtn, refreshBtn), refresh
}

// ---- hero CTA -------------------------------------------------------------

func buildHero(vc *ui.ViewContext, onCreated func()) fyne.CanvasObject {
	cta := widget.NewButtonWithIcon("+ Upload new content", ftheme.UploadIcon(), func() {
		wizards.NewCIDWizard(vc, onCreated)
	})
	cta.Importance = widget.HighImportance

	hint := widget.NewLabel("Pin a file to IPFS, register a CID object on chain, and seed it with an initial storage budget — all in one wizard.")
	hint.Wrapping = fyne.TextWrapWord

	return components.Card(theme.AccentUser.Primary,
		"Upload content",
		"",
		container.NewVBox(hint),
		cta,
	)
}

// ---- My CIDs grid ---------------------------------------------------------

func buildMyCIDsCard(vc *ui.ViewContext) (fyne.CanvasObject, func()) {
	grid := container.NewGridWrap(fyne.NewSize(340, 260))
	statusLbl := widget.NewLabel("0 CIDs")
	emptyLbl := components.EmptyState(
		ftheme.UploadIcon(),
		"No CIDs yet",
		"Click \"+ Upload new content\" above to publish your first file.",
		"", nil,
	)
	host := container.NewMax(emptyLbl)

	var refresh func()
	refresh = func() {
		go func() {
			snap := vc.Snapshot()
			if strings.TrimSpace(snap.Address) == "" {
				fyne.Do(func() {
					statusLbl.SetText("(no address)")
				})
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			cids, err := service.MyCIDs(ctx, snap, snap.Address)
			if err != nil {
				fyne.Do(func() { dialog.ShowError(err, vc.Window) })
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
					host.Objects = []fyne.CanvasObject{container.NewVScroll(grid)}
				}
				host.Refresh()
				statusLbl.SetText(fmt.Sprintf("%d CID(s)", len(cids)))
			})
		}()
	}

	refreshBtn := widget.NewButtonWithIcon("Refresh", ftheme.ViewRefreshIcon(), refresh)
	header := container.NewBorder(nil, nil, statusLbl, refreshBtn)
	body := container.NewBorder(header, nil, nil, nil, host)

	refresh()

	return components.CardStretch(theme.AccentUser.Primary,
		"My CIDs",
		"Content you've registered on chain",
		body,
	), refresh
}

func buildCIDTile(vc *ui.ViewContext, sum service.CIDSummary, refresh func()) fyne.CanvasObject {
	state := badgeStateForCID(sum.Status)
	badge := components.Badge(state, sum.Status.String())

	cidLbl := widget.NewLabelWithStyle(components.Short(sum.CIDStr), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	now := time.Now().UnixMilli()
	winLbl := widget.NewLabel(windowText(sum, now))
	winLbl.Wrapping = fyne.TextWrapWord
	balLbl := widget.NewLabel(fmt.Sprintf("Balance: %s nanos", strconv.FormatInt(sum.Balance, 10)))
	offersLbl := widget.NewLabel(fmt.Sprintf("Offers · next %d  current %d  prev %d",
		sum.NextOffers, sum.CurrentOffers, sum.PrevOffers))

	body := container.NewVBox(badge, cidLbl, winLbl, balLbl, offersLbl)

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
	return components.Card(theme.AccentUser.Primary, "", "", body, actions)
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

func windowText(sum service.CIDSummary, nowMs int64) string {
	if sum.CurrentEpochEnd == 0 {
		return "no epoch data"
	}
	switch sum.Status {
	case service.CIDStatusPending:
		mins := (sum.CurrentEpochEnd - nowMs) / 60_000
		return fmt.Sprintf("Active until in %d min", mins)
	case service.CIDStatusOfferWindow:
		mins := (sum.NextEpochStart - nowMs) / 60_000
		return fmt.Sprintf("Offers open · next epoch starts in %d min", mins)
	case service.CIDStatusNextScheduled:
		mins := (sum.NextEpochEnd - nowMs) / 60_000
		return fmt.Sprintf("Next epoch active for %d min", mins)
	case service.CIDStatusExpired:
		return "Expired — consider removing"
	}
	return ""
}

// ---- per-tile dialogs -----------------------------------------------------

func openAddFundsDialog(vc *ui.ViewContext, sum service.CIDSummary, refresh func()) {
	amount := ui.NewAmountEntry("amount (IOTA nanos)", false)
	form := widget.NewForm(
		widget.NewFormItem("CID", widget.NewLabel(components.Short(sum.CIDStr))),
		widget.NewFormItem("Amount", amount),
	)
	d := dialog.NewCustomConfirm("Add funds", "Deposit", "Cancel", form,
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
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				digest, err := service.CIDAddFunds(ctx, vc.Snapshot(), sum.ID, amt, vc.Output)
				if err != nil {
					fyne.Do(func() { dialog.ShowError(err, vc.Window) })
					return
				}
				if digest != "" && vc.OnDigest != nil {
					vc.OnDigest(digest)
				}
				refresh()
			}()
		}, vc.Window)
	d.Resize(fyne.NewSize(520, 240))
	d.Show()
}

func openNextEpochDialog(vc *ui.ViewContext, sum service.CIDSummary, refresh func()) {
	dialog.ShowConfirm("Transition to next epoch",
		fmt.Sprintf("Move CID %s into its next epoch?", components.Short(sum.CIDStr)),
		func(ok bool) {
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				if err := service.CIDTransitionEpoch(ctx, vc.Snapshot(), sum.ID, vc.Output); err != nil {
					fyne.Do(func() { dialog.ShowError(err, vc.Window) })
					return
				}
				refresh()
			}()
		}, vc.Window)
}

func openRemoveDialog(vc *ui.ViewContext, sum service.CIDSummary, refresh func()) {
	dialog.ShowConfirm("Remove CID",
		fmt.Sprintf("Remove %s from the CID list? This cannot be undone.", components.Short(sum.CIDStr)),
		func(ok bool) {
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				digest, err := service.CIDRemove(ctx, vc.Snapshot(), sum.ID, vc.Output)
				if err != nil {
					fyne.Do(func() { dialog.ShowError(err, vc.Window) })
					return
				}
				if digest != "" && vc.OnDigest != nil {
					vc.OnDigest(digest)
				}
				refresh()
			}()
		}, vc.Window)
}

// openOffersDialog shows all three offer arrays for a CID, with
// Approve / Honor row actions where appropriate.
func openOffersDialog(vc *ui.ViewContext, sum service.CIDSummary, refresh func()) {
	body := container.NewVBox(widget.NewLabel("Loading offers…"))
	scroll := container.NewVScroll(body)

	var d dialog.Dialog
	loadFn := func() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			next, current, prev, err := service.OffersForCID(ctx, vc.Snapshot(), sum.ID)
			fyne.Do(func() {
				body.RemoveAll()
				if err != nil {
					body.Add(widget.NewLabel("Error: " + err.Error()))
					body.Refresh()
					return
				}
				body.Add(widget.NewLabelWithStyle("Next epoch (approve)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				body.Add(buildOffersSection(vc, sum, next, "approve", refresh, func() {
					if d != nil {
						d.Hide()
					}
					openOffersDialog(vc, sum, refresh)
				}))
				body.Add(widget.NewSeparator())
				body.Add(widget.NewLabelWithStyle("Current epoch (honor)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				body.Add(buildOffersSection(vc, sum, current, "honor", refresh, func() {
					if d != nil {
						d.Hide()
					}
					openOffersDialog(vc, sum, refresh)
				}))
				body.Add(widget.NewSeparator())
				body.Add(widget.NewLabelWithStyle("Previous epoch (history)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				body.Add(buildOffersSection(vc, sum, prev, "", refresh, nil))
				body.Refresh()
			})
		}()
	}

	d = dialog.NewCustom("Offers — "+components.Short(sum.CIDStr), "Close", scroll, vc.Window)
	d.Resize(fyne.NewSize(720, 600))
	d.Show()
	loadFn()
}

func buildOffersSection(vc *ui.ViewContext, sum service.CIDSummary,
	rows []service.OfferRow, mode string, refresh, reload func()) fyne.CanvasObject {
	if len(rows) == 0 {
		l := widget.NewLabel("  (none)")
		l.TextStyle = fyne.TextStyle{Italic: true}
		return l
	}
	out := container.NewVBox()
	for _, o := range rows {
		oCopy := o
		stateLbl := widget.NewLabel(offerStateLabel(o))
		action := buildOfferAction(vc, sum, oCopy, mode, refresh, reload)
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
	mode string, refresh, reload func()) fyne.CanvasObject {
	switch mode {
	case "approve":
		if o.Approved {
			return widget.NewLabel("approved")
		}
		btn := widget.NewButton("Approve", func() {
			runOfferIndex(vc, sum, o, "approve", refresh, reload)
		})
		btn.Importance = widget.HighImportance
		return btn
	case "honor":
		if o.Honored {
			return widget.NewLabel("honored")
		}
		if !o.Approved {
			return widget.NewLabel("not approved")
		}
		btn := widget.NewButton("Honor", func() {
			runOfferIndex(vc, sum, o, "honor", refresh, reload)
		})
		btn.Importance = widget.HighImportance
		return btn
	}
	return widget.NewLabel("—")
}

func runOfferIndex(vc *ui.ViewContext, sum service.CIDSummary, o service.OfferRow,
	mode string, refresh, reload func()) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		var (
			digest string
			err    error
		)
		switch mode {
		case "approve":
			digest, err = service.ApproveOffer(ctx, vc.Snapshot(), service.OfferIndexForm{
				CIDObjectID: sum.ID,
				Index:       uint64(o.Index),
			}, vc.Output)
		case "honor":
			digest, err = service.HonorOffer(ctx, vc.Snapshot(), service.OfferIndexForm{
				CIDObjectID: sum.ID,
				Index:       uint64(o.Index),
			}, vc.Output)
		}
		if err != nil {
			fyne.Do(func() { dialog.ShowError(err, vc.Window) })
			return
		}
		if digest != "" && vc.OnDigest != nil {
			vc.OnDigest(digest)
		}
		refresh()
		if reload != nil {
			reload()
		}
	}()
}

// ---- tools disclosure -----------------------------------------------------

// buildTools collapses the legacy utility actions (IPFS upload, account
// new, ping, etc.) into a single accordion at the bottom of the
// dashboard so they remain accessible without dominating the layout.
func buildTools(vc *ui.ViewContext) fyne.CanvasObject {
	makeTool := func(label string, builder ui.View) *container.TabItem {
		return container.NewTabItem(label, container.NewVScroll(builder(vc)))
	}
	tabs := container.NewAppTabs(
		makeTool("Ping", PingView),
		makeTool("New account", AccountNewView),
		makeTool("List coins", AccountCoinsView),
		makeTool("IPFS upload", IPFSLoadView),
		makeTool("IPFS check CID", IPFSCheckCIDView),
		makeTool("IPFS check pins", IPFSCheckPinsView),
		makeTool("CID is in list?", CIDIsInListView),
		makeTool("Whitelist has?", WhitelistHasView),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	item := widget.NewAccordionItem("Tools  ·  utilities", tabs)
	return widget.NewAccordion(item)
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
