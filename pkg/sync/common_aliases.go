package sync

import "sync"

// Once is an alias for `sync.Once`.
type Once = sync.Once

// WaitGroup is an alias for `sync.WaitGroup`.
type WaitGroup = sync.WaitGroup

// Locker is an alias for `sync.Locker`.
type Locker = sync.Locker

// Map is an alias for `sync.Map`.
type Map = sync.Map

// Pool is an alias for `sync.Pool`.
type Pool = sync.Pool

// OnceValue returns a function that invokes f only once and returns the
// value returned by f. This is a re-export of sync.OnceValue to satisfy
// the project's "use pkg/sync instead of sync" lint rule.
func OnceValue[T any](f func() T) func() T {
	return sync.OnceValue(f)
}
