package channel

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// SafeChannel provides a thread-safe channel with race-free shutdown semantics.
// It encapsulates a channel along with synchronization primitives to ensure
// safe writes and closure even during concurrent shutdown scenarios.
type SafeChannel[T any] struct {
	mu       sync.Mutex
	ch       chan T
	closed   bool
	waitable concurrency.Waitable
}

// NewSafeChannel creates a new SafeChannel with the specified buffer size.
// The waitable parameter is used to coordinate shutdown - writes will fail
// when the waitable is triggered.
// If size is negative, it is treated as 0 (unbuffered channel).
func NewSafeChannel[T any](size int, waitable concurrency.Waitable) *SafeChannel[T] {
	if size < 0 {
		size = 0
	}
	return &SafeChannel[T]{
		ch:       make(chan T, size),
		waitable: waitable,
	}
}

// Write pushes an item to the channel, blocking if the channel is full.
// This operation is safe to call concurrently with Close.
//
// Returns ErrWaitableTriggered if the waitable is triggered before or during the write.
func (s *SafeChannel[T]) Write(item T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// First select will exit early if waitable is already triggered
	select {
	case <-s.waitable.Done():
		return ErrWaitableTriggered
	default:
	}

	// Second select will exit if we are blocked waiting to write to the channel
	select {
	case <-s.waitable.Done():
		return ErrWaitableTriggered
	case s.ch <- item:
		return nil
	}
}

// TryWrite attempts to push an item to the channel without blocking.
// If the channel is full, it returns ErrChannelFull immediately.
// This operation is safe to call concurrently with Close.
//
// Returns:
//   - ErrWaitableTriggered if the waitable is triggered before or during the write
//   - ErrChannelFull if the channel is full and cannot accept the item
func (s *SafeChannel[T]) TryWrite(item T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// First select will exit early if waitable is already triggered
	select {
	case <-s.waitable.Done():
		return ErrWaitableTriggered
	default:
	}

	// Second select will exit if we are blocked waiting to write to the channel
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
func (s *SafeChannel[T]) Chan() <-chan T {
	return s.ch
}

// Len returns the number of items currently in the channel.
func (s *SafeChannel[T]) Len() int {
	return len(s.ch)
}

// Cap returns the capacity of the channel.
func (s *SafeChannel[T]) Cap() int {
	return cap(s.ch)
}

// Close safely closes the underlying channel.
// This should be called after the waitable has been triggered.
// It is safe to call Close multiple times - subsequent calls are no-ops.
//
// Proper shutdown sequence:
//  1. Signal the waitable
//  3. Call Close()
func (s *SafeChannel[T]) Close() {
	<-s.waitable.Done()
	concurrency.WithLock(&s.mu, func() {
		if !s.closed {
			close(s.ch)
			s.closed = true
		}
	})
}
