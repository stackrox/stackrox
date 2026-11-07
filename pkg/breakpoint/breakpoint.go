package breakpoint

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Breakpoint represents a single breakpoint with its state and synchronization primitives
type Breakpoint struct {
	name    string
	enabled bool
	hit     bool

	// Channel to block execution when breakpoint is hit
	proceedCh chan struct{}

	// Channel to signal when breakpoint is hit (for WaitForBreakpoint)
	hitCh chan struct{}

	// Mutex to protect breakpoint state
	mu sync.RWMutex
}

// newBreakpoint creates a new breakpoint instance
func newBreakpoint(name string) *Breakpoint {
	return &Breakpoint{
		name:      name,
		enabled:   false,
		hit:       false,
		proceedCh: make(chan struct{}),
		hitCh:     make(chan struct{}),
	}
}

// isEnabled returns whether the breakpoint is enabled
func (b *Breakpoint) isEnabled() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.enabled
}

// enable enables the breakpoint
func (b *Breakpoint) enable() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.enabled = true
}

// disable disables the breakpoint
func (b *Breakpoint) disable() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.enabled = false
}

// isHit returns whether the breakpoint has been hit
func (b *Breakpoint) isHit() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.hit
}

// hit marks the breakpoint as hit and signals waiters
func (b *Breakpoint) markHit() {
	b.mu.Lock()
	if !b.hit {
		b.hit = true
		close(b.hitCh) // Signal that breakpoint was hit
	}
	b.mu.Unlock()
}

// proceed allows the breakpoint to continue execution
func (b *Breakpoint) proceed() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Close the proceed channel to unblock any waiting goroutines
	select {
	case <-b.proceedCh:
		// Already closed
	default:
		close(b.proceedCh)
	}
}

// reset resets the breakpoint to its initial state
func (b *Breakpoint) reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// First, unblock any waiting goroutines by closing the proceed channel
	select {
	case <-b.proceedCh:
		// Already closed
	default:
		close(b.proceedCh)
	}

	// Reset state
	b.enabled = false
	b.hit = false

	// Create new channels
	b.proceedCh = make(chan struct{})
	b.hitCh = make(chan struct{})
}

// waitForHit waits for the breakpoint to be hit with a timeout
func (b *Breakpoint) waitForHit(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-b.hitCh:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for breakpoint '%s' to be hit", b.name)
	}
}

// waitToProceed blocks until the breakpoint is allowed to proceed
func (b *Breakpoint) waitToProceed() {
	<-b.proceedCh
}

// String returns a string representation of the breakpoint
func (b *Breakpoint) String() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return fmt.Sprintf("Breakpoint{name: %s, enabled: %t, hit: %t}", b.name, b.enabled, b.hit)
}
