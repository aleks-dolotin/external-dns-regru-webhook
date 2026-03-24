package ratelimit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewZoneLimiter_Defaults(t *testing.T) {
	zl := NewZoneLimiter(Config{})
	if zl.RatePerHour() != DefaultRatePerHour {
		t.Errorf("expected default rate %v, got %v", DefaultRatePerHour, zl.RatePerHour())
	}
	if zl.Burst() < 1 {
		t.Errorf("expected burst >= 1, got %d", zl.Burst())
	}
}

func TestNewZoneLimiter_CustomConfig(t *testing.T) {
	zl := NewZoneLimiter(Config{RatePerHour: 1000, Burst: 10})
	if zl.RatePerHour() != 1000 {
		t.Errorf("expected rate 1000, got %v", zl.RatePerHour())
	}
	if zl.Burst() != 10 {
		t.Errorf("expected burst 10, got %d", zl.Burst())
	}
}

func TestAllow_WithinRate(t *testing.T) {
	// High rate so burst is large enough for a few requests.
	zl := NewZoneLimiter(Config{RatePerHour: 36000, Burst: 5})

	for i := 0; i < 5; i++ {
		if err := zl.Allow("example.com"); err != nil {
			t.Errorf("request %d should be allowed: %v", i, err)
		}
	}
}

func TestAllow_ExceedsRate(t *testing.T) {
	// Tiny rate: 1 req/hour, burst=1 → second request must be rejected.
	zl := NewZoneLimiter(Config{RatePerHour: 1, Burst: 1})

	if err := zl.Allow("example.com"); err != nil {
		t.Fatalf("first request should be allowed: %v", err)
	}
	if err := zl.Allow("example.com"); !errors.Is(err, ErrRateLimited) {
		t.Errorf("second request should be rate-limited, got: %v", err)
	}
}

func TestAllow_PerZoneIsolation(t *testing.T) {
	zl := NewZoneLimiter(Config{RatePerHour: 1, Burst: 1})

	// Exhaust zone A.
	_ = zl.Allow("zone-a.com")
	if err := zl.Allow("zone-a.com"); !errors.Is(err, ErrRateLimited) {
		t.Error("zone-a should be rate-limited")
	}

	// Zone B should still be allowed.
	if err := zl.Allow("zone-b.com"); err != nil {
		t.Errorf("zone-b should not be affected: %v", err)
	}
}

func TestAllow_OnLimitedCallback(t *testing.T) {
	zl := NewZoneLimiter(Config{RatePerHour: 1, Burst: 1})
	var callbackZone string
	var callbackCount int32
	zl.SetOnLimited(func(zone string) {
		atomic.AddInt32(&callbackCount, 1)
		callbackZone = zone
	})

	_ = zl.Allow("test.com") // allowed
	_ = zl.Allow("test.com") // limited → callback fires
	if atomic.LoadInt32(&callbackCount) != 1 {
		t.Errorf("expected 1 callback, got %d", callbackCount)
	}
	if callbackZone != "test.com" {
		t.Errorf("expected zone test.com, got %s", callbackZone)
	}
}

func TestWait_Success(t *testing.T) {
	zl := NewZoneLimiter(Config{RatePerHour: 360000, Burst: 10})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := zl.Wait(ctx, "example.com"); err != nil {
		t.Errorf("wait should succeed: %v", err)
	}
}

func TestWait_ContextCancelled(t *testing.T) {
	zl := NewZoneLimiter(Config{RatePerHour: 1, Burst: 1})
	_ = zl.Allow("example.com") // exhaust burst

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := zl.Wait(ctx, "example.com")
	if err == nil {
		t.Error("expected error when context expires")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got: %v", err)
	}
}

func TestZones(t *testing.T) {
	zl := NewZoneLimiter(Config{RatePerHour: 3600, Burst: 5})
	_ = zl.Allow("a.com")
	_ = zl.Allow("b.com")

	zones := zl.Zones()
	if len(zones) != 2 {
		t.Errorf("expected 2 zones, got %d", len(zones))
	}
}

func TestConcurrentAccess(t *testing.T) {
	zl := NewZoneLimiter(Config{RatePerHour: 360000, Burst: 100})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			zone := "zone.com"
			if id%2 == 0 {
				zone = "other.com"
			}
			_ = zl.Allow(zone)
		}(i)
	}
	wg.Wait()
	// No race, no panic — success.
}

func TestSyntheticBurst_CappedRate(t *testing.T) {
	// AC: Given synthetic burst, When limiter enabled, Then outgoing requests
	// are capped at configured per-zone rate.
	// Rate: 3600/hr = 1/sec, burst=2 → immediate 2, then ~1/sec.
	zl := NewZoneLimiter(Config{RatePerHour: 3600, Burst: 2})

	allowed := 0
	limited := 0
	// Fire 10 requests instantly.
	for i := 0; i < 10; i++ {
		if err := zl.Allow("burst-test.com"); err == nil {
			allowed++
		} else {
			limited++
		}
	}

	if allowed > 3 {
		t.Errorf("burst should cap allowed requests; got %d allowed out of 10", allowed)
	}
	if limited == 0 {
		t.Error("expected some requests to be rate-limited during burst")
	}
}
