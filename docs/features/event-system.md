# Event System

## What it is

The backend is the sole producer of domain events. The frontend consumes those events to keep UI state reactive without unnecessary reloads.

## Current behavior

- Go services emit events after successful state changes.
- Frontend components subscribe through Wails event bindings.
- Event names use a stable domain-oriented naming convention such as `connection:created`.

## Key components

- `services/events.go`
- `services/connection.go`
- `services/app.go`
- frontend event subscribers

## How it connects to other modules

- Connection management emits lifecycle events.
- Plugin manager emits readiness notifications.
- The app shell and native menu emit UI-level state events.

## Limits and roadmap

- The event system is local-process scoped today.
- It is not yet a generalized streaming transport for external systems.
