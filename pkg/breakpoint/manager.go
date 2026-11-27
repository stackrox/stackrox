package breakpoint

import (
	"fmt"
	"sync"
	"time"
)

// Manager handles all breakpoints in a thread-safe manner
type Manager struct {
	breakpoints map[string]*Breakpoint
	mu          sync.RWMutex
}

// NewManager creates a new breakpoint manager
func NewManager() *Manager {
	return &Manager{
		breakpoints: make(map[string]*Breakpoint),
	}
}

// registerBreakpoint registers a new breakpoint or returns existing one
func (m *Manager) registerBreakpoint(name string) *Breakpoint {
	m.mu.Lock()
	defer m.mu.Unlock()

	if bp, exists := m.breakpoints[name]; exists {
		return bp
	}

	bp := newBreakpoint(name)
	m.breakpoints[name] = bp
	return bp
}

// getBreakpoint returns a breakpoint by name
func (m *Manager) getBreakpoint(name string) (*Breakpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bp, exists := m.breakpoints[name]
	if !exists {
		return nil, fmt.Errorf("breakpoint '%s' not found", name)
	}
	return bp, nil
}

// getAllBreakpoints returns all registered breakpoints
func (m *Manager) getAllBreakpoints() []*Breakpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	breakpoints := make([]*Breakpoint, 0, len(m.breakpoints))
	for _, bp := range m.breakpoints {
		breakpoints = append(breakpoints, bp)
	}
	return breakpoints
}

// enableBreakpoint enables a specific breakpoint
func (m *Manager) enableBreakpoint(name string) error {
	bp := m.registerBreakpoint(name) // Register if it doesn't exist
	bp.enable()
	return nil
}

// enableAllBreakpoints enables all registered breakpoints
func (m *Manager) enableAllBreakpoints() {
	breakpoints := m.getAllBreakpoints()
	for _, bp := range breakpoints {
		bp.enable()
	}
}

// disableBreakpoint disables a specific breakpoint
func (m *Manager) disableBreakpoint(name string) error {
	bp := m.registerBreakpoint(name) // Register if it doesn't exist
	bp.disable()
	return nil
}

// disableAllBreakpoints disables all registered breakpoints
func (m *Manager) disableAllBreakpoints() {
	breakpoints := m.getAllBreakpoints()
	for _, bp := range breakpoints {
		bp.disable()
	}
}

// proceedBreakpoint allows a specific breakpoint to continue
func (m *Manager) proceedBreakpoint(name string) error {
	bp := m.registerBreakpoint(name) // Register if it doesn't exist
	bp.proceed()
	return nil
}

// proceedAllBreakpoints allows all breakpoints to continue
func (m *Manager) proceedAllBreakpoints() {
	breakpoints := m.getAllBreakpoints()
	for _, bp := range breakpoints {
		bp.proceed()
	}
}

// waitForBreakpoint waits for a specific breakpoint to be hit
func (m *Manager) waitForBreakpoint(name string, timeout time.Duration) error {
	bp := m.registerBreakpoint(name) // Register if it doesn't exist
	return bp.waitForHit(timeout)
}

// resetBreakpoint resets a specific breakpoint
func (m *Manager) resetBreakpoint(name string) error {
	bp := m.registerBreakpoint(name) // Register if it doesn't exist
	bp.reset()
	return nil
}

// resetAllBreakpoints resets all breakpoints
func (m *Manager) resetAllBreakpoints() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Reset all existing breakpoints
	for _, bp := range m.breakpoints {
		bp.reset()
	}

	// Clear the map to start fresh
	m.breakpoints = make(map[string]*Breakpoint)
}

// listBreakpoints returns information about all registered breakpoints
func (m *Manager) listBreakpoints() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var info []string
	for _, bp := range m.breakpoints {
		info = append(info, bp.String())
	}
	return info
}

// addBreaker is called from user code to hit a breakpoint
func (m *Manager) addBreaker(name string) {
	bp := m.registerBreakpoint(name)

	// Only block if the breakpoint is enabled
	if bp.isEnabled() {
		bp.markHit()
		bp.waitToProceed()
	}
}
