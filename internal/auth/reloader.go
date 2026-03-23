package auth

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Default rotation check interval.
const DefaultRotationInterval = 30 * time.Second

// driverHolder wraps AuthDriver so that atomic.Value always stores the
// same concrete type, regardless of the underlying driver implementation.
type driverHolder struct {
	d AuthDriver
}

// ReloadableDriver wraps an AuthDriver and periodically refreshes it
// by re-reading credentials from disk (K8s secret volume mount).
// During rotation the old driver continues serving requests — no downtime.
type ReloadableDriver struct {
	driver   atomic.Value // stores driverHolder
	interval time.Duration
	mu       sync.Mutex // serialises reload attempts

	// Callback invoked after successful reload (for readiness updates, metrics, etc.).
	OnReload func(AuthDriver)
	// Callback invoked when reload fails.
	OnReloadError func(error)
}

// NewReloadableDriver creates a ReloadableDriver seeded with initial.
// interval controls how often credentials are re-read; 0 means DefaultRotationInterval.
func NewReloadableDriver(initial AuthDriver, interval time.Duration) *ReloadableDriver {
	if interval <= 0 {
		interval = DefaultRotationInterval
	}
	r := &ReloadableDriver{interval: interval}
	r.driver.Store(driverHolder{d: initial})
	return r
}

// PrepareAuth delegates to the currently loaded driver (lock-free read).
func (r *ReloadableDriver) PrepareAuth() (map[string]string, error) {
	return r.current().PrepareAuth()
}

// current returns the active AuthDriver.
func (r *ReloadableDriver) current() AuthDriver {
	return r.driver.Load().(driverHolder).d
}

// Reload attempts to create a new driver from environment.
// On success the old driver is atomically replaced; on failure the old driver
// continues serving — no request is affected.
func (r *ReloadableDriver) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	newDriver, err := NewDriverFromEnv()
	if err != nil {
		if r.OnReloadError != nil {
			r.OnReloadError(err)
		}
		return err
	}

	// Verify the new driver can actually produce auth params.
	if _, err := newDriver.PrepareAuth(); err != nil {
		if r.OnReloadError != nil {
			r.OnReloadError(err)
		}
		return err
	}

	r.driver.Store(driverHolder{d: newDriver})
	if r.OnReload != nil {
		r.OnReload(newDriver)
	}
	return nil
}

// Run starts the periodic reload loop. It blocks until ctx is cancelled.
func (r *ReloadableDriver) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("credential reloader stopped")
			return
		case <-ticker.C:
			if err := r.Reload(); err != nil {
				log.Printf("credential reload failed (old creds still active): %v", err)
			} else {
				log.Println("credentials reloaded successfully")
			}
		}
	}
}
