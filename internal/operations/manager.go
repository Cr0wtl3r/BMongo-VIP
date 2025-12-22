package operations

import (
	"digisat-tools/internal/config"
	"digisat-tools/internal/database"
)

// LogFunc is a function type for logging messages
type LogFunc func(string)

// Manager handles all database operations
type Manager struct {
	conn  *database.Connection
	state *config.OperationState
}

// NewManager creates a new operations manager
func NewManager(conn *database.Connection) *Manager {
	return &Manager{
		conn:  conn,
		state: config.GetState(),
	}
}

// CancelAll cancels all running operations
func (m *Manager) CancelAll() {
	m.state.CancelAll()
}

// Reset resets the operation state
func (m *Manager) Reset() {
	m.state.Reset()
}
