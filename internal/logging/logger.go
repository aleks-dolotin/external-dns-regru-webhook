// Package logging provides a structured logger built on zap with
// correlating_id context propagation and automatic secret masking.
package logging

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const loggerKey contextKey = "logger"

// NewLogger creates a production JSON logger at the given level.
// Recognized levels: "debug", "info", "warn", "error". Default is "info".
func NewLogger(level string) *zap.Logger {
	var lvl zapcore.Level
	switch level {
	case "debug":
		lvl = zapcore.DebugLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	default:
		lvl = zapcore.InfoLevel
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(os.Stdout),
		lvl,
	)

	// Wrap with masking core to redact sensitive fields.
	masked := &maskingCore{Core: core}

	return zap.New(masked, zap.AddCaller())
}

// WithContext stores the logger in the context.
func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext extracts the logger from context. Returns a no-op logger if none found.
func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok && l != nil {
		return l
	}
	return zap.NewNop()
}

// WithCorrelatingID returns a new context whose embedded logger has the
// correlating_id field attached, and also stores the enriched logger.
func WithCorrelatingID(ctx context.Context, id string) context.Context {
	l := FromContext(ctx).With(zap.String("correlating_id", id))
	return WithContext(ctx, l)
}
