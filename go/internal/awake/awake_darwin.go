//go:build darwin

package awake

import (
	"os/exec"
	"sync"
	"sync/atomic"
)

// Manager prevents the OS from sleeping on macOS using caffeinate.
type Manager struct {
	displayOnly bool
	active      atomic.Bool
	mu          sync.Mutex
	cmd         *exec.Cmd
}

// New creates a new Manager.
func New(displayOnly bool) *Manager {
	return &Manager{displayOnly: displayOnly}
}

// Enable starts caffeinate to prevent sleep.
func (m *Manager) Enable() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil {
		// already running
		return nil
	}

	var args []string
	if m.displayOnly {
		args = []string{"-d"}
	} else {
		args = []string{"-d", "-i"}
	}

	cmd := exec.Command("caffeinate", args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	m.cmd = cmd
	m.active.Store(true)
	return nil
}

// Disable stops caffeinate, re-enabling normal sleep.
func (m *Manager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd == nil {
		return
	}
	_ = m.cmd.Process.Kill()
	_ = m.cmd.Wait()
	m.cmd = nil
	m.active.Store(false)
}

// IsActive reports whether sleep prevention is currently active.
func (m *Manager) IsActive() bool {
	return m.active.Load()
}

// Close shuts down the manager.
func (m *Manager) Close() {
	m.Disable()
}
