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

	// EventConnectionUpdated is emitted after a connection is successfully updated.
	EventConnectionUpdated = "connection:updated"

	// EventConnectionDeleted is emitted after a connection is successfully removed.
	EventConnectionDeleted = "connection:deleted"

	// EventMenuLogsToggled is emitted by the native menu to request the frontend toggle the logs panel.
	EventMenuLogsToggled = "menu:logs-toggled"

	// EventConnectionsWindowClosed is emitted when the connections window is hidden.
	EventConnectionsWindowClosed = "connections-window:closed"

	// EventEditConnectionWindowOpened is emitted when the edit-connection window is shown, carrying the target connection ID.
	EventEditConnectionWindowOpened = "edit-connection-window:opened"

	// EventEditConnectionWindowClosed is emitted when the edit-connection window is hidden.
	EventEditConnectionWindowClosed = "edit-connection-window:closed"

	// EventPluginsReady is emitted by the plugin manager once the initial async
	// scan has completed and ListPlugins() returns a populated result.
	EventPluginsReady = "plugins:ready"
)

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	// LogLevelDebug can be used for low‑priority messages that are useful
	// during development but not generally shown to end users.
	LogLevelDebug LogLevel = "debug"
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

// ConnectionUpdatedEvent is the payload emitted on EventConnectionUpdated.
type ConnectionUpdatedEvent struct {
	Connection Connection `json:"connection"`
}

// ConnectionDeletedEvent is the payload emitted on EventConnectionDeleted.
type ConnectionDeletedEvent struct {
	ID string `json:"id"`
}

// EditConnectionWindowOpenedEvent is the payload emitted on EventEditConnectionWindowOpened.
type EditConnectionWindowOpenedEvent struct {
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

// emitConnectionUpdated emits EventConnectionUpdated with the updated connection as payload.
func emitConnectionUpdated(app *application.App, conn Connection) {
	if app == nil {
		return
	}
	app.Event.Emit(EventConnectionUpdated, ConnectionUpdatedEvent{Connection: conn})
}

// emitConnectionDeleted emits EventConnectionDeleted with the removed connection's ID.
func emitConnectionDeleted(app *application.App, id string) {
	if app == nil {
		return
	}
	app.Event.Emit(EventConnectionDeleted, ConnectionDeletedEvent{ID: id})
}
