// Package gc holds the Ground Control workspace's action views.
// All views in this package assume the active actor is GC; views that
// require on-chain signing belong elsewhere.
package gc

import "github.com/teleconsys/DCS/cmd/gui/ui"

// All returns the GC action views in the order they should appear in
// the workspace sidebar.
func All() []ui.NamedView {
	return []ui.NamedView{
		{Name: "Ping", Builder: PingView},
		{Name: "Whitelist: has", Builder: WhitelistHasView},
		{Name: "Whitelist: add", Builder: WhitelistAddView},
		{Name: "Whitelist: remove", Builder: WhitelistRemoveView},
	}
}
