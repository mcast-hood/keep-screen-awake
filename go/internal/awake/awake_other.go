//go:build !windows && !darwin

package awake

import (
	"log"
	"sync/atomic"
)

// Manager is a stub for unsupported platforms.
type Manager struct {
	displayOnly bool
	active      atomic.Bool
}

// New creates a new Manager stub.
func New(displayOnly bool) *Manager {
	log.Println("awake: sleep prevention is not supported on this platform")
	return &Manager{displayOnly: displayOnly}
}

// Enable is a no-op on unsupported platforms.
func (m *Manager) Enable() error {
	log.Println("awake: Enable called but sleep prevention is not supported on this platform")
	m.active.Store(true)
	return nil
}

// Disable is a no-op on unsupported platforms.
func (m *Manager) Disable() {
	m.active.Store(false)
}

// IsActive reports whether Enable has been called.
func (m *Manager) IsActive() bool {
	return m.active.Load()
}

// Close is a no-op on unsupported platforms.
func (m *Manager) Close() {}
