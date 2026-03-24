// Package audit provides structured audit trail for DNS operations.
// LogAuditor writes events via zap structured logger with an audit=true marker,
// making them queryable in the cluster log aggregation system (Loki/ELK).
package audit

import "time"

// AuditEvent represents a single auditable DNS operation.
type AuditEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	Operation     string    `json:"operation"` // create, update, delete
	Actor         string    `json:"actor"`     // service identity or "system"
	Zone          string    `json:"zone"`
	FQDN          string    `json:"fqdn"`
	RecordType    string    `json:"record_type"`
	CorrelatingID string    `json:"correlating_id"`
	Result        string    `json:"result"` // success, failure
	ErrorDetail   string    `json:"error_detail,omitempty"`
}

// Auditor records audit events. Implementations must be safe for concurrent use.
type Auditor interface {
	// Record persists an audit event. Implementations must not panic on failure.
	Record(event AuditEvent)
}

// nopAuditor silently discards events. Used as fallback when no auditor is configured.
type nopAuditor struct{}

func (nopAuditor) Record(_ AuditEvent) {}

// Nop returns an Auditor that discards all events.
func Nop() Auditor { return nopAuditor{} }
