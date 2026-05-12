// Package state holds shared, UI-agnostic types used across the GUI:
// the three DCS actors (Ground Control, User, Provider), their identity
// profiles, and the overall application state.
//
// Keeping these types outside of the `ui` and `service` packages avoids
// import cycles: both packages depend on `state`, never on each other.
package state

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Actor enumerates the roles defined in testsheet.md § 2.
type Actor int

const (
	ActorGC Actor = iota
	ActorUser
	ActorProvider
)

// String returns the human-readable actor name (also used as the role tag
// persisted in accounts/<alias>.json).
func (a Actor) String() string {
	switch a {
	case ActorGC:
		return "Ground Control"
	case ActorUser:
		return "User"
	case ActorProvider:
		return "Provider"
	default:
		return fmt.Sprintf("Actor(%d)", int(a))
	}
}

// RoleTag returns the short form ("gc", "user", "provider") suited for
// machine-readable storage (accounts/<alias>.json "role" field).
func (a Actor) RoleTag() string {
	switch a {
	case ActorGC:
		return "gc"
	case ActorUser:
		return "user"
	case ActorProvider:
		return "provider"
	default:
		return ""
	}
}

// ActorProfile is the immutable snapshot of an actor's identity and
// network configuration. Service functions accept a value (not a pointer)
// so that an in-flight operation always runs against the values that were
// active when it was launched, even if the user switches actors or edits
// fields mid-flight.
type ActorProfile struct {
	Actor Actor

	// Optional alias (matches a file in ./accounts/<alias>.json).
	Alias string

	// Signing identity. Empty for GC.
	PrivateKey string
	Address    string
	GasCoinID  string

	// Ground Control HTTP credentials. Empty for User/Provider.
	GCEndpoint string
	GCToken    string

	// Common network configuration (defaults loaded from .env).
	RPCURL          string
	GraphQLEndpoint string
	PackageID       string
	WhitelistID     string
	CIDListID       string
	ClockID         string
	GasBudget       uint64
	FaucetURL       string
}

// Clone returns a shallow copy. Use this to snapshot a profile before
// kicking off a long-running goroutine.
func (p ActorProfile) Clone() ActorProfile { return p }

// ActorRegistry holds the three actor profiles and the currently-selected
// actor. It is the single source of truth for which identity any signed
// action should use.
type ActorRegistry struct {
	profiles map[Actor]*ActorProfile
	current  Actor
}

// NewRegistry constructs a registry populated with defaults from the
// current process environment. Callers should invoke this after the
// .env file has been loaded (see internal/config.LoadEnv).
func NewRegistry() *ActorRegistry {
	r := &ActorRegistry{
		profiles: make(map[Actor]*ActorProfile, 3),
		current:  ActorUser,
	}
	r.profiles[ActorGC] = defaultGCProfile()
	r.profiles[ActorUser] = defaultUserProfile()
	r.profiles[ActorProvider] = defaultProviderProfile()
	return r
}

// Profile returns the mutable profile for the given actor.
// The pointer remains stable for the lifetime of the registry, so widgets
// may bind to fields directly.
func (r *ActorRegistry) Profile(a Actor) *ActorProfile {
	if p, ok := r.profiles[a]; ok {
		return p
	}
	p := &ActorProfile{Actor: a}
	r.profiles[a] = p
	return p
}

// Snapshot returns an immutable copy suitable for handing to a goroutine.
func (r *ActorRegistry) Snapshot(a Actor) ActorProfile {
	return r.Profile(a).Clone()
}

// Current returns the actor that is currently selected.
func (r *ActorRegistry) Current() Actor { return r.current }

// SetCurrent updates the currently-selected actor.
func (r *ActorRegistry) SetCurrent(a Actor) { r.current = a }

// defaultRPC and defaultGraphQL match the fallbacks used by the CLI when
// no .env values are present.
const (
	defaultRPC     = "https://api.testnet.iota.cafe:443"
	defaultGraphQL = "https://graphql.testnet.iota.cafe"
)

func defaultGCProfile() *ActorProfile {
	return &ActorProfile{
		Actor:           ActorGC,
		GCEndpoint:      envFirst("GC_ENDPOINT"),
		GCToken:         envFirst("GC_API_TOKEN"),
		RPCURL:          envFirstDefault(defaultRPC, "REBASE_RPC", "DCS_RPC"),
		GraphQLEndpoint: envFirstDefault(defaultGraphQL, "IOTA_GRAPHQL_ENDPOINT"),
		PackageID:       envFirst("DCS_PACKAGE_ID"),
		WhitelistID:     envFirst("DCS_WHITELIST_ID"),
		CIDListID:       envFirst("DCS_CIDLIST_ID"),
		ClockID:         envFirstDefault("0x6", "DCS_CLOCK_ID"),
		GasBudget:       envUint("WALLET_GAS_BUDGET", 10_000_000),
		FaucetURL:       envFirst("FAUCET_URL"),
	}
}

func defaultUserProfile() *ActorProfile {
	return &ActorProfile{
		Actor:           ActorUser,
		PrivateKey:      envFirst("ACTIVE_PRIVATE_KEY", "USER_PRIVATE_KEY"),
		Address:         envFirst("ACTIVE_ADDRESS", "USER_ADDRESS"),
		GasCoinID:       envFirst("ACTIVE_GAS_COIN_ID", "USER_GAS_COIN_ID"),
		RPCURL:          envFirstDefault(defaultRPC, "REBASE_RPC", "DCS_RPC"),
		GraphQLEndpoint: envFirstDefault(defaultGraphQL, "IOTA_GRAPHQL_ENDPOINT"),
		PackageID:       envFirst("DCS_PACKAGE_ID"),
		WhitelistID:     envFirst("DCS_WHITELIST_ID"),
		CIDListID:       envFirst("DCS_CIDLIST_ID"),
		ClockID:         envFirstDefault("0x6", "DCS_CLOCK_ID"),
		GasBudget:       envUint("WALLET_GAS_BUDGET", 10_000_000),
		FaucetURL:       envFirst("FAUCET_URL"),
	}
}

func defaultProviderProfile() *ActorProfile {
	return &ActorProfile{
		Actor:           ActorProvider,
		PrivateKey:      envFirst("PROVIDER_PRIVATE_KEY"),
		Address:         envFirst("PROVIDER_ADDRESS"),
		GasCoinID:       envFirst("PROVIDER_GAS_COIN_ID"),
		RPCURL:          envFirstDefault(defaultRPC, "REBASE_RPC", "DCS_RPC"),
		GraphQLEndpoint: envFirstDefault(defaultGraphQL, "IOTA_GRAPHQL_ENDPOINT"),
		PackageID:       envFirst("DCS_PACKAGE_ID"),
		WhitelistID:     envFirst("DCS_WHITELIST_ID"),
		CIDListID:       envFirst("DCS_CIDLIST_ID"),
		ClockID:         envFirstDefault("0x6", "DCS_CLOCK_ID"),
		GasBudget:       envUint("WALLET_GAS_BUDGET", 10_000_000),
		FaucetURL:       envFirst("FAUCET_URL"),
	}
}

func envFirst(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func envFirstDefault(def string, keys ...string) string {
	if v := envFirst(keys...); v != "" {
		return v
	}
	return def
}

func envUint(key string, def uint64) uint64 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}
