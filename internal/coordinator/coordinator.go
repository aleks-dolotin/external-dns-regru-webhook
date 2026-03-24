// Package coordinator provides an optional central coordination mode for
// global rate limit caps (Story 5.3, P2 — Growth phase).
//
// When coordinator is unavailable, sidecars operate standalone using local
// per-zone rate limiters (graceful degradation per ADR-002).
//
// This package defines the interface and a no-op stub; full implementation
// is deferred to Growth phase.
package coordinator

import (
	"context"
	"time"
)

// Coordinator defines the interface for a central quota coordinator.
// Sidecars call RequestQuota before making API calls; the coordinator
// tracks global usage and grants/denies tokens.
type Coordinator interface {
	// RequestQuota asks for a token for the given zone.
	// Returns nil if granted, error if denied or coordinator unavailable.
	RequestQuota(ctx context.Context, zone string) error

	// ReleaseQuota returns an unused token (optional, best-effort).
	ReleaseQuota(zone string)

	// Available returns true if the coordinator is reachable.
	Available() bool
}

// Config holds coordinator configuration.
type Config struct {
	Enabled       bool          // whether central coordination is active
	Endpoint      string        // coordinator service URL (HTTP or gRPC)
	Timeout       time.Duration // per-request timeout; default 500ms
	FallbackLocal bool          // if true, fall back to local limiter when unavailable (default: true)
}

// DefaultConfig returns the default coordinator configuration (disabled).
func DefaultConfig() Config {
	return Config{
		Enabled:       false,
		Timeout:       500 * time.Millisecond,
		FallbackLocal: true,
	}
}

// NoopCoordinator is a stub that always grants quota.
// Used when central coordination is disabled (default/MVP).
type NoopCoordinator struct{}

// RequestQuota always returns nil (quota granted).
func (n *NoopCoordinator) RequestQuota(_ context.Context, _ string) error { return nil }

// ReleaseQuota is a no-op.
func (n *NoopCoordinator) ReleaseQuota(_ string) {}

// Available always returns false (no coordinator configured).
func (n *NoopCoordinator) Available() bool { return false }
