// Package ratelimit provides a per-zone token-bucket rate limiter for
// outgoing Reg.ru API requests (Story 5.1, ADR-002).
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// DefaultRatePerHour is the default per-zone rate limit when not configured.
const DefaultRatePerHour = 300

// ErrRateLimited is returned when a request is rejected by the rate limiter.
var ErrRateLimited = fmt.Errorf("ratelimit: request rejected — zone rate limit exceeded")

// ZoneLimiter manages per-zone token-bucket rate limiters.
// Each zone gets its own rate.Limiter; limiters are lazily created.
type ZoneLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	ratePerH float64 // configured rate per hour per zone
	burst    int     // burst size (tokens available immediately)

	// onLimited is called every time a request is rate-limited (for metrics).
	onLimited func(zone string)
}

// Config holds rate-limiter configuration.
type Config struct {
	RatePerHour float64 // max requests per hour per zone (0 → default 300)
	Burst       int     // burst capacity; 0 → max(1, ratePerHour/60)
}

// NewZoneLimiter creates a ZoneLimiter with the given configuration.
// If RatePerHour is zero or negative, it defaults to DefaultRatePerHour (300).
// If Burst is zero or negative, it defaults to max(1, RatePerHour/60), ensuring
// at least one token is always available for immediate use.
func NewZoneLimiter(cfg Config) *ZoneLimiter {
	r := cfg.RatePerHour
	if r <= 0 {
		r = DefaultRatePerHour
	}
	burst := cfg.Burst
	if burst <= 0 {
		burst = int(r / 60)
		if burst < 1 {
			burst = 1
		}
	}
	return &ZoneLimiter{
		limiters: make(map[string]*rate.Limiter),
		ratePerH: r,
		burst:    burst,
	}
}

// SetOnLimited registers a callback invoked when a request is rate-limited.
func (zl *ZoneLimiter) SetOnLimited(fn func(zone string)) {
	zl.mu.Lock()
	defer zl.mu.Unlock()
	zl.onLimited = fn
}

// Allow checks whether a request for the given zone is permitted without blocking.
// Returns nil if allowed, ErrRateLimited if the zone bucket is exhausted.
func (zl *ZoneLimiter) Allow(zone string) error {
	lim := zl.getLimiter(zone)
	if lim.Allow() {
		return nil
	}
	zl.mu.RLock()
	fn := zl.onLimited
	zl.mu.RUnlock()
	if fn != nil {
		fn(zone)
	}
	return ErrRateLimited
}

// Wait blocks until a token is available for the given zone or the context is cancelled.
// Returns nil if a token was acquired. If the context is cancelled or times out,
// the context error is returned directly (without triggering rate-limit metrics).
// The onLimited callback and ErrRateLimited are only used when the limiter itself
// rejects the request (e.g. rate.Wait returns an exceeded-rate error).
func (zl *ZoneLimiter) Wait(ctx context.Context, zone string) error {
	lim := zl.getLimiter(zone)
	if err := lim.Wait(ctx); err != nil {
		// Distinguish context cancellation from genuine rate-limit rejection.
		// Context errors (cancel/deadline) are not rate-limit events and should
		// not inflate regru_rate_limited_total.
		if ctx.Err() != nil {
			return ctx.Err()
		}
		zl.mu.RLock()
		fn := zl.onLimited
		zl.mu.RUnlock()
		if fn != nil {
			fn(zone)
		}
		return fmt.Errorf("%w: %v", ErrRateLimited, err)
	}
	return nil
}

// Reserve returns a rate.Reservation for the given zone.
func (zl *ZoneLimiter) Reserve(zone string) *rate.Reservation {
	return zl.getLimiter(zone).Reserve()
}

// getLimiter returns the limiter for a zone, creating one if absent.
func (zl *ZoneLimiter) getLimiter(zone string) *rate.Limiter {
	zl.mu.RLock()
	lim, ok := zl.limiters[zone]
	zl.mu.RUnlock()
	if ok {
		return lim
	}

	zl.mu.Lock()
	defer zl.mu.Unlock()
	// Double-check after acquiring write lock.
	if lim, ok = zl.limiters[zone]; ok {
		return lim
	}
	lim = rate.NewLimiter(rate.Limit(zl.ratePerH/3600.0), zl.burst)
	zl.limiters[zone] = lim
	return lim
}

// RatePerHour returns the configured rate per hour.
func (zl *ZoneLimiter) RatePerHour() float64 {
	return zl.ratePerH
}

// Burst returns the configured burst size.
func (zl *ZoneLimiter) Burst() int {
	return zl.burst
}

// Zones returns the list of zones that currently have active limiters.
func (zl *ZoneLimiter) Zones() []string {
	zl.mu.RLock()
	defer zl.mu.RUnlock()
	zones := make([]string, 0, len(zl.limiters))
	for z := range zl.limiters {
		zones = append(zones, z)
	}
	return zones
}

// IsThrottling returns true if the given zone has no tokens available
// (i.e., the zone is actively being rate-limited).
func (zl *ZoneLimiter) IsThrottling(zone string) bool {
	zl.mu.RLock()
	lim, ok := zl.limiters[zone]
	zl.mu.RUnlock()
	if !ok {
		return false
	}
	// If Allow would deny, the zone is throttling.
	r := lim.ReserveN(time.Now(), 0)
	return r.DelayFrom(time.Now()) > 0
}

// ThrottledZones returns a list of zones currently under throttling.
func (zl *ZoneLimiter) ThrottledZones() []string {
	zl.mu.RLock()
	defer zl.mu.RUnlock()
	var zones []string
	for z, lim := range zl.limiters {
		if !lim.AllowN(time.Now(), 0) {
			continue
		}
		// Check if a regular request would be denied.
		r := lim.ReserveN(time.Now(), 0)
		if r.DelayFrom(time.Now()) > 0 {
			zones = append(zones, z)
		}
	}
	return zones
}
