package breakpoint

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBreakpointBasicFunctionality(t *testing.T) {
	// Reset global state
	ResetAll()

	// Test that breakpoint doesn't block when disabled
	executed := false
	go func() {
		AddBreaker("test-breakpoint")
		executed = true
	}()

	// Give it some time to execute
	time.Sleep(100 * time.Millisecond)
	assert.True(t, executed, "Disabled breakpoint should not block execution")

	// Verify breakpoint was registered but not hit (since it was disabled)
	hit, err := IsHit("test-breakpoint")
	require.NoError(t, err)
	assert.False(t, hit, "Disabled breakpoint should not be marked as hit")
}

func TestBreakpointBlocksWhenEnabled(t *testing.T) {
	ResetAll()

	// Enable the breakpoint
	Enable("blocking-test")

	executed := false
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		AddBreaker("blocking-test")
		executed = true
	}()

	// Give it some time - should not execute due to breakpoint
	time.Sleep(100 * time.Millisecond)
	assert.False(t, executed, "Enabled breakpoint should block execution")

	// Verify breakpoint was hit
	hit, err := IsHit("blocking-test")
	require.NoError(t, err)
	assert.True(t, hit, "Enabled breakpoint should be marked as hit")

	// Proceed the breakpoint
	Proceed("blocking-test")

	// Wait for execution to complete
	wg.Wait()
	assert.True(t, executed, "Execution should complete after proceeding")
}

func TestWaitForBreakpoint(t *testing.T) {
	ResetAll()

	Enable("wait-test")

	// Start goroutine that will hit breakpoint
	go func() {
		time.Sleep(50 * time.Millisecond)
		AddBreaker("wait-test")
	}()

	// Wait for breakpoint to be hit
	err := WaitForBreakpoint("wait-test", time.Second)
	require.NoError(t, err)

	// Verify it was hit
	hit, err := IsHit("wait-test")
	require.NoError(t, err)
	assert.True(t, hit)

	// Proceed to clean up
	Proceed("wait-test")
}

func TestWaitForBreakpointTimeout(t *testing.T) {
	ResetAll()

	// Enable a breakpoint but don't hit it
	Enable("timeout-test")

	err := WaitForBreakpoint("timeout-test", 100*time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestMultipleBreakpoints(t *testing.T) {
	ResetAll()

	// Enable multiple breakpoints
	Enable("bp1")
	Enable("bp2")

	var wg sync.WaitGroup
	executed1, executed2 := false, false

	// Start two goroutines
	wg.Add(2)
	go func() {
		defer wg.Done()
		AddBreaker("bp1")
		executed1 = true
	}()

	go func() {
		defer wg.Done()
		AddBreaker("bp2")
		executed2 = true
	}()

	// Wait for both to hit
	err := WaitForBreakpoint("bp1", time.Second)
	require.NoError(t, err)
	err = WaitForBreakpoint("bp2", time.Second)
	require.NoError(t, err)

	// Neither should have completed execution
	assert.False(t, executed1)
	assert.False(t, executed2)

	// Proceed both
	ProceedAll()

	// Wait for completion
	wg.Wait()
	assert.True(t, executed1)
	assert.True(t, executed2)
}

func TestRaceConditionControl(t *testing.T) {
	ResetAll()

	Enable("race-bp")

	var results []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Function that simulates a race condition
	testFunc := func(id int) {
		defer wg.Done()
		AddBreaker("race-bp")

		// Critical section
		mu.Lock()
		results = append(results, id)
		mu.Unlock()
	}

	// Start multiple goroutines
	wg.Add(3)
	go testFunc(1)
	go testFunc(2)
	go testFunc(3)

	// Wait for all to hit the breakpoint
	time.Sleep(100 * time.Millisecond)

	// All should be blocked at the breakpoint
	mu.Lock()
	assert.Empty(t, results, "No goroutines should have entered critical section yet")
	mu.Unlock()

	// Proceed all at once
	ProceedAll()

	// Wait for completion
	wg.Wait()

	// All should have completed
	mu.Lock()
	assert.Len(t, results, 3, "All goroutines should have completed")
	mu.Unlock()
}

func TestEnableAll(t *testing.T) {
	ResetAll()

	// Register some breakpoints by hitting them (while disabled)
	AddBreaker("bp1")
	AddBreaker("bp2")
	AddBreaker("bp3")

	// Enable all
	EnableAll()

	// Verify all are enabled
	enabled1, err := IsEnabled("bp1")
	require.NoError(t, err)
	assert.True(t, enabled1)

	enabled2, err := IsEnabled("bp2")
	require.NoError(t, err)
	assert.True(t, enabled2)

	enabled3, err := IsEnabled("bp3")
	require.NoError(t, err)
	assert.True(t, enabled3)
}

func TestDisableAll(t *testing.T) {
	ResetAll()

	// Enable some breakpoints
	Enable("bp1")
	Enable("bp2")

	// Disable all
	DisableAll()

	// Verify all are disabled
	enabled1, err := IsEnabled("bp1")
	require.NoError(t, err)
	assert.False(t, enabled1)

	enabled2, err := IsEnabled("bp2")
	require.NoError(t, err)
	assert.False(t, enabled2)
}

func TestResetFunctionality(t *testing.T) {
	ResetAll()

	// Enable and hit a breakpoint
	Enable("reset-test")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		AddBreaker("reset-test")
	}()

	// Wait for it to be hit
	err := WaitForBreakpoint("reset-test", time.Second)
	require.NoError(t, err)

	// Reset the specific breakpoint - this should unblock the goroutine
	Reset("reset-test")

	// Wait for execution to complete
	wg.Wait()

	// Should no longer be enabled or hit
	enabled, err := IsEnabled("reset-test")
	require.NoError(t, err)
	assert.False(t, enabled)

	hit, err := IsHit("reset-test")
	require.NoError(t, err)
	assert.False(t, hit)
}

func TestListBreakpoints(t *testing.T) {
	ResetAll()

	// Initially should be empty
	list := List()
	assert.Empty(t, list)

	// Add some breakpoints
	AddBreaker("bp1")
	Enable("bp2")

	list = List()
	assert.Len(t, list, 2)

	// Check that list contains information about breakpoints
	listStr := ""
	for _, info := range list {
		listStr += info
	}
	assert.Contains(t, listStr, "bp1")
	assert.Contains(t, listStr, "bp2")
}

func TestErrorHandling(t *testing.T) {
	ResetAll()

	// Test operations on non-existent breakpoints - should not error now
	Enable("nonexistent")
	Disable("nonexistent")
	Proceed("nonexistent")
	Reset("nonexistent")

	// These should work without error now
	hit, err := IsHit("nonexistent")
	assert.NoError(t, err)
	assert.False(t, hit)

	enabled, err := IsEnabled("nonexistent")
	assert.NoError(t, err)
	assert.False(t, enabled)
}

func TestConcurrentAccess(t *testing.T) {
	ResetAll()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Test concurrent registration and access
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			bpName := fmt.Sprintf("concurrent-bp-%d", id)

			// Register breakpoint
			AddBreaker(bpName)

			// Enable it
			Enable(bpName)

			// Check if enabled
			_, _ = IsEnabled(bpName)

			// Proceed it
			Proceed(bpName)
		}(i)
	}

	wg.Wait()

	// Should have registered all breakpoints
	list := List()
	assert.Len(t, list, numGoroutines)
}
