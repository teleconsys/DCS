package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/gc"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/provider"
	"github.com/teleconsys/DCS/cmd/gui/ui/views/user"
)

// viewsFor returns the available action views for the given actor.
func viewsFor(actor state.Actor) []ui.NamedView {
	switch actor {
	case state.ActorGC:
		return gc.All()
	case state.ActorUser:
		return user.All()
	case state.ActorProvider:
		return provider.All()
	default:
		return nil
	}
}

// workspace holds the live widgets making up a single actor's workspace.
type workspace struct {
	actor   state.Actor
	profile *state.ActorProfile
	output  *ui.OutputView
	runner  *service.Runner
	identity *ui.IdentityPanel

	// status fields managed by this workspace (used by the parent status
	// bar to render an actor summary)
	digestLabel *widget.Label

	canvas fyne.CanvasObject
}

// buildWorkspace assembles the identity panel + action sidebar + content
// + output log for a single actor. The returned workspace is the
// authoritative container that owns its profile and runner.
func buildWorkspace(win fyne.Window, appState *state.AppState, actor state.Actor) *workspace {
	w := &workspace{
		actor:       actor,
		profile:     appState.Registry.Profile(actor),
		output:      ui.NewOutputView(fmt.Sprintf("%s log", actor)),
		runner:      service.NewRunner(),
		digestLabel: widget.NewLabel(""),
	}

	// Status banner for the actor (address + last digest).
	header := widget.NewLabelWithStyle(
		actorBanner(w.profile),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	refreshHeader := func() {
		header.SetText(actorBanner(w.profile))
	}

	// Identity panel + status bar
	w.identity = ui.NewIdentityPanel(win, w.profile, refreshHeader)
	identityCard := container.NewVBox(
		w.identity.CanvasObject(),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Last digest", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		w.digestLabel,
	)

	// Content + sidebar
	content := container.NewMax(widget.NewLabel("Select an action on the left."))
	views := viewsFor(actor)
	labels := make([]string, len(views))
	for i, v := range views {
		labels[i] = v.Name
	}

	selectAction := func(idx int) {
		if idx < 0 || idx >= len(views) {
			return
		}
		vc := &ui.ViewContext{
			Window:  win,
			Output:  w.output,
			Runner:  w.runner,
			Profile: w.profile,
			App:     appState,
			OnDigest: func(digest string) {
				appState.SetLastDigest(actor, digest)
				fyne.Do(func() {
					w.digestLabel.SetText(digest)
					refreshHeader()
				})
			},
		}
		obj := views[idx].Builder(vc)
		content.Objects = []fyne.CanvasObject{container.NewVScroll(obj)}
		content.Refresh()
	}

	list := widget.NewList(
		func() int { return len(views) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(labels[i])
		},
	)
	list.OnSelected = func(i widget.ListItemID) { selectAction(i) }
	if len(views) > 0 {
		list.Select(0)
	}

	leftPanel := container.NewBorder(header, nil, nil, nil,
		container.NewVScroll(identityCard))
	leftPanel.Resize(fyne.NewSize(380, 0))

	sidebarWithList := container.NewBorder(
		widget.NewLabelWithStyle("Actions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil, list,
	)

	center := container.NewBorder(nil, nil, nil, nil, content)
	right := w.output.CanvasObject()

	// Three-column layout: identity | center (sidebar+content) | output
	innerSplit := container.NewHSplit(sidebarWithList, center)
	innerSplit.SetOffset(0.30)

	mainSplit := container.NewHSplit(leftPanel, innerSplit)
	mainSplit.SetOffset(0.25)

	outerSplit := container.NewVSplit(mainSplit, right)
	outerSplit.SetOffset(0.65)

	w.canvas = outerSplit
	return w
}

func actorBanner(p *state.ActorProfile) string {
	switch p.Actor {
	case state.ActorGC:
		ep := strings.TrimSpace(p.GCEndpoint)
		if ep == "" {
			ep = "(no endpoint)"
		}
		return fmt.Sprintf("Ground Control · %s", ep)
	default:
		addr := strings.TrimSpace(p.Address)
		if addr == "" {
			addr = "(no address)"
		}
		return fmt.Sprintf("%s · %s", p.Actor, addr)
	}
}
