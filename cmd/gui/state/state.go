package state

import "sync"

// AppState carries cross-actor settings (chiefly: the actor registry and
// the last transaction digest seen by any view) so the status bar can
// render a global view.
type AppState struct {
	mu sync.RWMutex

	Registry *ActorRegistry

	lastDigest map[Actor]string
}

// NewAppState constructs an AppState populated with a fresh registry.
func NewAppState() *AppState {
	return &AppState{
		Registry:   NewRegistry(),
		lastDigest: map[Actor]string{},
	}
}

// SetLastDigest records the most recent on-chain transaction digest for
// the given actor.
func (s *AppState) SetLastDigest(a Actor, digest string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastDigest[a] = digest
}

// LastDigest returns the most recent transaction digest for the given
// actor, or the empty string.
func (s *AppState) LastDigest(a Actor) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastDigest[a]
}
