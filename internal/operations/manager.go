package operations

import (
	"BMongo-VIP/internal/config"
	"BMongo-VIP/internal/database"
)

type LogFunc func(string)

type Manager struct {
	conn     *database.Connection
	state    *config.OperationState
	rollback *RollbackManager
}

func NewManager(conn *database.Connection) *Manager {
	return &Manager{
		conn:  conn,
		state: config.GetState(),
	}
}

func NewManagerWithRollback(conn *database.Connection, rollback *RollbackManager) *Manager {
	return &Manager{
		conn:     conn,
		state:    config.GetState(),
		rollback: rollback,
	}
}

func (m *Manager) SetRollback(rollback *RollbackManager) {
	m.rollback = rollback
}

func (m *Manager) GetRollback() *RollbackManager {
	return m.rollback
}

func (m *Manager) CancelAll() {
	m.state.CancelAll()
}

func (m *Manager) Reset() {
	m.state.Reset()
}
