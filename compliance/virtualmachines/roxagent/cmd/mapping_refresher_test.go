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

func TestMappingRefresher_FetchWithRetry(t *testing.T) {
	t.Run("should succeed without retrying when the first attempt succeeds", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()
			var calls int
			m.fetchFn = countingFetchFn(&calls, 0)

			err := m.fetchWithRetry(t.Context())
			require.NoError(t, err)
			assert.Equal(t, 1, calls)
		})
	})

	t.Run("should succeed after retrying a transient failure", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()
			var calls int
			m.fetchFn = countingFetchFn(&calls, mappingFetchMaxAttempts-1)

			err := m.fetchWithRetry(t.Context())
			require.NoError(t, err)
			assert.Equal(t, mappingFetchMaxAttempts, calls)
		})
	})

	t.Run("should fail after exhausting all attempts", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()
			var calls int
			m.fetchFn = countingFetchFn(&calls, mappingFetchMaxAttempts+10)

			err := m.fetchWithRetry(t.Context())
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

			err := m.fetchWithRetry(ctx)
			require.ErrorIs(t, err, context.Canceled)
			assert.Equal(t, 1, calls, "should not attempt a retry once ctx is observed cancelled during backoff")
		})
	})
}

// totalMappingFetchBackoff mirrors fetchWithRetry's own backoff arithmetic to
// compute exactly how much (fake) time a fully-failing fetchWithRetry call
// spends waiting between attempts. synctest.Wait alone can't skip over
// this: it returns as soon as Run is durably blocked on any timer,
// including the ones fetchWithRetry blocks on mid-cascade, not only once the
// whole cascade has settled. Tests sleep this exact amount - derived from
// the real constants, not a guessed margin - to get past it before
// asserting on Run's next tick.
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
			ticks := make(chan time.Time)
			defer close(ticks)
			m.ticks = ticks
			var mu sync.Mutex
			var calls int
			m.fetchFn = func(context.Context) error {
				return concurrency.WithLock1(&mu, func() error { calls++; return nil })
			}

			ctx, cancel := context.WithCancel(t.Context())
			stopped := m.runAsync(ctx)
			// Stop Run before close(ticks): a closed ticks chan would make
			// select fire continuously with zero values.
			defer func() {
				cancel()
				<-stopped
			}()
			synctest.Wait() // Run is blocked waiting for the first tick

			ticks <- time.Time{}
			synctest.Wait()
			assert.Equal(t, 1, concurrency.WithLock1(&mu, func() int { return calls }))

			ticks <- time.Time{}
			synctest.Wait()
			assert.Equal(t, 2, concurrency.WithLock1(&mu, func() int { return calls }), "should refresh again on the next tick")
		})
	})

	t.Run("should keep ticking after a failed refresh", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()
			ticks := make(chan time.Time)
			defer close(ticks)
			m.ticks = ticks
			var mu sync.Mutex
			var calls int
			m.fetchFn = func(context.Context) error {
				return concurrency.WithLock1(&mu, func() error {
					calls++
					return errors.New("persistent fetch error")
				})
			}

			ctx, cancel := context.WithCancel(t.Context())
			stopped := m.runAsync(ctx)
			defer func() {
				cancel()
				<-stopped
			}()
			synctest.Wait()

			ticks <- time.Time{}
			time.Sleep(totalMappingFetchBackoff()) // let fetchWithRetry exhaust its own retries
			synctest.Wait()

			ticks <- time.Time{}
			time.Sleep(totalMappingFetchBackoff())
			synctest.Wait()

			assert.Equal(t, 2*mappingFetchMaxAttempts, concurrency.WithLock1(&mu, func() int { return calls }),
				"a failed refresh is not fatal: Run keeps accepting ticks and scans keep using the last file")
		})
	})

	t.Run("should stop promptly when the context is cancelled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := testMappingRefresher()

			ctx, cancel := context.WithCancel(t.Context())
			stopped := m.runAsync(ctx)

			cancel()
			<-stopped
		})
	})
}

// TestMappingRefresher_Fetch exercises the real fetch method end to end
// (HTTP GET plus atomic file publish), unlike the fetchWithRetry/Run tests
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
