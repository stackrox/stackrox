package concurrency_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/breakpoint"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkerPoolWithBreakpoints demonstrates how to use breakpoints
// to control and test race conditions in the worker pool
func TestWorkerPoolWithBreakpoints(t *testing.T) {
	breakpoint.ResetAll()

	// Enable breakpoints to control job execution order
	breakpoint.Enable("job-start")
	breakpoint.Enable("job-end")

	pool := concurrency.NewWorkerPool(2) // Allow 2 concurrent jobs
	err := pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	var results []int
	var mu sync.Mutex

	// Create jobs that will hit breakpoints
	createJob := func(id int) func() {
		return func() {
			breakpoint.AddBreaker("job-start")

			// Simulate work
			mu.Lock()
			results = append(results, id)
			mu.Unlock()

			breakpoint.AddBreaker("job-end")
		}
	}

	// Add 2 jobs (equal to pool capacity to avoid blocking)
	for i := 1; i <= 2; i++ {
		err := pool.AddJob(createJob(i))
		require.NoError(t, err)
	}

	// Wait for both jobs to start
	err = breakpoint.WaitForBreakpoint("job-start", time.Second)
	require.NoError(t, err)
	err = breakpoint.WaitForBreakpoint("job-start", time.Second)
	require.NoError(t, err)

	// At this point, 2 jobs should be running
	mu.Lock()
	assert.Empty(t, results, "No jobs should have completed work yet")
	mu.Unlock()

	// Let the jobs proceed to completion
	breakpoint.Proceed("job-start")

	// Wait for them to finish
	err = breakpoint.WaitForBreakpoint("job-end", time.Second)
	require.NoError(t, err)
	err = breakpoint.WaitForBreakpoint("job-end", time.Second)
	require.NoError(t, err)

	// Now let them complete
	breakpoint.Proceed("job-end")

	// Give a moment for completion
	time.Sleep(50 * time.Millisecond)

	// Verify all jobs completed
	mu.Lock()
	assert.Len(t, results, 2, "Both jobs should have completed")
	mu.Unlock()
}

// TestWorkerPoolConcurrencyControl demonstrates controlling the exact
// order of job execution using breakpoints
func TestWorkerPoolConcurrencyControl(t *testing.T) {
	breakpoint.ResetAll()

	// Enable breakpoints for precise control
	breakpoint.Enable("before-critical")
	breakpoint.Enable("after-critical")

	pool := concurrency.NewWorkerPool(3)
	err := pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	var sharedCounter int
	var mu sync.Mutex
	var executionOrder []int

	// Create jobs that increment a shared counter
	createIncrementJob := func(id int) func() {
		return func() {
			breakpoint.AddBreaker("before-critical")

			// Critical section - increment counter
			mu.Lock()
			currentValue := sharedCounter
			time.Sleep(10 * time.Millisecond) // Simulate some work
			sharedCounter = currentValue + 1
			executionOrder = append(executionOrder, id)
			mu.Unlock()

			breakpoint.AddBreaker("after-critical")
		}
	}

	// Add 3 jobs
	numJobs := 3
	for i := 1; i <= numJobs; i++ {
		err := pool.AddJob(createIncrementJob(i))
		require.NoError(t, err)
	}

	// Wait for all jobs to reach the critical section
	for i := 0; i < numJobs; i++ {
		err := breakpoint.WaitForBreakpoint("before-critical", time.Second)
		require.NoError(t, err)
	}

	// All jobs are now blocked before the critical section
	assert.Equal(t, 0, sharedCounter, "Counter should still be 0")

	// Let them proceed one by one to ensure serialized access
	for i := 0; i < numJobs; i++ {
		// Let one job proceed
		breakpoint.Proceed("before-critical")

		// Wait for it to complete the critical section
		err := breakpoint.WaitForBreakpoint("after-critical", time.Second)
		require.NoError(t, err)

		// Let it complete
		breakpoint.Proceed("after-critical")

		// Verify counter was incremented
		mu.Lock()
		expectedValue := i + 1
		assert.Equal(t, expectedValue, sharedCounter, "Counter should be %d after job %d", expectedValue, i+1)
		mu.Unlock()
	}

	// Verify final state
	mu.Lock()
	assert.Equal(t, numJobs, sharedCounter, "Final counter should equal number of jobs")
	assert.Len(t, executionOrder, numJobs, "All jobs should have executed")
	mu.Unlock()
}
