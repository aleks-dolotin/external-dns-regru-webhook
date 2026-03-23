package queue

import "testing"

func TestNew(t *testing.T) {
	q := New()
	if q == nil {
		t.Fatal("expected non-nil queue")
	}
	if q.Len() != 0 {
		t.Errorf("expected empty queue, got len %d", q.Len())
	}
}

func TestEnqueueDequeue(t *testing.T) {
	q := New()
	q.Enqueue(Operation{ID: "1"})
	q.Enqueue(Operation{ID: "2"})

	if q.Len() != 2 {
		t.Errorf("expected len 2, got %d", q.Len())
	}

	op := q.Dequeue()
	if op == nil || op.ID != "1" {
		t.Errorf("expected op '1', got %v", op)
	}
	if q.Len() != 1 {
		t.Errorf("expected len 1 after dequeue, got %d", q.Len())
	}

	op = q.Dequeue()
	if op == nil || op.ID != "2" {
		t.Errorf("expected op '2', got %v", op)
	}
	if q.Len() != 0 {
		t.Errorf("expected len 0 after dequeuing all, got %d", q.Len())
	}
}

func TestDequeueEmpty(t *testing.T) {
	q := New()
	op := q.Dequeue()
	if op != nil {
		t.Errorf("expected nil from empty queue, got %v", op)
	}
}

func TestLen(t *testing.T) {
	q := New()
	for i := 0; i < 5; i++ {
		q.Enqueue(Operation{ID: "op"})
	}
	if q.Len() != 5 {
		t.Errorf("expected len 5, got %d", q.Len())
	}
	q.Dequeue()
	q.Dequeue()
	if q.Len() != 3 {
		t.Errorf("expected len 3, got %d", q.Len())
	}
}
