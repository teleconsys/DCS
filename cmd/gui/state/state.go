package state

import "sync"

// AppState carries cross-actor settings (chiefly: the actor registry).
type AppState struct {
	mu sync.RWMutex

	Registry *ActorRegistry
}

// NewAppState constructs an AppState populated with a fresh registry.
func NewAppState() *AppState {
	return &AppState{
		Registry: NewRegistry(),
	}
}
