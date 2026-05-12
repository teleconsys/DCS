// Package provider hosts the Provider workspace's action views.
package provider

import "github.com/teleconsys/DCS/cmd/gui/ui"

// All returns the Provider action views in display order.
func All() []ui.NamedView {
	return []ui.NamedView{
		{Name: "Ping", Builder: PingView},
		{Name: "Account: new", Builder: AccountNewView},
		{Name: "Account: coins", Builder: AccountCoinsView},
		{Name: "Open offer windows", Builder: ListOpenOffersView},
		{Name: "Submit offer", Builder: SubmitOfferView},
		{Name: "Withdraw", Builder: WithdrawView},
		{Name: "Whitelist: has", Builder: WhitelistHasView},
	}
}
