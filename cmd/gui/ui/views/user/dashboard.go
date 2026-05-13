package user

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/teleconsys/DCS/cmd/gui/ui/wizards"
)

// railMinUserPane is the minimum size of the right-hand scroll (IPFS | Tools split).
// Height leaves a workable area for the vertical split between the two cards.
var railMinUserPane = fyne.NewSize(400, 480)
var railMinUserPaneDemo = fyne.NewSize(176, 300)

// clampMinWLayout reports a fixed minimum width (while keeping the child's
// natural height) but always lays out the child to the full allocated size.
// This works around fyne's VScroll: MinSize width becomes max(SetMinSize,
// content.MinSize), so wide IPFS/Tools cards would otherwise force the rail
// to stay huge and block the My CIDs | rail split in narrow Demo panes.
type clampMinWLayout struct {
	MinW float32
}

func (l clampMinWLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	h := objects[0].MinSize().Height
	return fyne.NewSize(l.MinW, h)
}

func (l clampMinWLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Resize(size)
	objects[0].Move(fyne.NewPos(0, 0))
}

// userDashboardOpts tweaks inner layout while keeping the main My CIDs | rail split.
type userDashboardOpts struct {
	cidGridCols    int  // AdaptiveGrid column count for CID tiles (Demo uses 1).
	stackIPFSTools bool // when true, IPFS and Tools stack vertically instead of side-by-side.
	// compactSplitMins lowers rail and CID scroll minimum sizes so the HSplit
	// between My CIDs and the rail stays draggable when the User pane is only
	// half the window (Demo tab).
	compactSplitMins bool
}

// Dashboard is the User workspace: My CIDs on the left (Add in the card header),
// IPFS and Tools on the right in a vertical split (draggable divider), and an HSplit
// between those panes.
func Dashboard(vc *ui.ViewContext) fyne.CanvasObject {
	return buildUserDashboard(vc, userDashboardOpts{cidGridCols: 2, stackIPFSTools: true})
}

// DashboardForDemo is like Dashboard but lists CID tiles in a single column and
// uses compact split minimums — for the Demo tab while preserving the My CIDs | rail split.
func DashboardForDemo(vc *ui.ViewContext) fyne.CanvasObject {
	return buildUserDashboard(vc, userDashboardOpts{
		cidGridCols:      1,
		stackIPFSTools:   true,
		compactSplitMins: true,
	})
}

func buildUserDashboard(vc *ui.ViewContext, opts userDashboardOpts) fyne.CanvasObject {
	var cidScrollMin fyne.Size
	if opts.compactSplitMins {
		cidScrollMin = fyne.NewSize(120, 96)
	}
	grid, _ := buildMyCIDsCard(vc, opts.cidGridCols, cidScrollMin)
	ipfs := buildIPFSSection(vc)
	tools := buildTools(vc)

	var ipfsTools fyne.CanvasObject
	if opts.stackIPFSTools {
		// VSplit divides rail height; CardStretch fills each half; tab bodies use
		// ui.TopBound inside VScroll so short forms are not stretched with empty space below.
		ipfsTools = container.NewVSplit(
			container.NewMax(ipfs),
			container.NewMax(tools),
		)
	} else {
		ipfsTools = container.NewGridWithColumns(2, ipfs, tools)
	}

	sidebar := container.NewVBox(ipfsTools)
	var railScroll fyne.CanvasObject
	if opts.compactSplitMins {
		railCore := container.New(clampMinWLayout{MinW: 176}, sidebar)
		sc := container.NewVScroll(railCore)
		sc.SetMinSize(railMinUserPaneDemo)
		railScroll = sc
	} else {
		sc := container.NewVScroll(sidebar)
		sc.SetMinSize(railMinUserPane)
		railScroll = sc
	}

	if opts.compactSplitMins {
		grid = container.New(clampMinWLayout{MinW: 152}, grid)
	}

	split := container.NewHSplit(
		grid,
		container.NewPadded(railScroll),
	)
	// Offset is the fraction for the first child (My CIDs); mirror the old 0.38 rail share.
	split.SetOffset(0.62)

	return container.NewStack(split)
}

// ---- My CIDs grid ---------------------------------------------------------

func buildMyCIDsCard(vc *ui.ViewContext, gridCols int, cidScrollMin fyne.Size) (fyne.CanvasObject, func()) {
	if gridCols < 1 {
		gridCols = 1
	}
	if cidScrollMin.Width <= 0 || cidScrollMin.Height <= 0 {
		cidScrollMin = fyne.NewSize(220, 160)
	}
	// Adaptive grid reflows columns with window width; scroll keeps MinSize bounded
	// so many tiles do not force the window to grow vertically.
	grid := container.NewAdaptiveGrid(gridCols)
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
					host.Objects = []fyne.CanvasObject{cidScroll}
				}
				host.Refresh()
				statusLbl.SetText(fmt.Sprintf("%d CID(s)", len(cids)))
			})
		}()
	}

	addBtn := widget.NewButtonWithIcon("Add", ftheme.ContentAddIcon(), func() {
		wizards.NewCIDWizard(vc, refresh)
	})
	addBtn.Importance = widget.HighImportance

	refreshBtn := widget.NewButtonWithIcon("Refresh", ftheme.ViewRefreshIcon(), refresh)
	headerRight := container.NewHBox(addBtn, refreshBtn)
	header := container.NewBorder(nil, nil, statusLbl, headerRight)
	body := container.NewBorder(header, nil, nil, nil, host)

	refresh()

	return components.CardStretch(theme.AccentUser.Primary, "My CIDs", "", body), refresh
}

func buildCIDTile(vc *ui.ViewContext, sum service.CIDSummary, refresh func()) fyne.CanvasObject {
	state := badgeStateForCID(sum.Status)
	badge := components.Badge(state, sum.Status.String())

	cidLbl := widget.NewLabelWithStyle(components.Short(sum.CIDStr), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	now := time.Now().UnixMilli()
	winLbl := widget.NewLabel(windowText(sum, now))
	winLbl.Wrapping = fyne.TextWrapWord
	var balReadable string
	if sum.Balance >= 0 {
		balReadable = formatIOTA(uint64(sum.Balance))
	} else {
		balReadable = strconv.FormatInt(sum.Balance, 10)
	}
	statsLbl := widget.NewLabel(fmt.Sprintf("%s IOTA · offers %d/%d/%d",
		balReadable, sum.NextOffers, sum.CurrentOffers, sum.PrevOffers))

	body := container.NewVBox(badge, cidLbl, winLbl, statsLbl)

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
		return "No epoch data"
	}
	switch sum.Status {
	case service.CIDStatusPending:
		mins := (sum.CurrentEpochEnd - nowMs) / 60_000
		return fmt.Sprintf("%d min left (active)", mins)
	case service.CIDStatusOfferWindow:
		mins := (sum.NextEpochStart - nowMs) / 60_000
		return fmt.Sprintf("Offers · next epoch in %d min", mins)
	case service.CIDStatusNextScheduled:
		mins := (sum.NextEpochEnd - nowMs) / 60_000
		return fmt.Sprintf("Next epoch · %d min left", mins)
	case service.CIDStatusExpired:
		return "Expired"
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
				_, err := service.CIDAddFunds(ctx, vc.Snapshot(), sum.ID, amt, vc.Output)
				if err != nil {
					fyne.Do(func() { dialog.ShowError(err, vc.Window) })
					return
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
				_, err := service.CIDRemove(ctx, vc.Snapshot(), sum.ID, vc.Output)
				if err != nil {
					fyne.Do(func() { dialog.ShowError(err, vc.Window) })
					return
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
		var err error
		switch mode {
		case "approve":
			_, err = service.ApproveOffer(ctx, vc.Snapshot(), service.OfferIndexForm{
				CIDObjectID: sum.ID,
				Index:       uint64(o.Index),
			}, vc.Output)
		case "honor":
			_, err = service.HonorOffer(ctx, vc.Snapshot(), service.OfferIndexForm{
				CIDObjectID: sum.ID,
				Index:       uint64(o.Index),
			}, vc.Output)
		}
		if err != nil {
			fyne.Do(func() { dialog.ShowError(err, vc.Window) })
			return
		}
		refresh()
		if reload != nil {
			reload()
		}
	}()
}

// ---- IPFS section --------------------------------------------------------

func buildIPFSSection(vc *ui.ViewContext) fyne.CanvasObject {
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
	return components.CardStretch(theme.AccentUser.Primary, "IPFS", "", tabs)
}

// ---- tools section --------------------------------------------------------

func buildTools(vc *ui.ViewContext) fyne.CanvasObject {
	makeTab := func(label string, builder ui.View) *container.TabItem {
		body := container.NewVScroll(ui.TopBound(builder(vc)))
		return container.NewTabItem(label, body)
	}
	tabs := container.NewAppTabs(
		makeTab("Ping", PingView),
		makeTab("New account", AccountNewView),
		makeTab("List coins", AccountCoinsView),
		makeTab("CID is in list?", CIDIsInListView),
		makeTab("Whitelist has?", WhitelistHasView),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return components.CardStretch(theme.AccentUser.Primary, "Tools", "", tabs)
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
