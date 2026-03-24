package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newTestLogger returns a logger that writes to buf in JSON format.
func newTestLogger(buf *bytes.Buffer) *zap.Logger {
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(buf),
		zapcore.DebugLevel,
	)
	masked := &maskingCore{Core: core}
	return zap.New(masked)
}

func TestStructuredOutput_ContainsRequiredFields(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	logger.Info("processing operation",
		zap.String("correlating_id", "uuid-123"),
		zap.String("zone", "example.com"),
		zap.String("operation", "create"),
		zap.String("resource", "my-svc"),
		zap.String("namespace", "default"),
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON log: %v", err)
	}

	required := []string{"timestamp", "level", "correlating_id", "zone", "operation", "resource", "namespace"}
	for _, field := range required {
		if _, ok := entry[field]; !ok {
			t.Errorf("expected field %q in log output, not found", field)
		}
	}
}

func TestStructuredOutput_MasksSensitiveFields(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	logger.Info("auth check",
		zap.String("password", "s3cret"),
		zap.String("zone", "example.com"),
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON log: %v", err)
	}

	if entry["password"] != redacted {
		t.Errorf("expected password to be %q, got %v", redacted, entry["password"])
	}
	if entry["zone"] != "example.com" {
		t.Errorf("expected zone to be 'example.com', got %v", entry["zone"])
	}
}

func TestWithCorrelatingID_PropagatesThroughContext(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	ctx := WithContext(context.Background(), logger)
	ctx = WithCorrelatingID(ctx, "corr-456")

	l := FromContext(ctx)
	l.Info("test message")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON log: %v", err)
	}

	if entry["correlating_id"] != "corr-456" {
		t.Errorf("expected correlating_id='corr-456', got %v", entry["correlating_id"])
	}
}

func TestFromContext_ReturnsNopWhenMissing(t *testing.T) {
	l := FromContext(context.Background())
	if l == nil {
		t.Fatal("expected non-nil logger from empty context")
	}
	// Should not panic.
	l.Info("should be silently discarded")
}

func TestNewLogger_DefaultLevel(t *testing.T) {
	l := NewLogger("")
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewLogger_AllLevels(t *testing.T) {
	for _, lvl := range []string{"debug", "info", "warn", "error"} {
		l := NewLogger(lvl)
		if l == nil {
			t.Errorf("expected non-nil logger for level %q", lvl)
		}
	}
}
