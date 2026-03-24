// Package normalizer converts incoming ExternalDNS/K8s events into
// uniform adapter.Operation payloads with unique correlating IDs.
package normalizer

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"go.uber.org/zap"
)

// logger is the package-level structured logger. Defaults to no-op.
// Use SetLogger to inject a real logger (Story 6.4).
var logger = zap.NewNop()

// SetLogger sets the package-level structured logger. Nil reverts to no-op.
func SetLogger(l *zap.Logger) {
	if l == nil {
		l = zap.NewNop()
	}
	logger = l
}

// validActions defines allowed operation actions.
var validActions = map[string]struct{}{
	"create": {},
	"update": {},
	"delete": {},
}

// DNSEndpointEvent represents an incoming ExternalDNS/K8s event
// (Ingress or Service) before normalization.
type DNSEndpointEvent struct {
	SourceEventID string                 `json:"source_event_id,omitempty"`
	ResourceRef   adapter.ResourceRef    `json:"resource_ref"`
	Zone          string                 `json:"zone"`
	FQDN          string                 `json:"fqdn"`
	RecordType    string                 `json:"record_type"`
	Content       string                 `json:"content,omitempty"`
	TTL           int                    `json:"ttl,omitempty"`
	Priority      int                    `json:"priority,omitempty"`
	Action        string                 `json:"action"` // create, update, delete
	K8sMeta       map[string]interface{} `json:"k8s_metadata,omitempty"`
}

// Normalize converts a single DNSEndpointEvent into an adapter.Operation
// with a unique correlating_id (UUID v4). Returns an error if required
// fields are missing or the action is invalid.
func Normalize(event DNSEndpointEvent) (adapter.Operation, error) {
	if event.Zone == "" {
		return adapter.Operation{}, fmt.Errorf("normalizer: zone is required")
	}
	if event.FQDN == "" {
		return adapter.Operation{}, fmt.Errorf("normalizer: fqdn is required")
	}
	if event.RecordType == "" {
		return adapter.Operation{}, fmt.Errorf("normalizer: record_type is required")
	}
	if event.Action == "" {
		return adapter.Operation{}, fmt.Errorf("normalizer: action is required")
	}
	if _, ok := validActions[event.Action]; !ok {
		return adapter.Operation{}, fmt.Errorf("normalizer: invalid action %q (allowed: create, update, delete)", event.Action)
	}

	opID, err := generateUUID()
	if err != nil {
		return adapter.Operation{}, fmt.Errorf("normalizer: failed to generate correlating_id: %w", err)
	}

	// Story 6.4: log normalization step with correlating_id for end-to-end tracing.
	logger.Debug("event normalized",
		zap.String("correlating_id", opID),
		zap.String("zone", event.Zone),
		zap.String("operation", event.Action),
		zap.String("fqdn", event.FQDN),
		zap.String("record_type", event.RecordType),
		zap.String("resource", event.ResourceRef.Name),
		zap.String("namespace", event.ResourceRef.Namespace),
	)

	return adapter.Operation{
		OpID:        opID,
		ResourceRef: event.ResourceRef,
		Action:      event.Action,
		Zone:        event.Zone,
		Name:        event.FQDN,
		Type:        event.RecordType,
		Content:     event.Content,
		TTL:         event.TTL,
		Priority:    event.Priority,
		Timestamp:   time.Now().UTC(),
		SourceEvent: event.SourceEventID,
		K8sMeta:     event.K8sMeta,
	}, nil
}

// NormalizeBatch converts a slice of events into operations. Valid events
// produce operations; invalid events produce errors. Both slices are returned
// so that partial failures do not block the entire batch.
func NormalizeBatch(events []DNSEndpointEvent) ([]adapter.Operation, []error) {
	if len(events) == 0 {
		return nil, nil
	}

	ops := make([]adapter.Operation, 0, len(events))
	var errs []error

	for i := range events {
		op, err := Normalize(events[i])
		if err != nil {
			errs = append(errs, fmt.Errorf("event[%d]: %w", i, err))
			continue
		}
		ops = append(ops, op)
	}
	return ops, errs
}

// generateUUID generates a random UUID v4 using crypto/rand.
func generateUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
