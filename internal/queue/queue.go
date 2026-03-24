package queue

import "sync"

// DefaultBackpressureThreshold is the default queue depth that triggers backpressure.
const DefaultBackpressureThreshold = 100

// Operation is a generic placeholder for queued work
type Operation struct {
	ID   string
	Body interface{}
}

// InMemoryQueue is a simple FIFO queue with mutex (not persistent)
type InMemoryQueue struct {
	mu                    sync.Mutex
	items                 []Operation
	backpressureThreshold int // Story 5.5: queue depth that triggers backpressure
}

func New() *InMemoryQueue {
	return &InMemoryQueue{
		items:                 make([]Operation, 0),
		backpressureThreshold: DefaultBackpressureThreshold,
	}
}

// SetBackpressureThreshold configures the depth at which backpressure is signalled.
func (q *InMemoryQueue) SetBackpressureThreshold(n int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if n > 0 {
		q.backpressureThreshold = n
	}
}

func (q *InMemoryQueue) Enqueue(op Operation) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, op)
}

func (q *InMemoryQueue) Dequeue() *Operation {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	op := q.items[0]
	q.items = q.items[1:]
	return &op
}

// Len returns the current number of items in the queue.
func (q *InMemoryQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// IsBackpressured returns true when queue depth exceeds the backpressure threshold (Story 5.5).
func (q *InMemoryQueue) IsBackpressured() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items) >= q.backpressureThreshold
}
