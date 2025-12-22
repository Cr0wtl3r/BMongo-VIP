package config

import (
	"sync"
)


type OperationState struct {
	running bool
	mu      sync.RWMutex
	cancel  chan struct{}
}

var state = &OperationState{
	running: true,
	cancel:  make(chan struct{}),
}


func GetState() *OperationState {
	return state
}


func (s *OperationState) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}


func (s *OperationState) CancelAll() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()


	select {
	case <-s.cancel:

	default:
		close(s.cancel)
	}
}


func (s *OperationState) Reset() {
	s.mu.Lock()
	s.running = true
	s.cancel = make(chan struct{})
	s.mu.Unlock()
}


func (s *OperationState) Cancelled() <-chan struct{} {
	return s.cancel
}


func (s *OperationState) ShouldStop() bool {
	select {
	case <-s.cancel:
		return true
	default:
		return !s.IsRunning()
	}
}
