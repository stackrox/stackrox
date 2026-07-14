package cmd

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMappingRefresher() *mappingRefresher {
	return newMappingRefresher("", "")
}

// countingFetchFn returns a fetchFn that fails failures times before
// succeeding.
func countingFetchFn(calls *int, failures int) func(context.Context) error {
	return func(context.Context) error {
		*calls++
		if *calls <= failures {
			return errors.New("transient fetch error")
		}
		return nil
	}
}

func TestMappingRefresher_FetchOnce(t *testing.T) {
	t.Run("should succeed without retrying when the first attempt succeeds", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()
			var calls int
			m.fetchFn = countingFetchFn(&calls, 0)

			err := m.fetchOnce(t.Context())
			require.NoError(t, err)
			assert.Equal(t, 1, calls)
		})
	})

	t.Run("should succeed after retrying a transient failure", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()
			var calls int
			m.fetchFn = countingFetchFn(&calls, mappingFetchMaxAttempts-1)

			err := m.fetchOnce(t.Context())
			require.NoError(t, err)
			assert.Equal(t, mappingFetchMaxAttempts, calls)
		})
	})

	t.Run("should fail after exhausting all attempts", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()
			var calls int
			m.fetchFn = countingFetchFn(&calls, mappingFetchMaxAttempts+10)

			err := m.fetchOnce(t.Context())
			require.Error(t, err)
			assert.Equal(t, mappingFetchMaxAttempts, calls, "should not retry past mappingFetchMaxAttempts")
		})
	})

	t.Run("should stop retrying promptly when the context is cancelled mid-backoff", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()
			var calls int
			m.fetchFn = countingFetchFn(&calls, mappingFetchMaxAttempts)

			ctx, cancel := context.WithCancel(t.Context())
			go func() {
				time.Sleep(10 * time.Millisecond) // fires well before mappingFetchBaseBackoff elapses
				cancel()
			}()

			err := m.fetchOnce(ctx)
			require.ErrorIs(t, err, context.Canceled)
			// retry.WithRetry always makes one more fetchFn call right
			// after a BetweenAttempts-observed cancellation (see
			// fetchOnce's doc comment) before its own next-iteration
			// check catches ctx being done, so 2 calls - not 1 - is
			// exactly the expected, harmless outcome here.
			assert.Equal(t, 2, calls, "should not attempt a second retry beyond the one already in flight when ctx is cancelled")
		})
	})
}

// totalMappingFetchBackoff mirrors fetchOnce's own backoff arithmetic to
// compute exactly how much (fake) time a fully-failing fetchOnce call
// spends waiting between attempts. synctest.Wait alone can't skip over
// this: it returns as soon as Run is durably blocked on any timer,
// including the ones fetchOnce blocks on mid-cascade, not only once the
// whole cascade has settled. Tests sleep this exact amount - derived from
// the real constants, not a guessed margin - to get past it before
// asserting on Run's rescheduling.
func totalMappingFetchBackoff() time.Duration {
	var total time.Duration
	backoff := mappingFetchBaseBackoff
	for attempt := 1; attempt < mappingFetchMaxAttempts; attempt++ {
		total += backoff
		backoff *= 2
	}
	return total
}

func TestMappingRefresher_Run(t *testing.T) {
	t.Run("should refresh on each tick", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()
			ticker := newFakeTicker()
			m.newTick = ticker.newTick
			var mu sync.Mutex
			var calls int
			m.fetchFn = func(context.Context) error {
				return concurrency.WithLock1(&mu, func() error { calls++; return nil })
			}

			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan struct{})
			go func() { defer close(done); m.Run(ctx) }()
			synctest.Wait() // Run is blocked waiting for the first tick

			ticker.fire()
			synctest.Wait()
			assert.Equal(t, 1, concurrency.WithLock1(&mu, func() int { return calls }))

			ticker.fire()
			synctest.Wait()
			assert.Equal(t, 2, concurrency.WithLock1(&mu, func() int { return calls }), "should refresh again on the next tick")

			cancel()
			<-done
		})
	})

	t.Run("should keep rescheduling on the same interval after a failed refresh", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()
			ticker := newFakeTicker()
			m.newTick = ticker.newTick
			m.fetchFn = func(context.Context) error { return errors.New("persistent fetch error") }

			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan struct{})
			go func() { defer close(done); m.Run(ctx) }()
			synctest.Wait()

			ticker.fire()
			time.Sleep(totalMappingFetchBackoff()) // let fetchOnce exhaust its own retries
			synctest.Wait()                        // let Run finish rescheduling

			assert.Equal(t, mappingRefreshInterval, ticker.lastReset(),
				"a failed refresh is not fatal: scans keep using the last successfully fetched file")

			cancel()
			<-done
		})
	})

	t.Run("should stop promptly when the context is cancelled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()

			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan struct{})
			go func() { defer close(done); m.Run(ctx) }()

			cancel()
			<-done
		})
	})
}

// TestMappingRefresher_Fetch exercises the real fetch method end to end
// (HTTP GET plus atomic file publish), unlike the fetchOnce/Run tests
// above, which inject fetchFn to isolate retry/scheduling behavior from
// actual network and filesystem I/O.
func TestMappingRefresher_Fetch(t *testing.T) {
	t.Run("should publish the response body to cachePath", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"data":{}}`))
		}))
		defer srv.Close()

		cachePath := filepath.Join(t.TempDir(), "repo2cpe.json")
		m := newMappingRefresher(srv.URL, cachePath)

		require.NoError(t, m.fetch(t.Context()))

		got, err := os.ReadFile(cachePath)
		require.NoError(t, err)
		assert.Equal(t, `{"data":{}}`, string(got))
	})

	t.Run("should leave any previously cached file untouched on a failed fetch", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		cachePath := filepath.Join(t.TempDir(), "repo2cpe.json")
		require.NoError(t, os.WriteFile(cachePath, []byte("stale"), 0o600))
		m := newMappingRefresher(srv.URL, cachePath)

		require.Error(t, m.fetch(t.Context()))

		got, err := os.ReadFile(cachePath)
		require.NoError(t, err)
		assert.Equal(t, "stale", string(got))
	})
}
