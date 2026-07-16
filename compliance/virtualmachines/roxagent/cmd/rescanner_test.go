package cmd

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

// testRescanner returns a rescanner with a long default interval so tests
// that don't care about the periodic loop never trigger it by accident.
func testRescanner() *rescanner {
	return newRescanner(&vsockserver.ReportCache{}, "", "", time.Hour)
}

// fakeTicker is a newTick func driven manually by a test: fire triggers a
// tick, and lastReset reports the duration most recently requested via
// newTick, letting tests assert on scheduling decisions directly instead of
// on elapsed time. Pair with synctest.Wait after fire to block until the
// loop under test has processed the tick and settled back into waiting for
// the next one.
type fakeTicker struct {
	tick chan time.Time

	mu     sync.Mutex
	resets []time.Duration
}

func newFakeTicker() *fakeTicker {
	return &fakeTicker{tick: make(chan time.Time, 1)}
}

func (f *fakeTicker) close() { close(f.tick) }

// newTick has the same signature as time.After, so it's directly assignable
// to a rescanner's newTick field.
func (f *fakeTicker) newTick(d time.Duration) <-chan time.Time {
	concurrency.WithLock(&f.mu, func() { f.resets = append(f.resets, d) })
	return f.tick
}

func (f *fakeTicker) fire() { f.tick <- time.Time{} }

func (f *fakeTicker) lastReset() time.Duration {
	return concurrency.WithLock1(&f.mu, func() time.Duration {
		if len(f.resets) == 0 {
			return 0
		}
		return f.resets[len(f.resets)-1]
	})
}

func TestRescanner_Run(t *testing.T) {
	t.Run("should publish to cache on each tick", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := testRescanner()
			ticker := newFakeTicker()
			defer ticker.close()
			r.newTick = ticker.newTick
			var mu sync.Mutex
			var calls int
			r.scanFn = func(_ context.Context, _, _ string) (*v4.IndexReport, error) {
				return concurrency.WithLock2(&mu, func() (*v4.IndexReport, error) {
					calls++
					return &v4.IndexReport{HashId: "ok"}, nil
				})
			}

			ctx, cancel := context.WithCancel(t.Context())
			stopped := r.runAsync(ctx)
			// Stop Run before close(tick): a closed tick chan would make
			// select fire continuously with zero values.
			defer func() {
				cancel()
				<-stopped
			}()
			synctest.Wait() // Run is blocked waiting for the first tick

			ticker.fire()
			synctest.Wait()
			assert.Equal(t, 1, concurrency.WithLock1(&mu, func() int { return calls }))

			ticker.fire()
			synctest.Wait()
			assert.Equal(t, 2, concurrency.WithLock1(&mu, func() int { return calls }), "should rescan again on the next tick")
		})
	})

	t.Run("should retry sooner than the full interval after a failed rescan", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := testRescanner()
			ticker := newFakeTicker()
			defer ticker.close()
			r.newTick = ticker.newTick
			var mu sync.Mutex
			var calls int
			r.scanFn = func(_ context.Context, _, _ string) (*v4.IndexReport, error) {
				return concurrency.WithLock2(&mu, func() (*v4.IndexReport, error) {
					calls++
					if calls == 1 {
						return nil, errors.New("transient scan error")
					}
					return &v4.IndexReport{HashId: "ok"}, nil
				})
			}

			ctx, cancel := context.WithCancel(t.Context())
			stopped := r.runAsync(ctx)
			defer func() {
				cancel()
				<-stopped
			}()
			synctest.Wait() // Run is blocked waiting for the first tick

			ticker.fire()
			synctest.Wait()

			assert.Equal(t, rescanRetryBaseBackoff, ticker.lastReset(),
				"a failed rescan should be rescheduled after rescanRetryBaseBackoff, not r.interval")
			assert.Equal(t, 1, concurrency.WithLock1(&mu, func() int { return calls }))

			ticker.fire() // the rescheduled retry firing
			synctest.Wait()

			assert.Equal(t, 2, concurrency.WithLock1(&mu, func() int { return calls }), "the failed rescan was never retried")
			assert.Equal(t, r.interval, ticker.lastReset(), "a successful rescan should reschedule after the full interval, resetting backoff")
		})
	})

	t.Run("should stop promptly when the context is cancelled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := testRescanner()

			ctx, cancel := context.WithCancel(t.Context())
			stopped := r.runAsync(ctx)

			// Run must return promptly once ctx is cancelled: if it
			// doesn't, the bubble deadlocks on the blocked <-stopped below
			// (nothing left to advance the fake clock), and synctest.Test
			// fails the test on deadlock automatically.
			cancel()
			<-stopped
		})
	})
}
