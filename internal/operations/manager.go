package operations

import (
	"BMongo-VIP/internal/config"
	"BMongo-VIP/internal/database"
)


type LogFunc func(string)


type Manager struct {
	conn  *database.Connection
	state *config.OperationState
}


func NewManager(conn *database.Connection) *Manager {
	return &Manager{
		conn:  conn,
		state: config.GetState(),
	}
}


func (m *Manager) CancelAll() {
	m.state.CancelAll()
}


func (m *Manager) Reset() {
	m.state.Reset()
}
