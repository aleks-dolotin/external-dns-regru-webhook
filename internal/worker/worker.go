package worker

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
)

// ZoneError holds the last error information for a zone.
type ZoneError struct {
	Zone    string
	Message string
	Time    time.Time
}

type WorkerPool struct {
	adapter     adapter.Adapter
	queue       *queue.InMemoryQueue
	wg          sync.WaitGroup
	stop        chan struct{}
	workerCount int32 // atomic: running workers

	mu            sync.RWMutex
	lastHeartbeat time.Time            // last successful operation time
	lastErrors    map[string]ZoneError // last error per zone
}

func New(adapter adapter.Adapter, q *queue.InMemoryQueue) *WorkerPool {
	return &WorkerPool{
		adapter:    adapter,
		queue:      q,
		stop:       make(chan struct{}),
		lastErrors: make(map[string]ZoneError),
	}
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
			_ = p.handle(op)
		}
	}
}

func (p *WorkerPool) handle(op *queue.Operation) error {
	// Placeholder: actual handling will use adapter
	// For now just log
	log.Printf("handling op %s", op.ID)

	// Record heartbeat for successful operations
	p.mu.Lock()
	p.lastHeartbeat = time.Now().UTC()
	p.mu.Unlock()

	return nil
}

// recordZoneError stores the last error for a specific zone.
func (p *WorkerPool) recordZoneError(zone string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastErrors[zone] = ZoneError{
		Zone:    zone,
		Message: err.Error(),
		Time:    time.Now().UTC(),
	}
}

func (p *WorkerPool) Stop() {
	close(p.stop)
	p.wg.Wait()
}
