package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/yourorg/externaldns-regru-sidecar/internal/auth"
)

// app holds the wired application state, exposed for testing.
type app struct {
	mux        *http.ServeMux
	driver     auth.AuthDriver
	credsValid *int32 // 1 = valid, 0 = invalid
}

// newApp loads credentials via EnvSecretProvider, builds the HTTP mux,
// and wires /ready to credential validity. This is the single source of
// truth for application setup — both main() and tests use it.
func newApp() *app {
	a := &app{
		mux:        http.NewServeMux(),
		credsValid: new(int32),
	}

	// --- credential loading ---
	provider := &auth.EnvSecretProvider{}
	creds, err := provider.LoadCredentials()
	if err != nil {
		log.Printf("WARNING: credentials not available: %v (adapter will not be ready)", err)
		// NOTE: credential values are never printed — only presence/absence is logged (NFR-Sec1).
	} else {
		atomic.StoreInt32(a.credsValid, 1)
		log.Println("credentials loaded successfully")
		a.driver = &auth.TokenDriver{Creds: creds}
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

func main() {
	a := newApp()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: a.mux,
	}

	go func() {
		log.Println("sidecar starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("sidecar shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("sidecar stopped")
}
