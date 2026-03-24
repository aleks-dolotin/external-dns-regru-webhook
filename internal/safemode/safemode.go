// Package safemode provides a thread-safe toggle that prevents writes to Reg.ru.
// When safe-mode is enabled, the worker pool suppresses dispatch calls and logs
// the suppression. Operations remain in the queue for later processing.
//
// Story 8.3: Safe-mode (no-write) toggle.
package safemode

import (
	"sync"
	"sync/atomic"
	"time"
)

// SafeMode is a thread-safe toggle for preventing writes.
type SafeMode struct {
	mu              sync.RWMutex
	enabled         bool
	since           time.Time
	suppressedCount int64 // atomic
}

// Status is the JSON-serializable state of safe-mode.
type Status struct {
	Enabled         bool       `json:"enabled"`
	Since           *time.Time `json:"since,omitempty"`
	SuppressedCount int64      `json:"suppressed_count"`
}

// New creates a new SafeMode (disabled by default).
func New() *SafeMode {
	return &SafeMode{}
}

// Enable activates safe-mode.
func (s *SafeMode) Enable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled {
		s.enabled = true
		s.since = time.Now().UTC()
		atomic.StoreInt64(&s.suppressedCount, 0)
	}
}

// Disable deactivates safe-mode.
func (s *SafeMode) Disable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = false
	s.since = time.Time{}
}

// IsEnabled returns true if safe-mode is currently active.
func (s *SafeMode) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// IncrementSuppressed increments the suppressed operation counter. Thread-safe.
func (s *SafeMode) IncrementSuppressed() {
	atomic.AddInt64(&s.suppressedCount, 1)
}

// Status returns the current safe-mode state.
func (s *SafeMode) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := Status{
		Enabled:         s.enabled,
		SuppressedCount: atomic.LoadInt64(&s.suppressedCount),
	}
	if s.enabled && !s.since.IsZero() {
		t := s.since
		st.Since = &t
	}
	return st
}
