package saml

import (
	"sync/atomic"
	"time"
)

// testClock provides a controllable clock for tests
type testClock struct {
	now atomic.Value // *time.Time
}

func newTestClock(initial time.Time) *testClock {
	tc := &testClock{}
	tc.now.Store(&initial)
	return tc
}

func (tc *testClock) Now() time.Time {
	return *tc.now.Load().(*time.Time)
}

func (tc *testClock) SetTime(t time.Time) {
	tc.now.Store(&t)
}
