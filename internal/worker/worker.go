package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/metrics"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/retry"
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
	wg          sync.WaitGroup
	stop        chan struct{}
	workerCount int32 // atomic: running workers

	mu            sync.RWMutex
	lastHeartbeat time.Time            // last successful operation time
	lastErrors    map[string]ZoneError // last error per zone
	failedOps     []FailedOp           // dead-letter list
}

// New creates a WorkerPool with the default retry policy.
func New(adapter adapter.Adapter, q *queue.InMemoryQueue) *WorkerPool {
	return &WorkerPool{
		adapter:     adapter,
		queue:       q,
		retryPolicy: retry.DefaultPolicy,
		stop:        make(chan struct{}),
		lastErrors:  make(map[string]ZoneError),
	}
}

// SetRetryPolicy overrides the default retry policy.
func (p *WorkerPool) SetRetryPolicy(pol retry.Policy) {
	p.retryPolicy = pol
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
	defer atomic.AddInt32(&p.workerCount, -1)
	log.Printf("worker %d started", id)
	for {
		select {
		case <-ctx.Done():
			log.Printf("worker %d stopping: context done", id)
			return
		case <-p.stop:
			log.Printf("worker %d stopping: stop signal", id)
			return
		default:
			op := p.queue.Dequeue()
			if op == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			p.handleWithRetry(ctx, op)
		}
	}
}

// handleWithRetry processes an operation with the configured retry policy.
func (p *WorkerPool) handleWithRetry(ctx context.Context, op *queue.Operation) {
	adapterOp, ok := op.Body.(adapter.Operation)
	if !ok {
		log.Printf("worker: skipping op %s: body is not adapter.Operation (type %T)", op.ID, op.Body)
		return
	}

	result := p.retryPolicy.Do(ctx, func() error {
		return p.dispatch(adapterOp)
	}, func(attempt int, err error) {
		metrics.RetriesTotal.Inc()
		log.Printf("worker: retrying op %s (attempt %d): %v", op.ID, attempt, err)
	})

	if result.Success {
		p.mu.Lock()
		p.lastHeartbeat = time.Now().UTC()
		p.mu.Unlock()
		log.Printf("worker: op %s succeeded after %d attempt(s)", op.ID, result.Attempts)
	} else {
		// Dead-letter: max retries exhausted
		log.Printf("worker: op %s FAILED after %d attempts: %v", op.ID, result.Attempts, result.Err)
		metrics.FailedOpsTotal.WithLabelValues(adapterOp.Action).Inc()
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
