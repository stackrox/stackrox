# Breakpoint Testing Framework

A powerful testing and debugging framework for Go that allows you to control parallel execution flow and test race conditions by adding breakpoints to your code.

## Overview

The breakpoint framework enables you to:
- Add breakpoints anywhere in your code using `AddBreaker(name)`
- Control execution flow by enabling/disabling breakpoints
- Proceed breakpoints in a desired order to test race conditions
- Wait for breakpoints to be hit for test orchestration
- Test complex concurrent scenarios with precise timing control

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "sync"
    "time"
    
    "github.com/stackrox/rox/pkg/breakpoint"
)

func workerFunction(id int) {
    fmt.Printf("Worker %d starting\n", id)
    
    // Add a breakpoint before critical section
    breakpoint.AddBreaker("before-critical")
    
    // Critical section
    fmt.Printf("Worker %d in critical section\n", id)
    
    // Add a breakpoint after critical section
    breakpoint.AddBreaker("after-critical")
    
    fmt.Printf("Worker %d finished\n", id)
}

func main() {
    // Clean state
    breakpoint.ResetAll()
    
    // Enable the breakpoint we want to control
    breakpoint.Enable("before-critical")
    
    var wg sync.WaitGroup
    
    // Start multiple workers
    for i := 1; i <= 3; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            workerFunction(id)
        }(i)
    }
    
    // Wait for all workers to hit the breakpoint
    for i := 0; i < 3; i++ {
        breakpoint.WaitForBreakpoint("before-critical", time.Second)
    }
    
    fmt.Println("All workers are blocked at breakpoint")
    
    // Let them proceed one by one
    breakpoint.Proceed("before-critical")
    
    wg.Wait()
    fmt.Println("All workers completed")
}
```

## API Reference

### Core Functions

#### `AddBreaker(name string)`
Adds a breakpoint in your code. If the breakpoint is enabled, execution will pause until `Proceed` is called.

```go
breakpoint.AddBreaker("my-breakpoint")
```

#### `Enable(name string)`
Enables a specific breakpoint. Breakpoints are disabled by default.

```go
breakpoint.Enable("my-breakpoint")
```

#### `Proceed(name string)`
Allows a specific breakpoint to continue execution.

```go
breakpoint.Proceed("my-breakpoint")
```

#### `WaitForBreakpoint(name string, timeout time.Duration) error`
Waits for a breakpoint to be hit within the specified timeout.

```go
err := breakpoint.WaitForBreakpoint("my-breakpoint", time.Second)
if err != nil {
    // Handle timeout or error
}
```

### Utility Functions

#### `EnableAll()` / `DisableAll()`
Enable or disable all registered breakpoints.

#### `ProceedAll()`
Allow all breakpoints to continue execution.

#### `Reset(name string)` / `ResetAll()`
Reset specific or all breakpoints to their initial state.

#### `List() []string`
Get information about all registered breakpoints.

#### `IsHit(name string) (bool, error)`
Check if a breakpoint has been hit.

#### `IsEnabled(name string) (bool, error)`
Check if a breakpoint is enabled.

## Testing Patterns

### 1. Race Condition Testing

```go
func TestRaceCondition(t *testing.T) {
    breakpoint.ResetAll()
    breakpoint.Enable("critical-section")
    
    var sharedResource int
    var wg sync.WaitGroup
    
    // Start competing goroutines
    for i := 0; i < 3; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            breakpoint.AddBreaker("critical-section")
            sharedResource++ // Race condition here
        }()
    }
    
    // Wait for all to reach the breakpoint
    for i := 0; i < 3; i++ {
        breakpoint.WaitForBreakpoint("critical-section", time.Second)
    }
    
    // Control the order - let them proceed one by one
    breakpoint.Proceed("critical-section")
    
    wg.Wait()
    assert.Equal(t, 3, sharedResource)
}
```

### 2. Producer-Consumer Testing

```go
func TestProducerConsumer(t *testing.T) {
    breakpoint.ResetAll()
    breakpoint.Enable("producer-ready")
    breakpoint.Enable("consumer-ready")
    
    ch := make(chan int, 1)
    
    // Producer
    go func() {
        breakpoint.AddBreaker("producer-ready")
        ch <- 42
    }()
    
    // Consumer
    go func() {
        breakpoint.AddBreaker("consumer-ready")
        value := <-ch
        assert.Equal(t, 42, value)
    }()
    
    // Control execution order
    breakpoint.WaitForBreakpoint("producer-ready", time.Second)
    breakpoint.WaitForBreakpoint("consumer-ready", time.Second)
    
    // Let producer go first
    breakpoint.Proceed("producer-ready")
    time.Sleep(10 * time.Millisecond)
    
    // Then consumer
    breakpoint.Proceed("consumer-ready")
}
```

### 3. Deadlock Prevention Testing

```go
func TestDeadlockPrevention(t *testing.T) {
    breakpoint.ResetAll()
    breakpoint.Enable("acquire-lock1")
    breakpoint.Enable("acquire-lock2")
    
    var lock1, lock2 sync.Mutex
    
    // Goroutine 1: lock1 -> lock2
    go func() {
        breakpoint.AddBreaker("acquire-lock1")
        lock1.Lock()
        defer lock1.Unlock()
        
        breakpoint.AddBreaker("acquire-lock2")
        lock2.Lock()
        defer lock2.Unlock()
    }()
    
    // Goroutine 2: lock2 -> lock1 (potential deadlock)
    go func() {
        breakpoint.AddBreaker("acquire-lock2")
        lock2.Lock()
        defer lock2.Unlock()
        
        breakpoint.AddBreaker("acquire-lock1")
        lock1.Lock()
        defer lock1.Unlock()
    }()
    
    // Wait for both to reach their first locks
    breakpoint.WaitForBreakpoint("acquire-lock1", time.Second)
    breakpoint.WaitForBreakpoint("acquire-lock2", time.Second)
    
    // Control order to prevent deadlock
    breakpoint.Proceed("acquire-lock1") // Let first goroutine get both locks
    time.Sleep(50 * time.Millisecond)
    breakpoint.Proceed("acquire-lock2") // Then let second goroutine proceed
    
    breakpoint.ProceedAll() // Clean up any remaining breakpoints
}
```

## Integration Examples

The framework can be integrated with existing code to test complex scenarios:

### Worker Pool Testing

```go
func TestWorkerPoolConcurrency(t *testing.T) {
    breakpoint.ResetAll()
    breakpoint.Enable("job-start")
    
    pool := concurrency.NewWorkerPool(2)
    pool.Start()
    defer pool.Stop()
    
    var results []int
    var mu sync.Mutex
    
    // Add jobs with breakpoints
    for i := 1; i <= 4; i++ {
        pool.AddJob(func(id int) func() {
            return func() {
                breakpoint.AddBreaker("job-start")
                mu.Lock()
                results = append(results, id)
                mu.Unlock()
            }
        }(i))
    }
    
    // Control job execution order
    breakpoint.WaitForBreakpoint("job-start", time.Second)
    breakpoint.WaitForBreakpoint("job-start", time.Second)
    
    // Only 2 jobs should be running (pool capacity)
    breakpoint.Proceed("job-start")
    
    // Test continues...
}
```

## Best Practices

### 1. Always Reset State
Start each test with `breakpoint.ResetAll()` to ensure clean state:

```go
func TestSomething(t *testing.T) {
    breakpoint.ResetAll() // Always start with clean state
    // ... test code
}
```

### 2. Use Descriptive Names
Use clear, descriptive names for breakpoints:

```go
// Good
breakpoint.AddBreaker("before-database-write")
breakpoint.AddBreaker("after-cache-update")

// Avoid
breakpoint.AddBreaker("bp1")
breakpoint.AddBreaker("x")
```

### 3. Handle Timeouts
Always handle timeouts when waiting for breakpoints:

```go
err := breakpoint.WaitForBreakpoint("my-breakpoint", time.Second)
if err != nil {
    t.Fatalf("Timeout waiting for breakpoint: %v", err)
}
```

### 4. Clean Up
Use `defer` or explicit cleanup to ensure breakpoints don't interfere with other tests:

```go
func TestSomething(t *testing.T) {
    breakpoint.ResetAll()
    defer breakpoint.ResetAll() // Clean up after test
    
    // ... test code
}
```

## Thread Safety

The framework is fully thread-safe and can be used safely from multiple goroutines. All operations are protected by appropriate synchronization primitives.

## Performance Considerations

- Breakpoints have minimal overhead when disabled
- Enabled breakpoints add synchronization overhead
- Use breakpoints primarily in test code, not production code
- Reset breakpoints between tests to avoid memory leaks

## Debugging Tips

### List Active Breakpoints
```go
breakpoints := breakpoint.List()
for _, bp := range breakpoints {
    fmt.Println(bp)
}
```

### Check Breakpoint Status
```go
enabled, _ := breakpoint.IsEnabled("my-breakpoint")
hit, _ := breakpoint.IsHit("my-breakpoint")
fmt.Printf("Breakpoint enabled: %t, hit: %t\n", enabled, hit)
```

## Limitations

- Breakpoints persist globally across the process
- Not suitable for production use
- Requires careful test isolation
- May impact performance when enabled

## Contributing

When adding new features:
1. Add comprehensive tests
2. Update this documentation
3. Follow the existing code patterns
4. Ensure thread safety
