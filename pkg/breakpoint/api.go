// Package breakpoint provides a testing/debugging framework for managing parallel execution
// with breakpoints to help test race conditions and control execution flow.
//
// The framework allows you to:
// - Add breakpoints in your code using AddBreaker(name)
// - Enable/disable breakpoints dynamically
// - Control execution flow by proceeding breakpoints in desired order
// - Wait for breakpoints to be hit for test orchestration
//
// Example usage:
//
//	// In your code under test
//	func SomeFunction() {
//		// ... some logic ...
//		breakpoint.AddBreaker("before-critical-section")
//		// ... critical section ...
//		breakpoint.AddBreaker("after-critical-section")
//	}
//
//	// In your test
//	func TestRaceCondition(t *testing.T) {
//		breakpoint.Reset() // Clean state
//		breakpoint.Enable("before-critical-section")
//
//		// Start multiple goroutines
//		go SomeFunction()
//		go SomeFunction()
//
//		// Wait for both to hit the breakpoint
//		breakpoint.WaitForBreakpoint("before-critical-section", time.Second)
//
//		// Proceed in desired order
//		breakpoint.Proceed("before-critical-section")
//	}
package breakpoint

import (
	"time"
)

// Global manager instance
var globalManager = NewManager()

// AddBreaker adds a breakpoint in the code. If the breakpoint is enabled,
// execution will pause until Proceed is called for this breakpoint.
func AddBreaker(name string) {
	globalManager.addBreaker(name)
}

// Enable enables a specific breakpoint by name.
func Enable(name string) {
	globalManager.enableBreakpoint(name)
}

// EnableAll enables all registered breakpoints.
func EnableAll() {
	globalManager.enableAllBreakpoints()
}

// Disable disables a specific breakpoint by name.
func Disable(name string) {
	globalManager.disableBreakpoint(name)
}

// DisableAll disables all registered breakpoints.
func DisableAll() {
	globalManager.disableAllBreakpoints()
}

// Proceed allows a specific breakpoint to continue execution.
func Proceed(name string) {
	globalManager.proceedBreakpoint(name)
}

// ProceedAll allows all breakpoints to continue execution.
func ProceedAll() {
	globalManager.proceedAllBreakpoints()
}

// WaitForBreakpoint waits for a specific breakpoint to be hit within the given timeout.
// Returns an error if the breakpoint doesn't exist or timeout is reached.
func WaitForBreakpoint(name string, timeout time.Duration) error {
	return globalManager.waitForBreakpoint(name, timeout)
}

// Reset resets a specific breakpoint to its initial state.
func Reset(name string) {
	globalManager.resetBreakpoint(name)
}

// ResetAll resets all breakpoints to their initial state.
// This is useful for cleaning up between tests.
func ResetAll() {
	globalManager.resetAllBreakpoints()
}

// List returns information about all registered breakpoints.
// Useful for debugging and introspection.
func List() []string {
	return globalManager.listBreakpoints()
}

// IsHit returns whether a specific breakpoint has been hit.
// Returns false and an error if the breakpoint doesn't exist.
func IsHit(name string) (bool, error) {
	bp := globalManager.registerBreakpoint(name) // Register if it doesn't exist
	return bp.isHit(), nil
}

// IsEnabled returns whether a specific breakpoint is enabled.
// Returns false and an error if the breakpoint doesn't exist.
func IsEnabled(name string) (bool, error) {
	bp := globalManager.registerBreakpoint(name) // Register if it doesn't exist
	return bp.isEnabled(), nil
}
