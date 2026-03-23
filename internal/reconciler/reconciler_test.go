package reconciler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
)

// mockFetcher implements RecordFetcher for testing.
type mockFetcher struct {
	records map[string]*adapter.Record // key: "zone/fqdn/type"
	err     error
}

func (m *mockFetcher) FindRecord(zone, name, typ string) (*adapter.Record, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := zone + "/" + name + "/" + typ
	return m.records[key], nil
}

func TestReconcile_MissingRecord_Creates(t *testing.T) {
	f := &mockFetcher{records: map[string]*adapter.Record{}}
	q := queue.New()
	r := New(f, q, time.Minute)

	desired := []DesiredRecord{
		{Zone: "example.com", FQDN: "app.example.com", RecordType: "A", Content: "1.2.3.4", TTL: 300},
	}

	actions, err := r.Reconcile(context.Background(), desired, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Action != "create" {
		t.Errorf("expected 'create', got %q", actions[0].Action)
	}
	if actions[0].Reason == "" {
		t.Error("expected non-empty reason")
	}
	// Should have enqueued
	if q.Len() != 1 {
		t.Errorf("expected 1 queued item, got %d", q.Len())
	}
}

func TestReconcile_ContentDrift_Updates(t *testing.T) {
	f := &mockFetcher{records: map[string]*adapter.Record{
		"example.com/app.example.com/A": {Name: "app.example.com", Type: "A", Content: "9.9.9.9", TTL: 300},
	}}
	q := queue.New()
	r := New(f, q, time.Minute)

	desired := []DesiredRecord{
		{Zone: "example.com", FQDN: "app.example.com", RecordType: "A", Content: "1.2.3.4", TTL: 300},
	}

	actions, err := r.Reconcile(context.Background(), desired, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Action != "update" {
		t.Errorf("expected 'update', got %q", actions[0].Action)
	}
	if q.Len() != 1 {
		t.Errorf("expected 1 queued item, got %d", q.Len())
	}
}

func TestReconcile_TTLDrift_Updates(t *testing.T) {
	f := &mockFetcher{records: map[string]*adapter.Record{
		"example.com/app.example.com/A": {Name: "app.example.com", Type: "A", Content: "1.2.3.4", TTL: 60},
	}}
	q := queue.New()
	r := New(f, q, time.Minute)

	desired := []DesiredRecord{
		{Zone: "example.com", FQDN: "app.example.com", RecordType: "A", Content: "1.2.3.4", TTL: 300},
	}

	actions, err := r.Reconcile(context.Background(), desired, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action for TTL drift, got %d", len(actions))
	}
	if actions[0].Action != "update" {
		t.Errorf("expected 'update', got %q", actions[0].Action)
	}
}

func TestReconcile_NoDrift_NoActions(t *testing.T) {
	f := &mockFetcher{records: map[string]*adapter.Record{
		"example.com/app.example.com/A": {Name: "app.example.com", Type: "A", Content: "1.2.3.4", TTL: 300},
	}}
	q := queue.New()
	r := New(f, q, time.Minute)

	desired := []DesiredRecord{
		{Zone: "example.com", FQDN: "app.example.com", RecordType: "A", Content: "1.2.3.4", TTL: 300},
	}

	actions, err := r.Reconcile(context.Background(), desired, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(actions))
	}
	if q.Len() != 0 {
		t.Errorf("expected empty queue, got %d", q.Len())
	}
}

func TestReconcile_FetcherError(t *testing.T) {
	f := &mockFetcher{err: errors.New("api unavailable")}
	q := queue.New()
	r := New(f, q, time.Minute)

	desired := []DesiredRecord{
		{Zone: "example.com", FQDN: "app.example.com", RecordType: "A", Content: "1.2.3.4"},
	}

	_, err := r.Reconcile(context.Background(), desired, true)
	if err == nil {
		t.Fatal("expected error from fetcher")
	}
}

func TestReconcile_ContextCancelled(t *testing.T) {
	f := &mockFetcher{records: map[string]*adapter.Record{}}
	q := queue.New()
	r := New(f, q, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	desired := []DesiredRecord{
		{Zone: "a.com", FQDN: "x.a.com", RecordType: "A", Content: "1.1.1.1"},
	}

	_, err := r.Reconcile(ctx, desired, true)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestReconcile_EnqueueFalse_NoQueue(t *testing.T) {
	f := &mockFetcher{records: map[string]*adapter.Record{}}
	q := queue.New()
	r := New(f, q, time.Minute)

	desired := []DesiredRecord{
		{Zone: "a.com", FQDN: "x.a.com", RecordType: "A", Content: "1.1.1.1"},
	}

	actions, err := r.Reconcile(context.Background(), desired, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if q.Len() != 0 {
		t.Errorf("expected empty queue with enqueue=false, got %d", q.Len())
	}
}

func TestReconcile_MultipleDrifts(t *testing.T) {
	f := &mockFetcher{records: map[string]*adapter.Record{
		"a.com/x.a.com/A": {Name: "x.a.com", Type: "A", Content: "old", TTL: 300},
		// y.a.com missing
	}}
	q := queue.New()
	r := New(f, q, time.Minute)

	desired := []DesiredRecord{
		{Zone: "a.com", FQDN: "x.a.com", RecordType: "A", Content: "new", TTL: 300},
		{Zone: "a.com", FQDN: "y.a.com", RecordType: "CNAME", Content: "lb.a.com"},
	}

	actions, err := r.Reconcile(context.Background(), desired, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	if actions[0].Action != "update" {
		t.Errorf("expected update for x.a.com, got %q", actions[0].Action)
	}
	if actions[1].Action != "create" {
		t.Errorf("expected create for y.a.com, got %q", actions[1].Action)
	}
	if q.Len() != 2 {
		t.Errorf("expected 2 queued items, got %d", q.Len())
	}
}

func TestReconcile_EmptyDesired(t *testing.T) {
	f := &mockFetcher{records: map[string]*adapter.Record{}}
	q := queue.New()
	r := New(f, q, time.Minute)

	actions, err := r.Reconcile(context.Background(), nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected 0 actions for empty desired, got %d", len(actions))
	}
}

func TestRunPeriodic_Cancels(t *testing.T) {
	f := &mockFetcher{records: map[string]*adapter.Record{}}
	q := queue.New()
	r := New(f, q, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		r.RunPeriodic(ctx, func() []DesiredRecord { return nil })
		close(done)
	}()

	select {
	case <-done:
		// OK — periodic loop stopped
	case <-time.After(2 * time.Second):
		t.Fatal("RunPeriodic did not stop after context cancel")
	}
}

func TestRunPeriodic_EnqueuesDrift(t *testing.T) {
	f := &mockFetcher{records: map[string]*adapter.Record{}}
	q := queue.New()
	r := New(f, q, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	desired := func() []DesiredRecord {
		return []DesiredRecord{
			{Zone: "z.com", FQDN: "a.z.com", RecordType: "A", Content: "1.1.1.1"},
		}
	}

	r.RunPeriodic(ctx, desired)

	if q.Len() == 0 {
		t.Error("expected at least 1 queued corrective operation from periodic reconciliation")
	}
}
