package safemode

import (
	"testing"
)

func TestNew_DisabledByDefault(t *testing.T) {
	sm := New()
	if sm.IsEnabled() {
		t.Error("expected safe-mode disabled by default")
	}
	st := sm.Status()
	if st.Enabled {
		t.Error("expected status.Enabled=false")
	}
	if st.Since != nil {
		t.Error("expected status.Since=nil when disabled")
	}
	if st.SuppressedCount != 0 {
		t.Errorf("expected suppressed_count=0, got %d", st.SuppressedCount)
	}
}

func TestEnable(t *testing.T) {
	sm := New()
	sm.Enable()

	if !sm.IsEnabled() {
		t.Error("expected safe-mode enabled")
	}
	st := sm.Status()
	if !st.Enabled {
		t.Error("expected status.Enabled=true")
	}
	if st.Since == nil {
		t.Error("expected non-nil Since after enable")
	}
}

func TestDisable(t *testing.T) {
	sm := New()
	sm.Enable()
	sm.Disable()

	if sm.IsEnabled() {
		t.Error("expected safe-mode disabled after Disable()")
	}
	st := sm.Status()
	if st.Enabled {
		t.Error("expected status.Enabled=false after Disable()")
	}
	if st.Since != nil {
		t.Error("expected status.Since=nil after Disable()")
	}
}

func TestEnableIdempotent(t *testing.T) {
	sm := New()
	sm.Enable()
	since1 := sm.Status().Since

	// Second enable should not reset since.
	sm.Enable()
	since2 := sm.Status().Since

	if since1 == nil || since2 == nil {
		t.Fatal("expected non-nil Since")
	}
	if !since1.Equal(*since2) {
		t.Errorf("double Enable() changed Since: %v → %v", since1, since2)
	}
}

func TestIncrementSuppressed(t *testing.T) {
	sm := New()
	sm.Enable()
	sm.IncrementSuppressed()
	sm.IncrementSuppressed()
	sm.IncrementSuppressed()

	st := sm.Status()
	if st.SuppressedCount != 3 {
		t.Errorf("expected suppressed_count=3, got %d", st.SuppressedCount)
	}
}

func TestSuppressedCountResetsOnEnable(t *testing.T) {
	sm := New()
	sm.Enable()
	sm.IncrementSuppressed()
	sm.IncrementSuppressed()

	// Disable then re-enable should reset counter.
	sm.Disable()
	sm.Enable()

	st := sm.Status()
	if st.SuppressedCount != 0 {
		t.Errorf("expected suppressed_count=0 after re-enable, got %d", st.SuppressedCount)
	}
}
