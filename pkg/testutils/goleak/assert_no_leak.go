package goleak

import (
	"testing"

	"go.uber.org/goleak"
)

// Option re-exports goleak.Option so callers don't need to import
// go.uber.org/goleak directly (disallowed by roxvet, see validateimports).
type Option = goleak.Option

// IgnoreCurrent re-exports goleak.IgnoreCurrent, for tests that need to check
// for leaks before test cleanup runs rather than via AssertNoGoroutineLeaks.
func IgnoreCurrent() Option {
	return goleak.IgnoreCurrent()
}

// Find re-exports goleak.Find, for tests that need to check for leaks before
// test cleanup runs rather than via AssertNoGoroutineLeaks.
func Find(options ...Option) error {
	return goleak.Find(options...)
}

// CommonIgnores returns the goleak options for known, unrelated background
// goroutines. Shared so that tests calling IgnoreCurrent/Find directly stay in
// sync with AssertNoGoroutineLeaks instead of maintaining their own copy of
// this list.
func CommonIgnores() []Option {
	return []goleak.Option{
		// Ignore a known leak: https://github.com/DataDog/dd-trace-go/issues/1469
		goleak.IgnoreTopFunction("github.com/golang/glog.(*fileSink).flushDaemon"),
		// Ignore a known leak caused by importing the GCP cscc SDK.
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		// Ignore a known leak from https://github.com/hashicorp/golang-lru/blob/v2.0.7/expirable/expirable_lru.go#L77-L80
		goleak.IgnoreTopFunction("github.com/hashicorp/golang-lru/v2/expirable.NewLRU[...].func1"),
	}
}

func AssertNoGoroutineLeaks(t testing.TB) {
	t.Cleanup(func() {
		goleak.VerifyNone(t, CommonIgnores()...)
	})
}
