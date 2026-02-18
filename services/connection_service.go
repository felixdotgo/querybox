package services

import (
	"context"

	"github.com/felixdotgo/querybox/services/connection"
)

// ConnectionService is the application-facing service that exposes connection
// management APIs to the frontend. Methods are intentionally thin wrappers over
// the lower-level ConnectionManager to keep bindings small and focused.
type ConnectionService struct {
	mgr *connection.ConnectionManager
}

// NewConnectionService constructs a ConnectionService and initializes the
// underlying SQLite-backed ConnectionManager.
func NewConnectionService() *ConnectionService {
	return &ConnectionService{mgr: connection.New()}
}

// ListConnections returns all configured connections.
func (s *ConnectionService) ListConnections(ctx context.Context) ([]connection.Connection, error) {
	return s.mgr.List(ctx)
}

// CreateConnection creates and persists a new connection record.
func (s *ConnectionService) CreateConnection(ctx context.Context, name, driverType, credential string) (connection.Connection, error) {
	return s.mgr.Create(ctx, name, driverType, credential)
}

// DeleteConnection removes a connection by id.
func (s *ConnectionService) DeleteConnection(ctx context.Context, id string) error {
	return s.mgr.Delete(ctx, id)
}

// GetConnection retrieves a single connection by id.
func (s *ConnectionService) GetConnection(ctx context.Context, id string) (connection.Connection, error) {
	return s.mgr.Get(ctx, id)
}
