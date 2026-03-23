package worker

import (
    "context"
    "log"
    "sync"
    "time"

    "github.com/yourorg/externaldns-regru-sidecar/internal/adapter"
    "github.com/yourorg/externaldns-regru-sidecar/internal/queue"
)

type WorkerPool struct {
    adapter adapter.Adapter
    queue   *queue.InMemoryQueue
    wg      sync.WaitGroup
    stop    chan struct{}
}

func New(adapter adapter.Adapter, q *queue.InMemoryQueue) *WorkerPool {
    return &WorkerPool{adapter: adapter, queue: q, stop: make(chan struct{})}
}

func (p *WorkerPool) Start(ctx context.Context, workers int) {
    for i := 0; i < workers; i++ {
        p.wg.Add(1)
        go p.workerLoop(ctx, i)
    }
}

func (p *WorkerPool) workerLoop(ctx context.Context, id int) {
    defer p.wg.Done()
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
            // TODO: cast op.Body and handle appropriately
            _ = p.handle(op)
        }
    }
}

func (p *WorkerPool) handle(op *queue.Operation) error {
    // Placeholder: actual handling will use adapter
    // For now just log
    log.Printf("handling op %s", op.ID)
    return nil
}

func (p *WorkerPool) Stop() {
    close(p.stop)
    p.wg.Wait()
}

