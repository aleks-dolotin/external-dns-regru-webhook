package audit

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func sampleEvent() AuditEvent {
	return AuditEvent{
		Timestamp:     time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC),
		Operation:     "create",
		Actor:         "system",
		Zone:          "example.com",
		FQDN:          "app.example.com",
		RecordType:    "A",
		CorrelatingID: "uuid-abc-123",
		Result:        "success",
	}
}

// --- LogAuditor tests ---

func TestLogAuditor_EmitsStructuredLogWithAuditMarker(t *testing.T) {
	var buf bytes.Buffer
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:     "timestamp",
		LevelKey:    "level",
		MessageKey:  "msg",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
		EncodeTime:  zapcore.ISO8601TimeEncoder,
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(&buf),
		zapcore.DebugLevel,
	)
	logger := zap.New(core)

	auditor := NewLogAuditor(logger)
	auditor.Record(sampleEvent())

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log output: %v", err)
	}

	// Check audit=true marker.
	if entry["audit"] != true {
		t.Errorf("expected audit=true, got %v", entry["audit"])
	}

	// Check required fields.
	requiredFields := []string{"correlating_id", "zone", "operation", "result", "fqdn", "record_type", "actor"}
	for _, f := range requiredFields {
		if _, ok := entry[f]; !ok {
			t.Errorf("expected field %q in audit log, not found", f)
		}
	}

	if entry["msg"] != "audit_event" {
		t.Errorf("expected msg='audit_event', got %v", entry["msg"])
	}
}

// --- Nop auditor tests ---

func TestNopAuditor_DoesNotPanic(t *testing.T) {
	a := Nop()
	// Should not panic.
	a.Record(sampleEvent())
}
