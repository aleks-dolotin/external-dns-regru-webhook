package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/auth"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/config"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/health"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/metrics"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/normalizer"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/worker"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// app holds the wired application state, exposed for testing.
type app struct {
	mux         *http.ServeMux
	driver      auth.AuthDriver
	reloader    *auth.ReloadableDriver // nil when rotation is disabled or initial load failed
	credsValid  *int32                 // 1 = valid, 0 = invalid
	checker     *health.Checker
	diagSrc     *diagnosticsSource   // diagnostics data source (Story 10.2)
	queue       *queue.InMemoryQueue // event queue (Story 2.1)
	pool        *worker.WorkerPool   // worker pool (Story 2.4)
	configStore *config.Store        // zone-namespace mapping config (Story 3.1)
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

	// --- Prometheus metrics ---
	metrics.Register(prometheus.DefaultRegisterer)
	a.mux.Handle("/metrics", promhttp.Handler())

	// --- diagnostics endpoint (Story 10.2) ---
	a.queue = queue.New()
	a.pool = worker.New(nil, a.queue) // adapter wired when Epic 1 adapter is instantiated
	a.diagSrc = &diagnosticsSource{q: a.queue, p: a.pool}
	a.mux.HandleFunc("/adapter/v1/diagnostics", health.DiagnosticsHandler(a.diagSrc))

	// --- event intake endpoint (Story 2.1) ---
	a.mux.HandleFunc("/adapter/v1/events", a.handleEvents)

	// --- config store: zone-namespace mappings (Story 3.1) ---
	cfgPath := os.Getenv("REGADAPTER_MAPPINGS_PATH")
	if cfgPath == "" {
		cfgPath = "/etc/reg-adapter/mappings.yaml"
	}
	cfgStore, err := config.NewStore(cfgPath)
	if err != nil {
		log.Printf("WARNING: config store not loaded: %v (will retry via hot-reload)", err)
	} else {
		a.configStore = cfgStore
		log.Printf("config store loaded: %d zone mappings from %s", len(cfgStore.Get().Zones), cfgPath)
	}

	return a
}

// handleEvents accepts POST requests with ExternalDNS events, normalizes
// them into adapter.Operation payloads, filters by zone-namespace mapping
// (Story 3.1), applies FQDN templates (Story 3.3), and enqueues for processing.
func (a *app) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var events []normalizer.DNSEndpointEvent
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		http.Error(w, `{"error":"invalid JSON: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	ops, errs := normalizer.NormalizeBatch(events)

	// Enqueue valid operations, applying config rules when configStore is loaded.
	var accepted int
	for i := range ops {
		// Story 3.1: enforce zone-namespace mapping when config is available.
		if a.configStore != nil {
			zone := ops[i].Zone
			namespace := ops[i].ResourceRef.Namespace

			if !a.configStore.IsNamespaceAllowed(zone, namespace) {
				errs = append(errs, fmt.Errorf("op %s: namespace %q not allowed for zone %q",
					ops[i].OpID, namespace, zone))
				continue
			}

			// Story 3.3: apply FQDN template from zone mapping.
			zm := a.configStore.FindZone(zone)
			if zm != nil {
				labels := extractLabels(ops[i].K8sMeta)
				fqdn, err := config.RenderFQDNForZone(zm, ops[i].ResourceRef.Name, namespace, labels)
				if err != nil {
					log.Printf("WARNING: FQDN template failed for op %s: %v (keeping original FQDN)",
						ops[i].OpID, err)
				} else {
					ops[i].Name = fqdn
				}

				// Apply TTL/priority defaults from zone mapping when not set on event.
				if ops[i].TTL == 0 && zm.TTL > 0 {
					ops[i].TTL = zm.TTL
				}
				if ops[i].Priority == 0 && zm.Priority > 0 {
					ops[i].Priority = zm.Priority
				}
			}
		}

		a.queue.Enqueue(queue.Operation{
			ID:   ops[i].OpID,
			Body: ops[i],
		})
		accepted++
	}

	resp := eventIntakeResponse{
		Accepted: accepted,
		Errors:   len(errs),
	}
	for _, e := range errs {
		resp.ErrorDetails = append(resp.ErrorDetails, e.Error())
	}

	code := http.StatusOK
	if len(errs) > 0 && accepted == 0 {
		code = http.StatusBadRequest
	} else if len(errs) > 0 {
		code = http.StatusPartialContent // 206: some accepted, some rejected
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(resp)
}

// extractLabels safely extracts string labels from K8sMeta["labels"].
func extractLabels(meta map[string]interface{}) map[string]string {
	if meta == nil {
		return nil
	}
	raw, ok := meta["labels"]
	if !ok {
		return nil
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	labels := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			labels[k] = s
		}
	}
	return labels
}

// eventIntakeResponse is returned by POST /adapter/v1/events.
type eventIntakeResponse struct {
	Accepted     int      `json:"accepted"`
	Errors       int      `json:"errors"`
	ErrorDetails []string `json:"error_details,omitempty"`
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

// DefaultWorkerConcurrency is used when WORKER_CONCURRENCY is not set.
const DefaultWorkerConcurrency = 2

// workerConcurrency reads WORKER_CONCURRENCY from env; defaults to DefaultWorkerConcurrency.
func workerConcurrency() int {
	s := os.Getenv("WORKER_CONCURRENCY")
	if s == "" {
		return DefaultWorkerConcurrency
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return DefaultWorkerConcurrency
	}
	return n
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

	// Start config hot-reload loop (Story 3.1).
	if a.configStore != nil {
		go a.configStore.RunReloader(ctx.Done(), config.DefaultReloadInterval)
	}

	// Start worker pool with configurable concurrency (Story 2.4).
	concurrency := workerConcurrency()
	a.pool.Start(ctx, concurrency)
	log.Printf("worker pool started with %d workers", concurrency)

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
