//go:build windows

package awake

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/windows"
)

// SetThreadExecutionState flags (from winbase.h).
const (
	esContinuous      uint32 = 0x80000000
	esSystemRequired  uint32 = 0x00000001
	esDisplayRequired uint32 = 0x00000002
)

var setThreadExecutionState = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetThreadExecutionState")

// Manager prevents the OS from sleeping on Windows.
type Manager struct {
	displayOnly bool
	active      atomic.Bool
	mu          sync.Mutex
	enableCh    chan struct{}
	disableCh   chan struct{}
	closeCh     chan struct{}
	wg          sync.WaitGroup
}

// New creates a new Manager. Call Enable/Disable to control sleep prevention.
func New(displayOnly bool) *Manager {
	m := &Manager{
		displayOnly: displayOnly,
		enableCh:    make(chan struct{}, 1),
		disableCh:   make(chan struct{}, 1),
		closeCh:     make(chan struct{}),
	}
	m.wg.Add(1)
	go m.loop()
	return m
}

func (m *Manager) loop() {
	defer m.wg.Done()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	var running bool

	applyState := func() {
		if running {
			flags := esContinuous | esDisplayRequired | esSystemRequired
			if m.displayOnly {
				flags = esContinuous | esDisplayRequired
			}
			setThreadExecutionState.Call(uintptr(flags)) //nolint:errcheck
		} else {
			setThreadExecutionState.Call(uintptr(esContinuous)) //nolint:errcheck
		}
	}

	for {
		select {
		case <-m.closeCh:
			if running {
				setThreadExecutionState.Call(uintptr(esContinuous)) //nolint:errcheck
			}
			return
		case <-m.enableCh:
			running = true
			m.active.Store(true)
			applyState()
		case <-m.disableCh:
			running = false
			m.active.Store(false)
			applyState()
		case <-ticker.C:
			if running {
				applyState()
			}
		}
	}
}

// Enable activates sleep prevention.
func (m *Manager) Enable() error {
	select {
	case m.enableCh <- struct{}{}:
	default:
	}
	return nil
}

// Disable deactivates sleep prevention.
func (m *Manager) Disable() {
	select {
	case m.disableCh <- struct{}{}:
	default:
	}
}

// IsActive reports whether sleep prevention is currently active.
func (m *Manager) IsActive() bool {
	return m.active.Load()
}

// Close shuts down the manager and restores normal sleep behaviour.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	select {
	case <-m.closeCh:
		// already closed
	default:
		close(m.closeCh)
	}
	m.wg.Wait()
}
