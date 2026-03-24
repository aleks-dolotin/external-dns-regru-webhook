package logging

import (
	"strings"

	"go.uber.org/zap/zapcore"
)

const redacted = "***REDACTED***"

// sensitiveKeys lists substrings that identify sensitive field names.
var sensitiveKeys = []string{
	"password",
	"token",
	"secret",
	"key",
	"credential",
	"authorization",
}

// isSensitive returns true if the lowercased field name contains any sensitive substring.
func isSensitive(name string) bool {
	lower := strings.ToLower(name)
	for _, k := range sensitiveKeys {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

// maskFields returns a copy of fields with sensitive string values replaced.
func maskFields(fields []zapcore.Field) []zapcore.Field {
	out := make([]zapcore.Field, len(fields))
	for i, f := range fields {
		if isSensitive(f.Key) && f.Type == zapcore.StringType {
			out[i] = zapcore.Field{
				Key:    f.Key,
				Type:   zapcore.StringType,
				String: redacted,
			}
		} else {
			out[i] = f
		}
	}
	return out
}

// maskingCore wraps a zapcore.Core and masks sensitive fields before writing.
type maskingCore struct {
	zapcore.Core
}

func (c *maskingCore) With(fields []zapcore.Field) zapcore.Core {
	return &maskingCore{Core: c.Core.With(maskFields(fields))}
}

func (c *maskingCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

func (c *maskingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(entry, maskFields(fields))
}
