// Package reconciler compares desired DNS state with actual Reg.ru state
// and emits corrective operations to repair drift.
package reconciler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/normalizer"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
)

// DesiredRecord represents a DNS record that should exist in Reg.ru.
type DesiredRecord struct {
	Zone       string
	FQDN       string
	RecordType string
	Content    string
	TTL        int
}

// RecordFetcher retrieves actual DNS records for a zone from Reg.ru.
type RecordFetcher interface {
	FindRecord(zone, name, typ string) (*adapter.Record, error)
}

// DriftAction describes a corrective action produced by reconciliation.
type DriftAction struct {
	Action string // "create", "update", "delete"
	Record DesiredRecord
	Reason string
}

// Reconciler compares desired state with actual state and enqueues corrections.
type Reconciler struct {
	fetcher  RecordFetcher
	queue    *queue.InMemoryQueue
	interval time.Duration
}

// New creates a Reconciler with the given fetcher, queue, and check interval.
func New(fetcher RecordFetcher, q *queue.InMemoryQueue, interval time.Duration) *Reconciler {
	return &Reconciler{
		fetcher:  fetcher,
		queue:    q,
		interval: interval,
	}
}

// Reconcile performs a single reconciliation pass: compares desired records
// with actual records and returns corrective actions. If enqueue is true,
// corrective operations are also enqueued.
func (r *Reconciler) Reconcile(ctx context.Context, desired []DesiredRecord, enqueue bool) ([]DriftAction, error) {
	var actions []DriftAction

	for _, d := range desired {
		select {
		case <-ctx.Done():
			return actions, ctx.Err()
		default:
		}

		actual, err := r.fetcher.FindRecord(d.Zone, d.FQDN, d.RecordType)
		if err != nil {
			return actions, fmt.Errorf("reconciler: fetching %s/%s/%s: %w", d.Zone, d.FQDN, d.RecordType, err)
		}

		if actual == nil {
			// Missing record — create
			action := DriftAction{
				Action: "create",
				Record: d,
				Reason: "record missing in Reg.ru",
			}
			actions = append(actions, action)
			if enqueue {
				r.enqueueAction(action)
			}
			continue
		}

		// Record exists — check for content/TTL drift
		if actual.Content != d.Content || (d.TTL > 0 && actual.TTL != d.TTL) {
			action := DriftAction{
				Action: "update",
				Record: d,
				Reason: fmt.Sprintf("drift detected: content=%q→%q ttl=%d→%d",
					actual.Content, d.Content, actual.TTL, d.TTL),
			}
			actions = append(actions, action)
			if enqueue {
				r.enqueueAction(action)
			}
		}
	}

	return actions, nil
}

// enqueueAction normalizes a drift action and pushes it to the queue.
func (r *Reconciler) enqueueAction(da DriftAction) {
	event := normalizer.DNSEndpointEvent{
		Zone:       da.Record.Zone,
		FQDN:       da.Record.FQDN,
		RecordType: da.Record.RecordType,
		Content:    da.Record.Content,
		TTL:        da.Record.TTL,
		Action:     da.Action,
	}
	op, err := normalizer.Normalize(event)
	if err != nil {
		log.Printf("reconciler: failed to normalize corrective action: %v", err)
		return
	}
	r.queue.Enqueue(queue.Operation{ID: op.OpID, Body: op})
	log.Printf("reconciler: enqueued %s for %s/%s (%s)", da.Action, da.Record.Zone, da.Record.FQDN, da.Reason)
}

// RunPeriodic starts a background reconciliation loop that runs at the
// configured interval. It blocks until the context is cancelled.
func (r *Reconciler) RunPeriodic(ctx context.Context, desired func() []DesiredRecord) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("reconciler: stopping periodic loop")
			return
		case <-ticker.C:
			records := desired()
			if len(records) == 0 {
				continue
			}
			actions, err := r.Reconcile(ctx, records, true)
			if err != nil {
				log.Printf("reconciler: periodic run error: %v", err)
				continue
			}
			if len(actions) > 0 {
				log.Printf("reconciler: periodic run found %d drift actions", len(actions))
			}
		}
	}
}
