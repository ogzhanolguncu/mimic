package logger

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
)

// Global logger instance
var Logger *slog.Logger

// NoOpHandler is a handler that does nothing
type NoOpHandler struct{}

// Enabled always returns false for NoOpHandler
// Note the updated signature with context.Context parameter
func (h NoOpHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

// Handle is a no-op
func (h NoOpHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

// WithAttrs returns the same no-op handler
func (h NoOpHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

// WithGroup returns the same no-op handler
func (h NoOpHandler) WithGroup(_ string) slog.Handler {
	return h
}

// Config holds logger configuration
type Config struct {
	Level   slog.Level
	Output  io.Writer
	Handler slog.Handler
}

// Initialize sets up the global logger with the specified configuration
func Initialize(cfg Config) {
	output := cfg.Output
	if output == nil {
		output = os.Stderr
	}

	handler := cfg.Handler
	if handler == nil {
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{
			Level: cfg.Level,
		})
	}

	Logger = slog.New(handler)

	log.SetOutput(output)
	log.SetFlags(0)
}

// InitNoOp initializes a no-op logger that discards all log messages
func InitNoOp() {
	Logger = slog.New(NoOpHandler{})
	log.SetOutput(io.Discard)
	log.SetFlags(0)
}

// Debug logs at debug level (no-op in tests)
func Debug(msg string, args ...any) {
	if Logger != nil {
		Logger.Debug(msg, args...)
	}
}

// Info logs at info level (no-op in tests)
func Info(msg string, args ...any) {
	if Logger != nil {
		Logger.Info(msg, args...)
	}
}

// Warn logs at warning level (no-op in tests)
func Warn(msg string, args ...any) {
	if Logger != nil {
		Logger.Warn(msg, args...)
	}
}

// Error logs at error level (no-op in tests)
func Error(msg string, args ...any) {
	if Logger != nil {
		Logger.Error(msg, args...)
	}
}

// Fatal logs at error level and then exits (no-op in tests if skipExit is true)
func Fatal(msg string, args ...any) {
	if Logger != nil {
		Logger.Error(msg, args...)
	}
	if !skipExit {
		os.Exit(1)
	}
}

// For testing, we need a way to avoid the os.Exit in Fatal
var skipExit bool

// TestMode enables test mode which prevents Fatal from calling os.Exit
func TestMode() {
	skipExit = true
}
