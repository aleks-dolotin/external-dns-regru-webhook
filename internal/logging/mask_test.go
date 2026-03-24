package logging

import (
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestIsSensitive(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"password", true},
		{"db_password", true},
		{"PASSWORD", true},
		{"api_token", true},
		{"Token", true},
		{"secret_key", true},
		{"SecretValue", true},
		{"auth_credential", true},
		{"Authorization", true},
		{"private_key", true},
		{"username", false},
		{"zone", false},
		{"operation", false},
		{"correlating_id", false},
		{"namespace", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSensitive(tt.name)
			if got != tt.expected {
				t.Errorf("isSensitive(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestMaskFields(t *testing.T) {
	fields := []zapcore.Field{
		{Key: "zone", Type: zapcore.StringType, String: "example.com"},
		{Key: "password", Type: zapcore.StringType, String: "super-secret-123"},
		{Key: "api_token", Type: zapcore.StringType, String: "tok_abc"},
		{Key: "operation", Type: zapcore.StringType, String: "create"},
		{Key: "Authorization", Type: zapcore.StringType, String: "Bearer xyz"},
		{Key: "private_key", Type: zapcore.StringType, String: "-----BEGIN RSA PRIVATE KEY-----"},
	}

	masked := maskFields(fields)

	if masked[0].String != "example.com" {
		t.Errorf("expected 'zone' to pass through, got %q", masked[0].String)
	}
	if masked[1].String != redacted {
		t.Errorf("expected 'password' to be redacted, got %q", masked[1].String)
	}
	if masked[2].String != redacted {
		t.Errorf("expected 'api_token' to be redacted, got %q", masked[2].String)
	}
	if masked[3].String != "create" {
		t.Errorf("expected 'operation' to pass through, got %q", masked[3].String)
	}
	if masked[4].String != redacted {
		t.Errorf("expected 'Authorization' to be redacted, got %q", masked[4].String)
	}
	if masked[5].String != redacted {
		t.Errorf("expected 'private_key' to be redacted, got %q", masked[5].String)
	}
}

func TestMaskFields_NonStringUnchanged(t *testing.T) {
	fields := []zapcore.Field{
		{Key: "secret_count", Type: zapcore.Int64Type, Integer: 42},
	}

	masked := maskFields(fields)

	// Non-string fields with sensitive names should NOT be masked (only string type).
	if masked[0].Integer != 42 {
		t.Errorf("expected non-string field to pass through, got %d", masked[0].Integer)
	}
}

func TestMaskFields_EmptySlice(t *testing.T) {
	masked := maskFields(nil)
	if len(masked) != 0 {
		t.Errorf("expected empty result for nil input, got %d", len(masked))
	}
}
