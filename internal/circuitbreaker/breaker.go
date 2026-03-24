// Package circuitbreaker provides per-zone circuit breaker logic (Story 5.4, ADR-005).
//
// State machine: closed → open → half-open → closed.
// Each zone maintains independent state. When a zone's error rate exceeds the
// configured threshold within a sliding window the circuit opens, preventing
// further writes until a cooldown expires and probe requests succeed.
package circuitbreaker

import (
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	StateClosed   State = 0
	StateHalfOpen State = 1
	StateOpen     State = 2
)

// String returns a human-readable state name.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit is open and requests are rejected.
var ErrCircuitOpen = fmt.Errorf("circuitbreaker: circuit is open — writes suspended for zone")

// Config holds circuit breaker configuration per ADR-005.
type Config struct {
	ErrorThreshold float64       // error rate threshold (0.0–1.0); default 0.5
	Window         time.Duration // sliding window for error rate calculation; default 60s
	CooldownPeriod time.Duration // how long circuit stays open; default 5min
	HalfOpenProbes int           // successful probes needed to close; default 3
}

// DefaultConfig returns the default circuit breaker configuration per ADR-005.
func DefaultConfig() Config {
	return Config{
		ErrorThreshold: 0.50,
		Window:         60 * time.Second,
		CooldownPeriod: 5 * time.Minute,
		HalfOpenProbes: 3,
	}
}

// breaker holds state for a single zone's circuit breaker.
type breaker struct {
	mu             sync.RWMutex
	state          State
	successes      int       // within window (closed) or probe successes (half-open)
	failures       int       // within window
	windowStart    time.Time // start of current measurement window
	openedAt       time.Time // when circuit was opened
	halfOpenProbes int       // successful probes counted in half-open
}

// Manager manages per-zone circuit breakers.
type Manager struct {
	mu       sync.RWMutex
	breakers map[string]*breaker
	cfg      Config
	now      func() time.Time // injectable clock for testing

	// OnStateChange is called when any zone's circuit state changes.
	// Signature: func(zone string, from State, to State)
	OnStateChange func(zone string, from, to State)
}

// NewManager creates a circuit breaker manager with the given configuration.
func NewManager(cfg Config) *Manager {
	if cfg.ErrorThreshold <= 0 {
		cfg.ErrorThreshold = DefaultConfig().ErrorThreshold
	}
	if cfg.Window <= 0 {
		cfg.Window = DefaultConfig().Window
	}
	if cfg.CooldownPeriod <= 0 {
		cfg.CooldownPeriod = DefaultConfig().CooldownPeriod
	}
	if cfg.HalfOpenProbes <= 0 {
		cfg.HalfOpenProbes = DefaultConfig().HalfOpenProbes
	}
	return &Manager{
		breakers: make(map[string]*breaker),
		cfg:      cfg,
		now:      time.Now,
	}
}

// getBreaker returns the breaker for a zone, creating one if absent.
func (m *Manager) getBreaker(zone string) *breaker {
	m.mu.RLock()
	b, ok := m.breakers[zone]
	m.mu.RUnlock()
	if ok {
		return b
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok = m.breakers[zone]; ok {
		return b
	}
	b = &breaker{
		state:       StateClosed,
		windowStart: m.now(),
	}
	m.breakers[zone] = b
	return b
}

// AllowRequest checks if a request to the given zone is allowed.
// Returns nil if allowed, ErrCircuitOpen if the circuit is open.
func (m *Manager) AllowRequest(zone string) error {
	b := m.getBreaker(zone)
	b.mu.Lock()
	defer b.mu.Unlock()

	now := m.now()

	switch b.state {
	case StateClosed:
		return nil
	case StateOpen:
		// Check if cooldown has elapsed → transition to half-open.
		if now.Sub(b.openedAt) >= m.cfg.CooldownPeriod {
			m.transition(b, zone, StateHalfOpen)
			return nil // allow probe request
		}
		return ErrCircuitOpen
	case StateHalfOpen:
		// Allow probe requests in half-open state.
		return nil
	}
	return nil
}

// RecordSuccess records a successful operation for the given zone.
func (m *Manager) RecordSuccess(zone string) {
	b := m.getBreaker(zone)
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		b.successes++
		m.maybeResetWindow(b)
	case StateHalfOpen:
		b.halfOpenProbes++
		if b.halfOpenProbes >= m.cfg.HalfOpenProbes {
			m.transition(b, zone, StateClosed)
		}
	case StateOpen:
		// Ignore successes while open (shouldn't happen).
	}
}

// RecordFailure records a failed operation for the given zone.
func (m *Manager) RecordFailure(zone string) {
	b := m.getBreaker(zone)
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		b.failures++
		m.maybeResetWindow(b)
		// Check error rate threshold.
		total := b.successes + b.failures
		if total > 0 {
			errRate := float64(b.failures) / float64(total)
			if errRate >= m.cfg.ErrorThreshold && total >= 5 {
				m.transition(b, zone, StateOpen)
			}
		}
	case StateHalfOpen:
		// Any failure in half-open → back to open.
		m.transition(b, zone, StateOpen)
	case StateOpen:
		// Already open, ignore.
	}
}

// GetState returns the current circuit breaker state for a zone.
func (m *Manager) GetState(zone string) State {
	b := m.getBreaker(zone)
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// ZoneStates returns a map of all zones to their current circuit state.
func (m *Manager) ZoneStates() map[string]State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	states := make(map[string]State, len(m.breakers))
	for zone, b := range m.breakers {
		b.mu.RLock()
		states[zone] = b.state
		b.mu.RUnlock()
	}
	return states
}

// transition changes the breaker's state and calls OnStateChange.
// Must be called with b.mu held.
func (m *Manager) transition(b *breaker, zone string, to State) {
	from := b.state
	b.state = to
	now := m.now()

	switch to {
	case StateClosed:
		b.successes = 0
		b.failures = 0
		b.halfOpenProbes = 0
		b.windowStart = now
	case StateOpen:
		b.openedAt = now
		b.halfOpenProbes = 0
	case StateHalfOpen:
		b.halfOpenProbes = 0
	}

	if m.OnStateChange != nil && from != to {
		m.OnStateChange(zone, from, to)
	}
}

// maybeResetWindow resets the sliding window counters if the window has elapsed.
// Must be called with b.mu held.
func (m *Manager) maybeResetWindow(b *breaker) {
	now := m.now()
	if now.Sub(b.windowStart) >= m.cfg.Window {
		b.successes = 0
		b.failures = 0
		b.windowStart = now
	}
}

// ForceClose forces a zone's circuit to the closed state (admin operation).
func (m *Manager) ForceClose(zone string) {
	b := m.getBreaker(zone)
	b.mu.Lock()
	defer b.mu.Unlock()
	m.transition(b, zone, StateClosed)
}
