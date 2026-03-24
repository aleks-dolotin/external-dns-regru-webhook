package coordinator

import (
	"context"
	"testing"
	"time"
)

func TestNoopCoordinator_AlwaysGrants(t *testing.T) {
	c := &NoopCoordinator{}
	if err := c.RequestQuota(context.Background(), "zone.com"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestNoopCoordinator_NotAvailable(t *testing.T) {
	c := &NoopCoordinator{}
	if c.Available() {
		t.Error("NoopCoordinator should report not available")
	}
}

func TestNoopCoordinator_ReleaseQuota(t *testing.T) {
	c := &NoopCoordinator{}
	c.ReleaseQuota("zone.com") // should not panic
}

func TestDefaultConfig_Disabled(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Enabled {
		t.Error("default config should be disabled")
	}
	if cfg.Timeout != 500*time.Millisecond {
		t.Errorf("expected 500ms timeout, got %v", cfg.Timeout)
	}
	if !cfg.FallbackLocal {
		t.Error("expected FallbackLocal=true")
	}
}
