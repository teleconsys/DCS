// Package user hosts the User workspace's action views.
package user

import "github.com/teleconsys/DCS/cmd/gui/ui"

// All returns the User action views in display order.
func All() []ui.NamedView {
	return []ui.NamedView{
		{Name: "Ping", Builder: PingView},
		{Name: "Account: new", Builder: AccountNewView},
		{Name: "Account: coins", Builder: AccountCoinsView},
		{Name: "IPFS: load file", Builder: IPFSLoadView},
		{Name: "IPFS: check CID", Builder: IPFSCheckCIDView},
		{Name: "IPFS: check pins", Builder: IPFSCheckPinsView},
		{Name: "CID: create", Builder: CIDCreateView},
		{Name: "CID: add funds", Builder: CIDAddFundsView},
		{Name: "CID: remove", Builder: CIDRemoveView},
		{Name: "CID: is in list", Builder: CIDIsInListView},
		{Name: "CID: next epoch", Builder: CIDNextEpochView},
		{Name: "Approve offer", Builder: ApproveOfferView},
		{Name: "Honor offer", Builder: HonorOfferView},
		{Name: "Whitelist: has", Builder: WhitelistHasView},
	}
}
