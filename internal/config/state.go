package config

import (
	"sync"
)

// OperationState manages the state of running operations
type OperationState struct {
	running bool
	mu      sync.RWMutex
	cancel  chan struct{}
}

var state = &OperationState{
	running: true,
	cancel:  make(chan struct{}),
}

// GetState returns the global operation state
func GetState() *OperationState {
	return state
}

// IsRunning checks if operations should continue
func (s *OperationState) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// CancelAll cancels all running operations
func (s *OperationState) CancelAll() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	// Signal cancellation
	select {
	case <-s.cancel:
		// Already closed
	default:
		close(s.cancel)
	}
}

// Reset resets the operation state to allow new operations
func (s *OperationState) Reset() {
	s.mu.Lock()
	s.running = true
	s.cancel = make(chan struct{})
	s.mu.Unlock()
}

// Cancelled returns a channel that is closed when operations are cancelled
func (s *OperationState) Cancelled() <-chan struct{} {
	return s.cancel
}

// ShouldStop checks if operation should stop
func (s *OperationState) ShouldStop() bool {
	select {
	case <-s.cancel:
		return true
	default:
		return !s.IsRunning()
	}
}
