// Package quota provides per-namespace request quota enforcement with
// fixed-window hourly counters.
//
// Story 9.2: Per-namespace quotas and alerts.
package quota

import (
	"fmt"
	"sync"
	"time"
)

// NamespaceQuota defines the quota configuration for a single namespace.
type NamespaceQuota struct {
	Namespace    string
	LimitPerHour int
}

// counter tracks usage within a fixed hourly window.
type counter struct {
	used      int
	limit     int
	windowEnd time.Time
}

// Manager enforces per-namespace request quotas using fixed hourly windows.
// Namespaces without an explicit quota have no per-namespace rate limiting.
type Manager struct {
	mu       sync.Mutex
	counters map[string]*counter // key: namespace
	now      func() time.Time    // injectable clock for testing
}

// New creates a Manager with the given per-namespace quotas.
// Namespaces not present in quotas have no limit applied.
func New(quotas []NamespaceQuota) *Manager {
	m := &Manager{
		counters: make(map[string]*counter, len(quotas)),
		now:      time.Now,
	}
	for _, q := range quotas {
		if q.LimitPerHour > 0 {
			m.counters[q.Namespace] = &counter{
				limit:     q.LimitPerHour,
				windowEnd: nextWindowEnd(time.Now()),
			}
		}
	}
	return m
}

// AllowRequest returns true if the namespace has budget remaining (or no quota).
// If quota is exceeded, returns false.
func (m *Manager) AllowRequest(namespace string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.counters[namespace]
	if !ok {
		// No quota configured for this namespace — allow.
		return true
	}

	now := m.now()
	// Reset window if expired.
	if now.After(c.windowEnd) || now.Equal(c.windowEnd) {
		c.used = 0
		c.windowEnd = nextWindowEnd(now)
	}

	if c.used >= c.limit {
		return false
	}
	c.used++
	return true
}

// CurrentUsage returns (used, limit) for a namespace.
// If no quota is configured, returns (0, 0).
func (m *Manager) CurrentUsage(namespace string) (used int, limit int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.counters[namespace]
	if !ok {
		return 0, 0
	}

	now := m.now()
	if now.After(c.windowEnd) || now.Equal(c.windowEnd) {
		c.used = 0
		c.windowEnd = nextWindowEnd(now)
	}

	return c.used, c.limit
}

// UpdateQuotas replaces the quota configuration. Existing counters for
// namespaces that still have quotas retain their current window usage.
// Story 9.2: supports config reload without restart.
func (m *Manager) UpdateQuotas(quotas []NamespaceQuota) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newCounters := make(map[string]*counter, len(quotas))
	now := m.now()

	for _, q := range quotas {
		if q.LimitPerHour <= 0 {
			continue
		}
		if existing, ok := m.counters[q.Namespace]; ok {
			// Retain usage, update limit.
			existing.limit = q.LimitPerHour
			newCounters[q.Namespace] = existing
		} else {
			newCounters[q.Namespace] = &counter{
				limit:     q.LimitPerHour,
				windowEnd: nextWindowEnd(now),
			}
		}
	}
	m.counters = newCounters
}

// RejectionMessage returns a formatted error message for quota exceeded.
func RejectionMessage(namespace string, limit int) string {
	return fmt.Sprintf("namespace %s: quota exceeded (limit: %d/hr)", namespace, limit)
}

// nextWindowEnd returns the end of the current fixed hourly window.
func nextWindowEnd(now time.Time) time.Time {
	return now.Truncate(time.Hour).Add(time.Hour)
}
