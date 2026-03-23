package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/auth"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/health"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/worker"
)

// app holds the wired application state, exposed for testing.
type app struct {
	mux        *http.ServeMux
	driver     auth.AuthDriver
	reloader   *auth.ReloadableDriver // nil when rotation is disabled or initial load failed
	credsValid *int32                 // 1 = valid, 0 = invalid
	checker    *health.Checker
	diagSrc    *diagnosticsSource // diagnostics data source (Story 10.2)
}

// diagnosticsSource bridges queue and worker to health.DiagnosticsSource.
// Nil-safe: returns zero values when components are not yet wired.
type diagnosticsSource struct {
	q *queue.InMemoryQueue
	p *worker.WorkerPool
}

func (d *diagnosticsSource) QueueDepth() int {
	if d.q == nil {
		return 0
	}
	return d.q.Len()
}

func (d *diagnosticsSource) WorkerCount() int {
	if d.p == nil {
		return 0
	}
	return d.p.WorkerCount()
}

func (d *diagnosticsSource) LastHeartbeat() time.Time {
	if d.p == nil {
		return time.Time{}
	}
	return d.p.LastHeartbeat()
}

func (d *diagnosticsSource) ZoneErrors() map[string]health.ZoneErrorInfo {
	if d.p == nil {
		return nil
	}
	errs := d.p.LastErrors()
	if len(errs) == 0 {
		return nil
	}
	result := make(map[string]health.ZoneErrorInfo, len(errs))
	for zone, ze := range errs {
		result[zone] = health.ZoneErrorInfo{
			Message: ze.Message,
			Time:    ze.Time,
		}
	}
	return result
}

// newApp loads credentials via NewDriverFromEnv, optionally wraps in
// ReloadableDriver for hot rotation, builds the HTTP mux, and wires
// /ready to credential validity.
func newApp() *app {
	a := &app{
		mux:        http.NewServeMux(),
		credsValid: new(int32),
	}

	// --- driver loading via pluggable factory ---
	driver, err := auth.NewDriverFromEnv()
	if err != nil {
		log.Printf("WARNING: auth driver not available: %v (adapter will not be ready)", err)
	} else {
		atomic.StoreInt32(a.credsValid, 1)
		log.Printf("auth driver loaded successfully (type: %s)", os.Getenv(auth.EnvAuthDriver))

		// --- credential rotation ---
		interval := rotationInterval()
		rd := auth.NewReloadableDriver(driver, interval)
		rd.OnReload = func(_ auth.AuthDriver) {
			atomic.StoreInt32(a.credsValid, 1)
		}
		rd.OnReloadError = func(e error) {
			log.Printf("credential reload failed (old creds still active): %v", e)
		}
		a.reloader = rd
		a.driver = rd // adapter uses reloadable wrapper
	}

	// --- health checker with readiness sub-checks ---
	a.checker = health.NewChecker(
		health.ReadyCheck{
			Name: "credentials",
			Check: func() (bool, string) {
				if atomic.LoadInt32(a.credsValid) == 1 {
					return true, ""
				}
				return false, "credentials not available"
			},
		},
	)

	// /healthz — liveness: always OK (process is running), JSON body.
	a.mux.HandleFunc("/healthz", a.checker.HealthzHandler)

	// /ready — readiness: OK only when all checks pass, JSON body.
	a.mux.HandleFunc("/ready", a.checker.ReadyHandler)

	// --- diagnostics endpoint (Story 10.2) ---
	// Wire queue/worker into diagSrc when Epic 2 is implemented.
	a.diagSrc = &diagnosticsSource{}
	a.mux.HandleFunc("/adapter/v1/diagnostics", health.DiagnosticsHandler(a.diagSrc))

	return a
}

// rotationInterval reads REGU_ROTATION_INTERVAL_SEC from env; defaults to 30s.
func rotationInterval() time.Duration {
	s := os.Getenv("REGU_ROTATION_INTERVAL_SEC")
	if s == "" {
		return auth.DefaultRotationInterval
	}
	sec, err := strconv.Atoi(s)
	if err != nil || sec <= 0 {
		return auth.DefaultRotationInterval
	}
	return time.Duration(sec) * time.Second
}

func main() {
	a := newApp()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: a.mux,
	}

	// Start credential rotation loop if reloader is available.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if a.reloader != nil {
		go a.reloader.Run(ctx)
	}

	go func() {
		log.Println("sidecar starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("sidecar shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("sidecar stopped")
}
