package provider

import (
	"context"
	"fmt"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui"
)

// ListOpenOffersView lists CIDs whose offer window is currently open.
func ListOpenOffersView(vc *ui.ViewContext) fyne.CanvasObject {
	desc := widget.NewLabel("Queries the GraphQL endpoint for CIDs whose offer window is currently open.")
	desc.Wrapping = fyne.TextWrapWord

	run := ui.RunButton(vc, "Refresh", "list-open-offers", nil,
		func(ctx context.Context, out io.Writer, snap state.ActorProfile) (string, error) {
			items, err := service.ListOpenOffers(ctx, snap, out)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Found %d open offer window(s).", len(items)), nil
		})
	return container.NewVBox(ui.Card("Open offer windows", desc), run)
}
