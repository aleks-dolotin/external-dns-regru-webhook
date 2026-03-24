package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/audit"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/logging"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/metrics"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/ratelimit"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/retry"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// ZoneError holds the last error information for a zone.
type ZoneError struct {
	Zone    string
	Message string
	Time    time.Time
}

// FailedOp is an operation that exhausted all retry attempts (dead-letter).
type FailedOp struct {
	Op       adapter.Operation
	Err      error
	Attempts int
	Time     time.Time
}

// WorkerPool processes queued operations using an adapter with retry support.
type WorkerPool struct {
	adapter     adapter.Adapter
	queue       *queue.InMemoryQueue
	retryPolicy retry.Policy
	limiter     *ratelimit.ZoneLimiter // Story 5.1: per-zone rate limiter (nil = disabled)
	breaker     CircuitBreakers        // Story 5.4: per-zone circuit breaker (nil = disabled)
	safeMode    SafeModeChecker        // Story 8.3: safe-mode toggle (nil = disabled)
	logger      *zap.Logger            // Story 6.2: structured logger (nil = nop)
	auditor     audit.Auditor          // Story 6.3: audit trail (nil = nop)
	wg          sync.WaitGroup
	stop        chan struct{}
	workerCount int32 // atomic: running workers

	mu            sync.RWMutex
	lastHeartbeat time.Time            // last successful operation time
	lastErrors    map[string]ZoneError // last error per zone
	failedOps     []FailedOp           // dead-letter list
}

// CircuitBreakers is an interface for per-zone circuit breaker checks.
// Implemented by circuitbreaker.Manager (Story 5.4).
type CircuitBreakers interface {
	// AllowRequest returns nil if the circuit for the zone is closed/half-open,
	// or an error if the circuit is open.
	AllowRequest(zone string) error
	// RecordSuccess records a successful operation for the zone.
	RecordSuccess(zone string)
	// RecordFailure records a failed operation for the zone.
	RecordFailure(zone string)
}

// SafeModeChecker checks whether safe-mode is active.
// Implemented by safemode.SafeMode (Story 8.3).
type SafeModeChecker interface {
	IsEnabled() bool
	IncrementSuppressed()
}

// New creates a WorkerPool with the default retry policy.
func New(adapter adapter.Adapter, q *queue.InMemoryQueue) *WorkerPool {
	return &WorkerPool{
		adapter:     adapter,
		queue:       q,
		retryPolicy: retry.DefaultPolicy,
		logger:      zap.NewNop(),
		auditor:     audit.Nop(),
		stop:        make(chan struct{}),
		lastErrors:  make(map[string]ZoneError),
	}
}

// SetLogger attaches a structured logger (Story 6.2). Nil reverts to no-op.
func (p *WorkerPool) SetLogger(l *zap.Logger) {
	if l == nil {
		l = zap.NewNop()
	}
	p.logger = l
}

// SetAuditor attaches an audit recorder (Story 6.3). Nil reverts to no-op.
func (p *WorkerPool) SetAuditor(a audit.Auditor) {
	if a == nil {
		a = audit.Nop()
	}
	p.auditor = a
}

// SetRetryPolicy overrides the default retry policy.
func (p *WorkerPool) SetRetryPolicy(pol retry.Policy) {
	p.retryPolicy = pol
}

// SetLimiter attaches a per-zone rate limiter (Story 5.1).
func (p *WorkerPool) SetLimiter(lim *ratelimit.ZoneLimiter) {
	p.limiter = lim
}

// SetCircuitBreakers attaches per-zone circuit breakers (Story 5.4).
func (p *WorkerPool) SetCircuitBreakers(cb CircuitBreakers) {
	p.breaker = cb
}

// SetSafeMode attaches a safe-mode checker (Story 8.3).
func (p *WorkerPool) SetSafeMode(sm SafeModeChecker) {
	p.safeMode = sm
}

// WorkerCount returns the number of currently running workers.
func (p *WorkerPool) WorkerCount() int {
	return int(atomic.LoadInt32(&p.workerCount))
}

// LastHeartbeat returns the time of the last successful operation.
func (p *WorkerPool) LastHeartbeat() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastHeartbeat
}

// LastErrors returns a copy of the last error per zone.
func (p *WorkerPool) LastErrors() map[string]ZoneError {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cp := make(map[string]ZoneError, len(p.lastErrors))
	for k, v := range p.lastErrors {
		cp[k] = v
	}
	return cp
}

// FailedOps returns a copy of the dead-letter list.
func (p *WorkerPool) FailedOps() []FailedOp {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cp := make([]FailedOp, len(p.failedOps))
	copy(cp, p.failedOps)
	return cp
}

// ReplayOp finds a failed operation by ID, removes it from the dead-letter list,
// and returns it for re-enqueue. Returns an error if the operation is not found.
// Story 8.2: replay single failed operation.
func (p *WorkerPool) ReplayOp(opID string) (adapter.Operation, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, fo := range p.failedOps {
		if fo.Op.OpID == opID {
			op := fo.Op
			// Remove from dead-letter list.
			p.failedOps = append(p.failedOps[:i], p.failedOps[i+1:]...)
			return op, nil
		}
	}
	return adapter.Operation{}, fmt.Errorf("operation %s not found in failed list", opID)
}

// ReplayAll removes all operations from the dead-letter list and returns them.
// Story 8.2: replay all failed operations.
func (p *WorkerPool) ReplayAll() []adapter.Operation {
	p.mu.Lock()
	defer p.mu.Unlock()

	ops := make([]adapter.Operation, len(p.failedOps))
	for i, fo := range p.failedOps {
		ops[i] = fo.Op
	}
	p.failedOps = p.failedOps[:0] // clear without reallocating
	return ops
}

// Start launches the given number of worker goroutines.
func (p *WorkerPool) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		atomic.AddInt32(&p.workerCount, 1)
		go p.workerLoop(ctx, i)
	}
}

func (p *WorkerPool) workerLoop(ctx context.Context, id int) {
	defer p.wg.Done()
	defer func() {
		atomic.AddInt32(&p.workerCount, -1)
		// Story 6.1: update gauge after worker exits.
		metrics.WorkerCountGauge.Set(float64(atomic.LoadInt32(&p.workerCount)))
	}()
	p.logger.Info("worker started", zap.Int("worker_id", id))
	// Story 6.1: update gauge after worker starts.
	metrics.WorkerCountGauge.Set(float64(atomic.LoadInt32(&p.workerCount)))
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("worker stopping: context done", zap.Int("worker_id", id))
			return
		case <-p.stop:
			p.logger.Info("worker stopping: stop signal", zap.Int("worker_id", id))
			return
		default:
			op := p.queue.Dequeue()
			if op == nil {
				// Story 6.1: update queue depth gauge even on empty poll.
				metrics.QueueDepth.Set(float64(p.queue.Len()))
				time.Sleep(100 * time.Millisecond)
				continue
			}
			// Story 6.1: update queue depth after dequeue.
			metrics.QueueDepth.Set(float64(p.queue.Len()))
			// Story 6.4: log dequeue with correlating_id.
			p.logger.Debug("operation dequeued", zap.String("correlating_id", op.ID))
			p.handleWithRetry(ctx, op)
		}
	}
}

// handleWithRetry processes an operation with the configured retry policy.
func (p *WorkerPool) handleWithRetry(ctx context.Context, op *queue.Operation) {
	adapterOp, ok := op.Body.(adapter.Operation)
	if !ok {
		p.logger.Warn("skipping operation: body is not adapter.Operation",
			zap.String("op_id", op.ID),
			zap.String("body_type", fmt.Sprintf("%T", op.Body)),
		)
		return
	}

	// Story 6.2: build per-operation logger with correlating_id and structured fields.
	opLog := p.logger.With(
		zap.String("correlating_id", adapterOp.OpID),
		zap.String("zone", adapterOp.Zone),
		zap.String("operation", adapterOp.Action),
		zap.String("resource", adapterOp.ResourceRef.Name),
		zap.String("namespace", adapterOp.ResourceRef.Namespace),
		zap.String("record_type", adapterOp.Type),
	)

	// Story 8.3: safe-mode check — suppress writes, re-enqueue after delay.
	if p.safeMode != nil && p.safeMode.IsEnabled() {
		p.safeMode.IncrementSuppressed()
		opLog.Info("safe-mode: write suppressed, re-enqueuing")
		time.Sleep(5 * time.Second)
		p.queue.Enqueue(*op)
		return
	}

	// Story 6.4 (review fix): propagate structured logger into context so that
	// downstream layers (adapter, dispatch) can extract it via logging.FromContext.
	ctx = logging.WithContext(ctx, opLog)

	// Story 5.4: check circuit breaker before processing.
	if p.breaker != nil {
		if err := p.breaker.AllowRequest(adapterOp.Zone); err != nil {
			opLog.Warn("rejected by circuit breaker", zap.Error(err))
			p.recordZoneError(adapterOp.Zone, err)
			// Re-enqueue for later retry when circuit closes.
			p.queue.Enqueue(*op)
			return
		}
	}

	// Story 5.1: enforce per-zone rate limit before dispatch.
	if p.limiter != nil {
		if err := p.limiter.Wait(ctx, adapterOp.Zone); err != nil {
			opLog.Warn("rate-limited", zap.Error(err))
			metrics.RateLimitedTotal.WithLabelValues(adapterOp.Zone, adapterOp.ResourceRef.Namespace).Inc()
			// Re-enqueue so the operation is not lost.
			p.queue.Enqueue(*op)
			return
		}
	}

	// Story 5.2: use DoWithRetryAfter to honor Retry-After headers and record backoff metrics.
	var lastRetryAfterWait time.Duration
	result := p.retryPolicy.DoWithRetryAfter(ctx, func() error {
		lastRetryAfterWait = 0 // reset per attempt
		// Story 6.1: wrap dispatch with timer for RequestDurationV2.
		start := time.Now()
		// Story 6.4: log dispatch start for tracing.
		opLog.Debug("dispatching to adapter")
		err := p.dispatch(adapterOp)
		duration := time.Since(start).Seconds()

		// Story 6.1: record V2 duration (every attempt, regardless of outcome).
		// Story 6.4: attach correlating_id as exemplar for request-level tracing.
		observer := metrics.RequestDurationV2.WithLabelValues(adapterOp.Zone, adapterOp.Type, adapterOp.Action)
		if eo, ok := observer.(prometheus.ExemplarObserver); ok {
			eo.ObserveWithExemplar(duration, prometheus.Labels{"correlating_id": adapterOp.OpID})
		} else {
			observer.Observe(duration)
		}

		if err != nil {
			// Extract Retry-After duration from 429/503 responses (Story 5.2).
			var raErr *adapter.RetryAfterError
			if errors.As(err, &raErr) && raErr.Wait > 0 {
				lastRetryAfterWait = raErr.Wait
			}
		}
		return err
	}, func() time.Duration {
		return lastRetryAfterWait
	}, func(attempt int, err error) {
		reason := "backoff"
		if lastRetryAfterWait > 0 {
			reason = "retry-after-header"
		}
		metrics.RetriesTotal.Inc()
		metrics.APIRetriesTotal.WithLabelValues(adapterOp.Zone, reason).Inc()
		opLog.Warn("retrying operation",
			zap.Int("attempt", attempt),
			zap.String("reason", reason),
			zap.Error(err),
		)
	}, func(wait time.Duration) {
		metrics.APIBackoffSeconds.WithLabelValues(adapterOp.Zone).Observe(wait.Seconds())
	})

	if result.Success {
		// Story 5.4: record success for circuit breaker.
		if p.breaker != nil {
			p.breaker.RecordSuccess(adapterOp.Zone)
		}
		// Story 6.1: record V2 request counter (success).
		metrics.RequestsTotalV2.WithLabelValues(adapterOp.Zone, adapterOp.Type, adapterOp.Action, "success", adapterOp.ResourceRef.Namespace).Inc()
		p.mu.Lock()
		p.lastHeartbeat = time.Now().UTC()
		p.mu.Unlock()
		opLog.Info("operation succeeded", zap.Int("attempts", result.Attempts))
		// Story 6.3: record audit event on success.
		p.auditor.Record(audit.AuditEvent{
			Timestamp:     time.Now().UTC(),
			Operation:     adapterOp.Action,
			Actor:         "system",
			Zone:          adapterOp.Zone,
			FQDN:          adapterOp.Name,
			RecordType:    adapterOp.Type,
			CorrelatingID: adapterOp.OpID,
			Result:        "success",
		})
	} else {
		// Story 5.4: record failure for circuit breaker.
		if p.breaker != nil {
			p.breaker.RecordFailure(adapterOp.Zone)
		}
		// Story 6.1: record V2 request counter (failure).
		metrics.RequestsTotalV2.WithLabelValues(adapterOp.Zone, adapterOp.Type, adapterOp.Action, "failure", adapterOp.ResourceRef.Namespace).Inc()
		// Dead-letter: max retries exhausted
		opLog.Error("operation FAILED",
			zap.Int("attempts", result.Attempts),
			zap.Error(result.Err),
		)
		// Story 6.3: record audit event on failure.
		errDetail := ""
		if result.Err != nil {
			errDetail = result.Err.Error()
		}
		p.auditor.Record(audit.AuditEvent{
			Timestamp:     time.Now().UTC(),
			Operation:     adapterOp.Action,
			Actor:         "system",
			Zone:          adapterOp.Zone,
			FQDN:          adapterOp.Name,
			RecordType:    adapterOp.Type,
			CorrelatingID: adapterOp.OpID,
			Result:        "failure",
			ErrorDetail:   errDetail,
		})
		metrics.FailedOpsTotal.WithLabelValues(adapterOp.Action, adapterOp.ResourceRef.Namespace).Inc()
		p.recordZoneError(adapterOp.Zone, result.Err)
		p.mu.Lock()
		p.failedOps = append(p.failedOps, FailedOp{
			Op:       adapterOp,
			Err:      result.Err,
			Attempts: result.Attempts,
			Time:     time.Now().UTC(),
		})
		p.mu.Unlock()
	}
}

// dispatch routes an adapter.Operation to the appropriate adapter method.
func (p *WorkerPool) dispatch(op adapter.Operation) error {
	if p.adapter == nil {
		return fmt.Errorf("adapter not configured")
	}

	switch op.Action {
	case "create":
		return p.adapter.CreateRecord(op.Zone, &adapter.Record{
			Name:     op.Name,
			Type:     op.Type,
			Content:  op.Content,
			TTL:      op.TTL,
			Priority: op.Priority,
		})
	case "update":
		return p.adapter.UpdateRecord(op.Zone, &adapter.Record{
			ID:       op.OpID,
			Name:     op.Name,
			Type:     op.Type,
			Content:  op.Content,
			TTL:      op.TTL,
			Priority: op.Priority,
		})
	case "delete":
		return p.adapter.DeleteRecord(op.Zone, op.OpID)
	default:
		return fmt.Errorf("unknown action: %s", op.Action)
	}
}

// recordZoneError stores the last error for a specific zone.
func (p *WorkerPool) recordZoneError(zone string, err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastErrors[zone] = ZoneError{
		Zone:    zone,
		Message: err.Error(),
		Time:    time.Now().UTC(),
	}
}

// Stop signals all workers to halt and waits for them to finish.
func (p *WorkerPool) Stop() {
	close(p.stop)
	p.wg.Wait()
}

// InjectFailedOp adds a FailedOp to the dead-letter list. Test helper only.
func (p *WorkerPool) InjectFailedOp(fo FailedOp) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failedOps = append(p.failedOps, fo)
}
