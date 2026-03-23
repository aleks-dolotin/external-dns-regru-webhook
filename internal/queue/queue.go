package queue

import "sync"

// Operation is a generic placeholder for queued work
type Operation struct {
    ID   string
    Body interface{}
}

// InMemoryQueue is a simple FIFO queue with mutex (not persistent)
type InMemoryQueue struct {
    mu    sync.Mutex
    items []Operation
}

func New() *InMemoryQueue {
    return &InMemoryQueue{items: make([]Operation, 0)}
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

