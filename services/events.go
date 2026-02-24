package services

import (
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Event name constants.
// All domain events are emitted exclusively from the backend. The frontend
// must never call Events.Emit for these topics; it only subscribes and reacts.
const (
	// EventAppLog is emitted by every service to stream structured log entries.
	EventAppLog = "app:log"

	// EventConnectionCreated is emitted after a connection is successfully persisted.
	EventConnectionCreated = "connection:created"

	// EventConnectionDeleted is emitted after a connection is successfully removed.
	EventConnectionDeleted = "connection:deleted"

	// EventMenuLogsToggled is emitted by the native menu to request the frontend toggle the logs panel.
	EventMenuLogsToggled = "menu:logs-toggled"

	// EventConnectionsWindowClosed is emitted when the connections window is hidden.
	EventConnectionsWindowClosed = "connections-window:closed"
)

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogEntry is the payload emitted on the EventAppLog event.
type LogEntry struct {
	Level     LogLevel `json:"level"`
	Message   string   `json:"message"`
	Timestamp string   `json:"timestamp"` // RFC3339Nano UTC
}

// ConnectionCreatedEvent is the payload emitted on EventConnectionCreated.
type ConnectionCreatedEvent struct {
	Connection Connection `json:"connection"`
}

// ConnectionDeletedEvent is the payload emitted on EventConnectionDeleted.
type ConnectionDeletedEvent struct {
	ID string `json:"id"`
}

// emitLog is a nil-safe helper that emits an EventAppLog event on the Wails app.
// If app is nil the call is a no-op so services remain functional in tests.
func emitLog(app *application.App, level LogLevel, message string) {
	if app == nil {
		return
	}
	app.Event.Emit(EventAppLog, LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// emitConnectionCreated emits EventConnectionCreated with the new connection as payload.
func emitConnectionCreated(app *application.App, conn Connection) {
	if app == nil {
		return
	}
	app.Event.Emit(EventConnectionCreated, ConnectionCreatedEvent{Connection: conn})
}

// emitConnectionDeleted emits EventConnectionDeleted with the removed connection's ID.
func emitConnectionDeleted(app *application.App, id string) {
	if app == nil {
		return
	}
	app.Event.Emit(EventConnectionDeleted, ConnectionDeletedEvent{ID: id})
}
