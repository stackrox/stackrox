package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/stackrox/rox/pkg/breakpoint"
)

// simulateWorker represents a worker that performs some task
func simulateWorker(id int, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("Worker %d: Starting\n", id)

	// Add breakpoint before critical section
	breakpoint.AddBreaker("before-work")

	fmt.Printf("Worker %d: Doing critical work\n", id)
	time.Sleep(100 * time.Millisecond) // Simulate work

	// Add breakpoint after critical section
	breakpoint.AddBreaker("after-work")

	fmt.Printf("Worker %d: Finished\n", id)
}

func main() {
	fmt.Println("=== Breakpoint Framework Demo ===")

	// Clean state
	breakpoint.ResetAll()

	fmt.Println("1. Starting workers without breakpoints (normal execution)...")
	var wg sync.WaitGroup

	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go simulateWorker(i, &wg)
	}
	wg.Wait()

	fmt.Println("\n2. Now with breakpoint control...")
	breakpoint.ResetAll()

	// Enable breakpoints
	breakpoint.Enable("before-work")
	breakpoint.Enable("after-work")

	fmt.Println("Starting workers with breakpoints enabled...")

	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go simulateWorker(i, &wg)
	}

	// Wait for all workers to hit the first breakpoint
	fmt.Println("Waiting for all workers to reach 'before-work' breakpoint...")
	for i := 0; i < 3; i++ {
		err := breakpoint.WaitForBreakpoint("before-work", 2*time.Second)
		if err != nil {
			fmt.Printf("Error waiting for breakpoint: %v\n", err)
			return
		}
	}

	fmt.Println("All workers are blocked at 'before-work' breakpoint!")
	fmt.Println("Proceeding workers one by one...")

	// Let them proceed one by one
	breakpoint.Proceed("before-work")

	// Wait for completion at second breakpoint
	for i := 0; i < 3; i++ {
		err := breakpoint.WaitForBreakpoint("after-work", 2*time.Second)
		if err != nil {
			fmt.Printf("Error waiting for breakpoint: %v\n", err)
			return
		}
	}

	fmt.Println("All workers completed their work and are at 'after-work' breakpoint!")
	fmt.Println("Letting all workers finish...")

	// Let them all finish
	breakpoint.ProceedAll()

	wg.Wait()

	fmt.Println("\n3. Breakpoint status:")
	breakpoints := breakpoint.List()
	for _, bp := range breakpoints {
		fmt.Printf("  %s\n", bp)
	}

	fmt.Println("\n=== Demo Complete ===")
}
