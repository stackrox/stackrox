package safe

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// Channel provides a thread-safe channel with panic-free shutdown semantics.
// It encapsulates a channel along with synchronization primitives to prevent
// panics from sending on a closed channel during concurrent shutdown scenarios.
// The channel is automatically closed when the waitable is triggered.
// Write rejection after shutdown is best-effort - writes in progress may succeed.
type Channel[T any] struct {
	mu       sync.RWMutex
	ch       chan T
	waitable concurrency.Waitable
}

// NewChannel creates a new Channel with the specified buffer size.
// The waitable parameter is used to coordinate shutdown - writes will make
// best-effort to fail when the waitable is triggered, and the channel will be
// automatically closed. Writes already in progress may complete successfully.
// Panics if waitable is nil or size is negative.
func NewChannel[T any](size int, waitable concurrency.Waitable) *Channel[T] {
	if waitable == nil {
		panic("waitable must not be nil")
	}
	if waitable.Done() == nil {
		panic("waitable.Done() must not be nil")
	}
	if size < 0 {
		panic("size must not be negative")
	}
	c := &Channel[T]{
		ch:       make(chan T, size),
		waitable: waitable,
	}

	// Spawn a goroutine that will close the channel when the waitable is triggered.
	// The lock ensures all in-flight writes complete before closing.
	go func() {
		<-waitable.Done()
		c.mu.Lock()
		defer c.mu.Unlock()
		close(c.ch)
	}()

	return c
}

// Write pushes an item to the channel, blocking if the channel is full.
// This operation is safe to call concurrently with channel closure (no panics).
//
// Returns ErrWaitableTriggered if the waitable is triggered before the write starts
// or while blocked waiting for channel space. Writes already holding the lock when
// the waitable triggers may complete successfully (best-effort rejection).
//
// Thread-safety and double-select pattern:
//
// The RLock is required because Write/TryWrite calls may occur in different goroutines
// from the auto-close goroutine, and not all Write/TryWrite calls are in the same goroutine either.
// RLock is sufficient (rather than full Lock) because writing to a channel is already
// thread-safe in Go; the lock only coordinates shutdown with the auto-close goroutine.
//
// The double-select pattern prevents panics when writing to a closed channel:
//
//  1. Caller A: Write() -> acquires RLock
//  2. Auto-close goroutine: waitable triggers -> waits for lock (blocked by A's RLock)
//  3. Caller A: Write() ends -> releases RLock
//  4. Auto-close goroutine: acquires lock -> closes channel -> releases lock
//  5. Caller C: Write() -> acquires RLock -> first select detects triggered waitable -> exits early
//
// Without the first select (fast-path check), we would proceed to the second select where
// Go's select would randomly choose between the waitable channel and writing to the closed
// channel, potentially causing a panic.
//
// The second select is needed because if we're blocked waiting to write to a full channel
// and the waitable is triggered, we should immediately stop trying to write and exit.
func (s *Channel[T]) Write(item T) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// First select: fast-path exit if waitable is already triggered
	select {
	case <-s.waitable.Done():
		return ErrWaitableTriggered
	default:
	}

	// Second select: exit if waitable is triggered while blocked on channel write
	select {
	case <-s.waitable.Done():
		return ErrWaitableTriggered
	case s.ch <- item:
		return nil
	}
}

// TryWrite attempts to push an item to the channel without blocking.
// If the channel is full, it returns ErrChannelFull immediately.
// This operation is safe to call concurrently with channel closure (no panics).
//
// Returns:
//   - ErrWaitableTriggered if the waitable is triggered before the write starts (best-effort)
//   - ErrChannelFull if the channel is full and cannot accept the item
//
// Thread-safety and double-select pattern:
//
// The RLock is required because Write/TryWrite calls may occur in different goroutines
// from the auto-close goroutine, and not all Write/TryWrite calls are in the same goroutine either.
// RLock is sufficient (rather than full Lock) because writing to a channel is already
// thread-safe in Go; the lock only coordinates shutdown with the auto-close goroutine.
//
// The double-select pattern prevents panics when writing to a closed channel:
// See the Write function documentation for a detailed explanation of the race condition
// this pattern prevents.
func (s *Channel[T]) TryWrite(item T) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// First select: fast-path exit if waitable is already triggered
	select {
	case <-s.waitable.Done():
		return ErrWaitableTriggered
	default:
	}

	// Second select: exit if waitable is triggered, or return ErrChannelFull if full
	select {
	case <-s.waitable.Done():
		return ErrWaitableTriggered
	case s.ch <- item:
		return nil
	default:
		return ErrChannelFull
	}
}

// Chan returns a read-only view of the underlying channel.
// This can be used in select statements or to read from the channel.
func (s *Channel[T]) Chan() <-chan T {
	return s.ch
}

// Len returns the number of items currently in the channel.
func (s *Channel[T]) Len() int {
	return len(s.ch)
}

// Cap returns the capacity of the channel.
func (s *Channel[T]) Cap() int {
	return cap(s.ch)
}
