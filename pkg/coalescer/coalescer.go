package coalescer

import (
	"context"

	"golang.org/x/sync/singleflight"
)

// Coalescer coalesces concurrent calls with the same key into a single execution,
// respecting context cancellation for each caller independently.
//
// Unlike raw singleflight.Group, Coalescer:
//   - Respects per-caller context deadlines (callers can fail fast).
//   - Returns typed results without requiring type assertions.
type Coalescer[T any] struct {
	group singleflight.Group
}

// New creates a new Coalescer instance.
func New[T any]() *Coalescer[T] {
	return &Coalescer[T]{}
}

// Coalesce executes fn for the given key, coalescing concurrent calls.
// If the context is cancelled while waiting, the context error is returned.
// The underlying function continues executing for other waiters.
func (c *Coalescer[T]) Coalesce(ctx context.Context, key string, fn func() (T, error)) (T, error) {
	ch := c.group.DoChan(key, func() (interface{}, error) {
		return fn()
	})

	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case result := <-ch:
		if result.Err != nil {
			var zero T
			return zero, result.Err
		}
		return result.Val.(T), nil
	}
}

// Forget tells the coalescer to forget about a key, allowing a new call to start.
// This is useful for cache invalidation scenarios.
func (c *Coalescer[T]) Forget(key string) {
	c.group.Forget(key)
}
