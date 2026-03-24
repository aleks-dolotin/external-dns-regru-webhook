package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/audit"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/auth"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/circuitbreaker"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/config"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/desiredstate"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/health"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/logging"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/metrics"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/normalizer"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/quota"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/ratelimit"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/reconciler"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/safemode"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/worker"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// app holds the wired application state, exposed for testing.

// Version is the adapter version string, injected at build time via
// -ldflags "-X main.Version=x.y.z". Defaults to "dev" for local builds.
var Version = "dev"

type app struct {
	mux         *http.ServeMux
	logger      *zap.Logger // Story 6.2: structured logger
	driver      auth.AuthDriver
	reloader    *auth.ReloadableDriver // nil when rotation is disabled or initial load failed
	credsValid  *int32                 // 1 = valid, 0 = invalid
	checker     *health.Checker
	diagSrc     *diagnosticsSource      // diagnostics data source (Story 10.2)
	queue       *queue.InMemoryQueue    // event queue (Story 2.1)
	pool        *worker.WorkerPool      // worker pool (Story 2.4)
	configStore *config.Store           // zone-namespace mapping config (Story 3.1)
	limiter     *ratelimit.ZoneLimiter  // Story 5.1: per-zone rate limiter
	breaker     *circuitbreaker.Manager // Story 5.4: per-zone circuit breaker
	auditor     audit.Auditor           // Story 6.3: audit trail
	reconciler  *reconciler.Reconciler  // Story 8.1: reconciler for force-resync
	resync      resyncState             // Story 8.1: force-resync state
	safeMode    *safemode.SafeMode      // Story 8.3: safe-mode toggle
	quotaMgr    *quota.Manager          // Story 9.2: per-namespace quota manager
	desired     *desiredstate.Cache     // Story 8.1: desired DNS state cache for force-resync
}

// diagnosticsSource bridges queue and worker to health.DiagnosticsSource.
// Nil-safe: returns zero values when components are not yet wired.
type diagnosticsSource struct {
	q      *queue.InMemoryQueue
	p      *worker.WorkerPool
	lim    *ratelimit.ZoneLimiter  // Story 5.1/5.5
	cb     *circuitbreaker.Manager // Story 5.4
	resync *resyncState            // Story 8.1: force-resync status
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

// IsBackpressured returns true when the queue depth exceeds the threshold (Story 5.5).
func (d *diagnosticsSource) IsBackpressured() bool {
	if d.q == nil {
		return false
	}
	return d.q.IsBackpressured()
}

// ThrottledZones returns zones where the rate limiter is actively throttling (Story 5.5).
func (d *diagnosticsSource) ThrottledZones() []string {
	if d.lim == nil {
		return nil
	}
	return d.lim.ThrottledZones()
}

// CircuitStates returns per-zone circuit breaker states (Story 5.4).
func (d *diagnosticsSource) CircuitStates() map[string]string {
	if d.cb == nil {
		return nil
	}
	states := d.cb.ZoneStates()
	if len(states) == 0 {
		return nil
	}
	result := make(map[string]string, len(states))
	for zone, st := range states {
		result[zone] = st.String()
	}
	return result
}

// ResyncRunning returns true when a force-resync is in progress (Story 8.1).
func (d *diagnosticsSource) ResyncRunning() bool {
	if d.resync == nil {
		return false
	}
	return d.resync.running
}

// LastResyncTime returns the time of the last completed resync (Story 8.1).
func (d *diagnosticsSource) LastResyncTime() time.Time {
	if d.resync == nil {
		return time.Time{}
	}
	return d.resync.lastResyncTime
}

// LastResyncActions returns the number of corrective actions from the last resync (Story 8.1).
func (d *diagnosticsSource) LastResyncActions() int {
	if d.resync == nil {
		return 0
	}
	return d.resync.lastResyncOps
}

// LastResyncError returns the error message from the last resync, if any (Story 8.1).
func (d *diagnosticsSource) LastResyncError() string {
	if d.resync == nil {
		return ""
	}
	return d.resync.lastResyncError
}

// newApp loads credentials via NewDriverFromEnv, optionally wraps in
// ReloadableDriver for hot rotation, builds the HTTP mux, and wires
// /ready to credential validity.
func newApp() *app {
	// Story 6.2: initialize structured logger.
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logger := logging.NewLogger(logLevel)

	a := &app{
		mux:        http.NewServeMux(),
		logger:     logger,
		credsValid: new(int32),
	}

	// --- driver loading via pluggable factory ---
	driver, err := auth.NewDriverFromEnv()
	if err != nil {
		logger.Warn("auth driver not available", zap.Error(err), zap.String("impact", "adapter will not be ready"))
	} else {
		atomic.StoreInt32(a.credsValid, 1)
		logger.Info("auth driver loaded successfully", zap.String("type", os.Getenv(auth.EnvAuthDriver)))

		// --- credential rotation ---
		interval := rotationInterval()
		rd := auth.NewReloadableDriver(driver, interval)
		rd.OnReload = func(_ auth.AuthDriver) {
			atomic.StoreInt32(a.credsValid, 1)
		}
		rd.OnReloadError = func(e error) {
			logger.Error("credential reload failed", zap.Error(e), zap.String("impact", "old creds still active"))
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
	// Story 6.4: enable OpenMetrics format to expose exemplars with correlating_id.
	a.mux.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	))

	// --- diagnostics endpoint (Story 10.2) ---
	a.queue = queue.New()
	a.desired = desiredstate.New()    // Story 8.1: desired DNS state cache
	a.pool = worker.New(nil, a.queue) // adapter wired when Epic 1 adapter is instantiated
	a.pool.SetLogger(logger)          // Story 6.2: inject structured logger

	// Story 6.4: inject structured logger into normalizer for tracing.
	normalizer.SetLogger(logger)

	// --- Story 6.3: audit trail (stdout via zap → cluster log aggregation) ---
	a.auditor = audit.NewLogAuditor(logger)
	a.pool.SetAuditor(a.auditor)

	// --- Story 5.1: per-zone rate limiter ---
	a.limiter = ratelimit.NewZoneLimiter(ratelimit.Config{
		RatePerHour: rateLimitPerHour(),
	})
	// Story 9.3: RateLimitedTotal now has namespace label; the per-zone limiter
	// callback lacks namespace context, so metric is recorded in worker.go instead.
	a.limiter.SetOnLimited(func(zone string) {
		// Intentionally empty — metric recorded in worker.handleWithRetry with full labels.
	})
	a.pool.SetLimiter(a.limiter)

	// --- Story 5.4: per-zone circuit breaker ---
	a.breaker = circuitbreaker.NewManager(circuitbreaker.DefaultConfig())
	a.breaker.OnStateChange = func(zone string, from, to circuitbreaker.State) {
		metrics.CircuitState.WithLabelValues(zone).Set(float64(to))
		logger.Info("circuit breaker state change",
			zap.String("zone", zone),
			zap.String("from", from.String()),
			zap.String("to", to.String()),
		)
	}
	a.pool.SetCircuitBreakers(a.breaker)

	// --- Story 8.3: safe-mode toggle ---
	a.safeMode = safemode.New()
	if os.Getenv("REGADAPTER_SAFE_MODE") == "true" {
		a.safeMode.Enable()
		logger.Info("safe-mode enabled from environment variable")
	}
	a.pool.SetSafeMode(a.safeMode)

	// --- Story 8.1: wire reconciler when adapter is available ---
	if a.driver != nil {
		httpAdapter := adapter.NewHTTPAdapter(a.driver)
		a.pool = worker.New(httpAdapter, a.queue) // re-create pool with actual adapter
		a.pool.SetLogger(logger)
		a.pool.SetAuditor(a.auditor)
		a.pool.SetLimiter(a.limiter)
		a.pool.SetCircuitBreakers(a.breaker)
		a.pool.SetSafeMode(a.safeMode)

		reconcileInterval := reconcileIntervalFromEnv()
		a.reconciler = reconciler.New(httpAdapter, a.queue, reconcileInterval)
		logger.Info("reconciler initialized", zap.Duration("interval", reconcileInterval))
	} else {
		logger.Warn("reconciler not available: adapter not initialized (credentials missing)")
	}

	a.diagSrc = &diagnosticsSource{q: a.queue, p: a.pool, lim: a.limiter, cb: a.breaker, resync: &a.resync}
	a.mux.HandleFunc("/adapter/v1/diagnostics", health.DiagnosticsHandler(a.diagSrc))

	// --- event intake endpoint (Story 2.1) ---
	a.mux.HandleFunc("/adapter/v1/events", a.handleEvents)

	// --- force-resync endpoint (Story 8.1) ---
	a.mux.HandleFunc("/adapter/v1/resync", a.handleResync)

	// --- replay failed operations (Story 8.2) ---
	a.mux.HandleFunc("GET /adapter/v1/failed", a.handleListFailed)
	a.mux.HandleFunc("POST /adapter/v1/replay/{id}", a.handleReplay)
	a.mux.HandleFunc("POST /adapter/v1/replay-all", a.handleReplayAll)

	// --- safe-mode toggle (Story 8.3) ---
	a.mux.HandleFunc("POST /adapter/v1/safe-mode", a.handleSafeModeToggle)
	a.mux.HandleFunc("GET /adapter/v1/safe-mode", a.handleSafeModeStatus)

	// --- config store: zone-namespace mappings (Story 3.1) ---
	cfgPath := os.Getenv("REGADAPTER_MAPPINGS_PATH")
	if cfgPath == "" {
		cfgPath = "/etc/reg-adapter/mappings.yaml"
	}
	cfgStore, err := config.NewStore(cfgPath)
	if err != nil {
		logger.Warn("config store not loaded", zap.Error(err), zap.String("impact", "will retry via hot-reload"))
	} else {
		a.configStore = cfgStore
		logger.Info("config store loaded", zap.Int("zone_count", len(cfgStore.Get().Zones)), zap.String("path", cfgPath))

		// Story 9.2: initialize per-namespace quota manager from config.
		a.quotaMgr = buildQuotaManager(cfgStore.Get())
		cfgStore.OnReload = func(cfg *config.MappingsConfig) {
			a.quotaMgr.UpdateQuotas(extractQuotas(cfg))
			logger.Info("quota manager updated after config reload")
		}
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

	// Story 6.4: log event receipt for end-to-end tracing (receive step).
	a.logger.Info("events received", zap.Int("count", len(events)))

	ops, errs := normalizer.NormalizeBatch(events)

	// Enqueue valid operations, applying config rules when configStore is loaded.
	var accepted int
	for i := range ops {
		// Story 3.1: enforce zone-namespace mapping when config is available.
		if a.configStore != nil {
			zone := ops[i].Zone
			namespace := ops[i].ResourceRef.Namespace

			if !a.configStore.IsNamespaceAllowed(zone, namespace) {
				// Story 9.1: increment namespace rejection metric.
				metrics.NamespaceRejectedTotal.WithLabelValues(zone, namespace).Inc()
				errs = append(errs, fmt.Errorf("op %s: namespace %q not allowed for zone %q",
					ops[i].OpID, namespace, zone))
				continue
			}

			// Story 9.1: use namespace-specific zone mapping for per-namespace templates.
			zm := a.configStore.FindZoneForNamespace(zone, namespace)
			if zm != nil {
				labels := extractLabels(ops[i].K8sMeta)
				fqdn, err := config.RenderFQDNForZone(zm, ops[i].ResourceRef.Name, namespace, labels)
				if err != nil {
					a.logger.Warn("FQDN template failed",
						zap.String("correlating_id", ops[i].OpID),
						zap.Error(err),
						zap.String("impact", "keeping original FQDN"),
					)
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

		// Story 9.2: per-namespace quota enforcement (after namespace check, before enqueue).
		if a.quotaMgr != nil {
			namespace := ops[i].ResourceRef.Namespace
			if !a.quotaMgr.AllowRequest(namespace) {
				used, limit := a.quotaMgr.CurrentUsage(namespace)
				metrics.NamespaceQuotaRejectedTotal.WithLabelValues(namespace).Inc()
				metrics.NamespaceQuotaUsed.WithLabelValues(namespace).Set(float64(used))
				errs = append(errs, fmt.Errorf("op %s: %s",
					ops[i].OpID, quota.RejectionMessage(namespace, limit)))
				continue
			}
		}

		a.queue.Enqueue(queue.Operation{
			ID:   ops[i].OpID,
			Body: ops[i],
		})
		// Story 8.1: update desired state cache for force-resync.
		if a.desired != nil {
			switch ops[i].Action {
			case "create", "update":
				a.desired.Put(ops[i].Zone, ops[i].Name, ops[i].Type, ops[i].Content, ops[i].TTL)
			case "delete":
				a.desired.Remove(ops[i].Zone, ops[i].Name, ops[i].Type)
			}
		}
		// Story 6.4: log enqueue with correlating_id for end-to-end tracing.
		a.logger.Info("event enqueued",
			zap.String("correlating_id", ops[i].OpID),
			zap.String("zone", ops[i].Zone),
			zap.String("operation", ops[i].Action),
			zap.String("fqdn", ops[i].Name),
			zap.String("record_type", ops[i].Type),
			zap.String("namespace", ops[i].ResourceRef.Namespace),
		)
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

// rateLimitPerHour reads REGADAPTER_RATE_PER_HOUR from env (Story 5.1).
// Returns 0 (use default) if not set or invalid.
func rateLimitPerHour() float64 {
	s := os.Getenv("REGADAPTER_RATE_PER_HOUR")
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

// DefaultReconcileInterval is the default periodic reconciliation interval.
const DefaultReconcileInterval = 5 * time.Minute

// reconcileIntervalFromEnv reads REGADAPTER_RECONCILE_INTERVAL_SEC from env (Story 8.1).
// Returns DefaultReconcileInterval if not set or invalid.
func reconcileIntervalFromEnv() time.Duration {
	s := os.Getenv("REGADAPTER_RECONCILE_INTERVAL_SEC")
	if s == "" {
		return DefaultReconcileInterval
	}
	sec, err := strconv.Atoi(s)
	if err != nil || sec <= 0 {
		return DefaultReconcileInterval
	}
	return time.Duration(sec) * time.Second
}

// versionMiddleware adds X-Adapter-Version header to all HTTP responses (Story 7.4).
func versionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Adapter-Version", Version)
		next.ServeHTTP(w, r)
	})
}

// extractQuotas builds a quota list from the current config.
// Story 9.2: maps ZoneMapping.QuotaPerHour to per-namespace quotas.
// If multiple zone entries share a namespace, the first non-zero quota wins.
func extractQuotas(cfg *config.MappingsConfig) []quota.NamespaceQuota {
	if cfg == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var quotas []quota.NamespaceQuota
	for _, zm := range cfg.Zones {
		if zm.QuotaPerHour <= 0 {
			continue
		}
		for _, ns := range zm.Namespaces {
			if _, ok := seen[ns]; ok {
				continue
			}
			seen[ns] = struct{}{}
			quotas = append(quotas, quota.NamespaceQuota{
				Namespace:    ns,
				LimitPerHour: zm.QuotaPerHour,
			})
		}
	}
	return quotas
}

// buildQuotaManager creates a quota.Manager from the current config.
func buildQuotaManager(cfg *config.MappingsConfig) *quota.Manager {
	return quota.New(extractQuotas(cfg))
}

func main() {
	a := newApp()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: versionMiddleware(a.mux),
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
	a.logger.Info("worker pool started", zap.Int("workers", concurrency))

	go func() {
		a.logger.Info("sidecar starting", zap.String("addr", ":8080"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Fatal("listen failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	a.logger.Info("sidecar shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		a.logger.Fatal("server forced to shutdown", zap.Error(err))
	}
	a.logger.Info("sidecar stopped")
}
