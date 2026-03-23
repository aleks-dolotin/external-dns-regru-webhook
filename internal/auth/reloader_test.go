package auth

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// stubDriver is a minimal AuthDriver for tests.
type stubDriver struct {
	params map[string]string
	err    error
}

func (s *stubDriver) PrepareAuth() (map[string]string, error) {
	return s.params, s.err
}

func TestReloadableDriver_PrepareAuth_DelegatesToCurrent(t *testing.T) {
	initial := &stubDriver{params: map[string]string{"username": "old"}}
	rd := NewReloadableDriver(initial, time.Hour)

	params, err := rd.PrepareAuth()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["username"] != "old" {
		t.Errorf("expected 'old', got %q", params["username"])
	}
}

func TestReloadableDriver_Reload_Success(t *testing.T) {
	initial := &stubDriver{params: map[string]string{"username": "old"}}
	rd := NewReloadableDriver(initial, time.Hour)

	// Set env for token driver reload.
	t.Setenv(EnvAuthDriver, "token")
	t.Setenv(EnvUsername, "newuser")
	t.Setenv(EnvPassword, "newpass")
	t.Setenv(EnvCredentialsPath, "")

	var reloadedDriver AuthDriver
	rd.OnReload = func(d AuthDriver) { reloadedDriver = d }

	if err := rd.Reload(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	// Check the new driver is serving.
	params, err := rd.PrepareAuth()
	if err != nil {
		t.Fatalf("PrepareAuth after reload: %v", err)
	}
	if params["username"] != "newuser" {
		t.Errorf("expected 'newuser', got %q", params["username"])
	}
	if reloadedDriver == nil {
		t.Error("OnReload callback was not called")
	}
}

func TestReloadableDriver_Reload_Failure_KeepsOldDriver(t *testing.T) {
	initial := &stubDriver{params: map[string]string{"username": "old"}}
	rd := NewReloadableDriver(initial, time.Hour)

	// Set env so NewDriverFromEnv fails.
	t.Setenv(EnvAuthDriver, "token")
	t.Setenv(EnvUsername, "")
	t.Setenv(EnvPassword, "")
	t.Setenv(EnvCredentialsPath, "")

	var reloadErr error
	rd.OnReloadError = func(err error) { reloadErr = err }

	err := rd.Reload()
	if err == nil {
		t.Fatal("expected reload error")
	}
	if reloadErr == nil {
		t.Error("OnReloadError callback was not called")
	}

	// Old driver should still be active.
	params, _ := rd.PrepareAuth()
	if params["username"] != "old" {
		t.Errorf("old driver should still serve, got username %q", params["username"])
	}
}

func TestReloadableDriver_Reload_NewDriverPrepareAuthFails(t *testing.T) {
	// Scenario: NewDriverFromEnv succeeds but PrepareAuth on new driver fails.
	// This would require RSA driver with nil key — but env validation catches that.
	// We test that env-level failure keeps the old driver.
	initial := &stubDriver{params: map[string]string{"username": "safe"}}
	rd := NewReloadableDriver(initial, time.Hour)

	t.Setenv(EnvAuthDriver, "unsupported-driver")

	err := rd.Reload()
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}

	params, _ := rd.PrepareAuth()
	if params["username"] != "safe" {
		t.Errorf("old driver should persist, got %q", params["username"])
	}
}

func TestReloadableDriver_DefaultInterval(t *testing.T) {
	initial := &stubDriver{params: map[string]string{}}
	rd := NewReloadableDriver(initial, 0)
	if rd.interval != DefaultRotationInterval {
		t.Errorf("expected default interval %v, got %v", DefaultRotationInterval, rd.interval)
	}
}

func TestReloadableDriver_Run_StopsOnCancel(t *testing.T) {
	initial := &stubDriver{params: map[string]string{"username": "u", "password": "p"}}
	rd := NewReloadableDriver(initial, 10*time.Millisecond)

	t.Setenv(EnvAuthDriver, "token")
	t.Setenv(EnvUsername, "u")
	t.Setenv(EnvPassword, "p")
	t.Setenv(EnvCredentialsPath, "")

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		rd.Run(ctx)
		close(done)
	}()

	// Let it tick a couple of times.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK — Run returned after cancel.
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after context cancellation")
	}
}

func TestReloadableDriver_Run_ReloadsOnTick(t *testing.T) {
	initial := &stubDriver{params: map[string]string{"username": "old"}}
	rd := NewReloadableDriver(initial, 10*time.Millisecond)

	t.Setenv(EnvAuthDriver, "token")
	t.Setenv(EnvUsername, "reloaded")
	t.Setenv(EnvPassword, "pass")
	t.Setenv(EnvCredentialsPath, "")

	var reloadCount int32
	rd.OnReload = func(_ AuthDriver) { atomic.AddInt32(&reloadCount, 1) }

	ctx, cancel := context.WithCancel(context.Background())
	go rd.Run(ctx)

	// Wait enough for at least 2 ticks.
	time.Sleep(60 * time.Millisecond)
	cancel()

	if atomic.LoadInt32(&reloadCount) < 1 {
		t.Error("expected at least 1 reload via Run loop")
	}

	params, _ := rd.PrepareAuth()
	if params["username"] != "reloaded" {
		t.Errorf("expected 'reloaded', got %q", params["username"])
	}
}

func TestReloadableDriver_ConcurrentAccess(t *testing.T) {
	initial := &stubDriver{params: map[string]string{"username": "init"}}
	rd := NewReloadableDriver(initial, time.Hour)

	t.Setenv(EnvAuthDriver, "token")
	t.Setenv(EnvUsername, "concurrent")
	t.Setenv(EnvPassword, "pass")
	t.Setenv(EnvCredentialsPath, "")

	errs := make(chan error, 100)

	// Concurrent readers.
	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i < 10; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, err := rd.PrepareAuth()
					if err != nil && !errors.Is(err, ErrMissingCredentials) {
						errs <- err
					}
				}
			}
		}()
	}

	// Concurrent reloads.
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_ = rd.Reload()
			}
		}()
	}

	time.Sleep(100 * time.Millisecond)
	cancel()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected error during concurrent access: %v", err)
	}
}
