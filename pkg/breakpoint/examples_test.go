package breakpoint_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/breakpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Example function that simulates a race condition scenario
func simulateRaceCondition(id int, sharedResource *[]int, mu *sync.Mutex) {
	// Add breakpoint before entering critical section
	breakpoint.AddBreaker("before-critical")

	// Critical section
	mu.Lock()
	*sharedResource = append(*sharedResource, id)
	mu.Unlock()

	// Add breakpoint after critical section
	breakpoint.AddBreaker("after-critical")
}

// Example function that simulates producer-consumer scenario
func producer(ch chan<- int, id int) {
	breakpoint.AddBreaker("producer-start")

	for i := 0; i < 3; i++ {
		breakpoint.AddBreaker(fmt.Sprintf("producer-%d-item-%d", id, i))
		ch <- i*10 + id
	}

	breakpoint.AddBreaker("producer-end")
}

func consumer(ch <-chan int, results *[]int, mu *sync.Mutex) {
	breakpoint.AddBreaker("consumer-start")

	for i := 0; i < 6; i++ { // Expecting 6 items from 2 producers
		breakpoint.AddBreaker(fmt.Sprintf("consumer-receive-%d", i))
		value := <-ch

		mu.Lock()
		*results = append(*results, value)
		mu.Unlock()
	}

	breakpoint.AddBreaker("consumer-end")
}

func TestRaceConditionControlExample(t *testing.T) {
	breakpoint.ResetAll()

	// Enable the breakpoint we want to control
	breakpoint.Enable("before-critical")

	var sharedResource []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Start multiple goroutines that will race
	numGoroutines := 3
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			simulateRaceCondition(id, &sharedResource, &mu)
		}(i)
	}

	// Wait for all goroutines to hit the breakpoint
	for i := 0; i < numGoroutines; i++ {
		err := breakpoint.WaitForBreakpoint("before-critical", time.Second)
		require.NoError(t, err, "Goroutine %d should hit breakpoint", i)
	}

	// At this point, all goroutines are blocked before the critical section
	mu.Lock()
	assert.Empty(t, sharedResource, "No goroutine should have entered critical section yet")
	mu.Unlock()

	// Now proceed them one by one to control the order
	breakpoint.Proceed("before-critical")

	// Wait for all to complete
	wg.Wait()

	// Verify all goroutines completed
	mu.Lock()
	assert.Len(t, sharedResource, numGoroutines, "All goroutines should have completed")
	mu.Unlock()
}

func TestProducerConsumerExample(t *testing.T) {
	breakpoint.ResetAll()

	// Enable specific breakpoints to control execution flow
	breakpoint.Enable("producer-start")
	breakpoint.Enable("consumer-start")

	ch := make(chan int, 10)
	var results []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Start producers and consumer
	wg.Add(3) // 2 producers + 1 consumer

	go func() {
		defer wg.Done()
		producer(ch, 1)
	}()

	go func() {
		defer wg.Done()
		producer(ch, 2)
	}()

	go func() {
		defer wg.Done()
		consumer(ch, &results, &mu)
	}()

	// Wait for all to hit their start breakpoints
	err := breakpoint.WaitForBreakpoint("producer-start", time.Second)
	require.NoError(t, err)
	err = breakpoint.WaitForBreakpoint("consumer-start", time.Second)
	require.NoError(t, err)

	// First, let consumer start
	breakpoint.Proceed("consumer-start")

	// Then let producers start
	breakpoint.Proceed("producer-start")

	// Let everything proceed
	breakpoint.ProceedAll()

	// Wait for completion
	wg.Wait()
	close(ch)

	// Verify results
	mu.Lock()
	assert.Len(t, results, 6, "Should have received 6 items")
	mu.Unlock()
}

func TestDeadlockDetectionExample(t *testing.T) {
	breakpoint.ResetAll()

	// Enable breakpoints to simulate potential deadlock scenario
	breakpoint.Enable("acquire-lock1")
	breakpoint.Enable("acquire-lock2")

	var lock1, lock2 sync.Mutex
	var wg sync.WaitGroup

	// Function that acquires locks in one order
	goroutine1 := func() {
		defer wg.Done()

		breakpoint.AddBreaker("acquire-lock1")
		lock1.Lock()

		breakpoint.AddBreaker("acquire-lock2")
		lock2.Lock()

		// Do some work
		time.Sleep(10 * time.Millisecond)

		lock2.Unlock()
		lock1.Unlock()
	}

	// Function that acquires locks in reverse order (potential deadlock)
	goroutine2 := func() {
		defer wg.Done()

		breakpoint.AddBreaker("acquire-lock2")
		lock2.Lock()

		breakpoint.AddBreaker("acquire-lock1")
		lock1.Lock()

		// Do some work
		time.Sleep(10 * time.Millisecond)

		lock1.Unlock()
		lock2.Unlock()
	}

	wg.Add(2)
	go goroutine1()
	go goroutine2()

	// Wait for both to hit their first breakpoints
	err := breakpoint.WaitForBreakpoint("acquire-lock1", time.Second)
	require.NoError(t, err)
	err = breakpoint.WaitForBreakpoint("acquire-lock2", time.Second)
	require.NoError(t, err)

	// Control the order to avoid deadlock
	// Let goroutine1 acquire both locks first
	breakpoint.Proceed("acquire-lock1")

	// Wait a bit, then proceed goroutine2
	time.Sleep(50 * time.Millisecond)
	breakpoint.Proceed("acquire-lock2")

	// Proceed any remaining breakpoints
	breakpoint.ProceedAll()

	// This should complete without deadlock
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(2 * time.Second):
		t.Fatal("Potential deadlock detected - test didn't complete in time")
	}
}

func TestComplexWorkflowExample(t *testing.T) {
	breakpoint.ResetAll()

	// Simulate a complex workflow with multiple stages
	stages := []string{"init", "validate", "process", "finalize"}

	for _, stage := range stages {
		breakpoint.Enable(stage)
	}

	workflow := func(id int, results *[]string, mu *sync.Mutex) {
		defer func() {
			mu.Lock()
			*results = append(*results, fmt.Sprintf("worker-%d-completed", id))
			mu.Unlock()
		}()

		for _, stage := range stages {
			breakpoint.AddBreaker(stage)

			// Simulate work for each stage
			mu.Lock()
			*results = append(*results, fmt.Sprintf("worker-%d-%s", id, stage))
			mu.Unlock()
		}
	}

	var results []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	numWorkers := 3
	wg.Add(numWorkers)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer wg.Done()
			workflow(id, &results, &mu)
		}(i)
	}

	// Control execution stage by stage
	for _, stage := range stages {
		// Wait for all workers to reach this stage
		for i := 0; i < numWorkers; i++ {
			err := breakpoint.WaitForBreakpoint(stage, time.Second)
			require.NoError(t, err)
		}

		// Let all workers proceed through this stage
		breakpoint.Proceed(stage)

		// Small delay to let stage complete
		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()

	// Verify all workers completed all stages
	mu.Lock()
	defer mu.Unlock()

	// Should have 4 stage entries + 1 completion entry per worker
	expectedEntries := numWorkers * (len(stages) + 1)
	assert.Len(t, results, expectedEntries)

	// Verify all workers completed
	completedCount := 0
	for _, result := range results {
		if fmt.Sprintf("%s", result)[len(result)-9:] == "completed" {
			completedCount++
		}
	}
	assert.Equal(t, numWorkers, completedCount, "All workers should have completed")
}
