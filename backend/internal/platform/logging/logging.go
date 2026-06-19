// Package logging provides the application's structured logger built on the
// standard library log/slog package.
//
// On Cloud Run (detected via the K_SERVICE environment variable) it emits JSON
// with field names that Cloud Logging understands ("severity", "message",
// "time") so log entries are parsed into structured payloads with the correct
// severity. Locally it emits human-readable text.
//
// Configuration via environment variables:
//
//	LOG_LEVEL  - debug | info | warn | error   (default: info)
//	LOG_FORMAT - json | text                    (default: json on Cloud Run, text locally)
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger builds the application logger and installs it as the slog default
// so any package calling slog.Info/slog.Error directly shares the same output.
func NewLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:       parseLevel(os.Getenv("LOG_LEVEL")),
		ReplaceAttr: cloudLoggingAttrs,
	}

	var handler slog.Handler
	if useJSON() {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		// Locally, keep the default text field names for readability.
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: opts.Level})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// useJSON reports whether JSON output should be used. It honours an explicit
// LOG_FORMAT override and otherwise defaults to JSON when running on Cloud Run.
func useJSON() bool {
	switch strings.ToLower(os.Getenv("LOG_FORMAT")) {
	case "json":
		return true
	case "text":
		return false
	default:
		// K_SERVICE is set by the Cloud Run runtime.
		return os.Getenv("K_SERVICE") != ""
	}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// cloudLoggingAttrs rewrites slog's default keys to the special fields Cloud
// Logging recognises, so JSON entries land with the right severity and message.
// See: https://cloud.google.com/logging/docs/structured-logging
func cloudLoggingAttrs(groups []string, a slog.Attr) slog.Attr {
	if len(groups) != 0 {
		return a
	}
	switch a.Key {
	case slog.MessageKey:
		a.Key = "message"
	case slog.TimeKey:
		a.Key = "time"
	case slog.LevelKey:
		a.Key = "severity"
		// Cloud Logging uses "WARNING" rather than slog's "WARN".
		if level, ok := a.Value.Any().(slog.Level); ok && level == slog.LevelWarn {
			a.Value = slog.StringValue("WARNING")
		}
	}
	return a
}
