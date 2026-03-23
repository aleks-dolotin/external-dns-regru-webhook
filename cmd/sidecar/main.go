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

	"github.com/yourorg/externaldns-regru-sidecar/internal/auth"
)

// app holds the wired application state, exposed for testing.
type app struct {
	mux        *http.ServeMux
	driver     auth.AuthDriver
	reloader   *auth.ReloadableDriver // nil when rotation is disabled or initial load failed
	credsValid *int32                 // 1 = valid, 0 = invalid
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

	// /healthz — liveness: always OK (process is running).
	a.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// /ready — readiness: OK only when credentials are valid.
	a.mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(a.credsValid) == 1 {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "credentials not available", http.StatusServiceUnavailable)
	})

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
