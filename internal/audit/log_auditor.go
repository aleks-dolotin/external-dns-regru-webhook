package audit

import (
	"go.uber.org/zap"
)

// LogAuditor writes audit events to a structured logger with an `audit=true`
// marker field, making them queryable via Loki/ELK queries like `{audit="true"}`.
type LogAuditor struct {
	logger *zap.Logger
}

// NewLogAuditor creates an auditor that writes to the given zap logger.
// A child logger with `audit=true` is created automatically.
func NewLogAuditor(logger *zap.Logger) *LogAuditor {
	return &LogAuditor{
		logger: logger.With(zap.Bool("audit", true)),
	}
}

// Record emits an audit event as a structured log entry.
func (a *LogAuditor) Record(event AuditEvent) {
	fields := []zap.Field{
		zap.Time("timestamp", event.Timestamp),
		zap.String("operation", event.Operation),
		zap.String("actor", event.Actor),
		zap.String("zone", event.Zone),
		zap.String("fqdn", event.FQDN),
		zap.String("record_type", event.RecordType),
		zap.String("correlating_id", event.CorrelatingID),
		zap.String("result", event.Result),
	}
	if event.ErrorDetail != "" {
		fields = append(fields, zap.String("error_detail", event.ErrorDetail))
	}
	a.logger.Info("audit_event", fields...)
}
