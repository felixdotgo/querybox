package services

import (
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogEntry is the payload emitted on the "app:log" event to the frontend.
type LogEntry struct {
	Level     LogLevel `json:"level"`
	Message   string   `json:"message"`
	Timestamp string   `json:"timestamp"` // RFC3339Nano UTC
}

// emitLog is a nil-safe helper that emits an app:log event on the Wails app.
// If app is nil the call is a no-op so services remain functional in tests.
func emitLog(app *application.App, level LogLevel, message string) {
	if app == nil {
		return
	}
	app.Event.Emit("app:log", LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	})
}
