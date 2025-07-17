package tlscheckcache

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

var (
	ctx = context.Background()
)

func TestCheckTLS(t *testing.T) {
	t.Run("secure", func(t *testing.T) {
		c := New(WithTLSCheckFunc(alwaysSecureCheckTLS)).(*cacheImpl)
		secure, skip, err := c.CheckTLS(ctx, "fake")
		assert.True(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, c.results.GetAll(), 1)

		// Ensure the results do not change when attempted again / using cache
		secure, skip, err = c.CheckTLS(ctx, "fake")
		assert.True(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, c.results.GetAll(), 1)
	})

	t.Run("insecure", func(t *testing.T) {
		c := New(WithTLSCheckFunc(alwaysInsecureCheckTLS)).(*cacheImpl)
		secure, skip, err := c.CheckTLS(ctx, "fake")
		assert.False(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, c.results.GetAll(), 1)

		// Ensure the results do not change when attempted again / using cache
		secure, skip, err = c.CheckTLS(ctx, "fake")
		assert.False(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, c.results.GetAll(), 1)
	})

	t.Run("error", func(t *testing.T) {
		c := New(WithTLSCheckFunc(alwaysFailCheckTLS)).(*cacheImpl)
		secure, skip, err := c.CheckTLS(ctx, "fake")
		assert.False(t, secure)
		assert.False(t, skip)
		assert.Error(t, err)
		assert.Len(t, c.results.GetAll(), 1)

		// Results expected to change, skip should be true due to previous error.
		secure, skip, err = c.CheckTLS(ctx, "fake")
		assert.False(t, secure)
		assert.True(t, skip)
		assert.NoError(t, err)
		assert.Len(t, c.results.GetAll(), 1)
	})
}

func TestAysncCheckTLS(t *testing.T) {
	callCounts := make(map[string]int)
	var callCountsMutex sync.Mutex

	countingCheckTLSFunc := func(_ context.Context, endpoint string) (bool, error) {
		callCountsMutex.Lock()
		defer callCountsMutex.Unlock()
		callCounts[endpoint]++
		return true, nil
	}
	regs := []string{"reg1", "reg2", "reg3"}

	c := New(WithTLSCheckFunc(countingCheckTLSFunc)).(*cacheImpl)
	runAsyncTLSChecks(c, regs)

	assert.Len(t, callCounts, len(regs))
	assert.Len(t, c.results.GetAll(), len(regs))
	// Ensure that the checkTLSFunc was not called more than once per endpoint.
	for _, reg := range callCounts {
		assert.Equal(t, 1, reg)
	}

	// Simulate cache expiry
	c.Cleanup()
	runAsyncTLSChecks(c, regs)

	assert.Len(t, callCounts, len(regs))
	assert.Len(t, c.results.GetAll(), len(regs))
	// Now the checkTLSFunc should have been called twice per endpoint.
	for _, reg := range callCounts {
		assert.Equal(t, 2, reg)
	}
}

func runAsyncTLSChecks(cache Cache, regs []string) {
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		for _, reg := range regs {
			wg.Add(1)
			go func(reg string) {
				_, _, _ = cache.CheckTLS(context.Background(), reg)
				wg.Done()
			}(reg)
		}
	}

	wg.Wait()
}

func TestMetrics(t *testing.T) {
	cache := New(WithTLSCheckFunc(alwaysSecureCheckTLS))
	_, _, err := cache.CheckTLS(ctx, "fake")

	c := tlsCheckCount
	// Counter metrics cannot be reset, so use the current
	// value as a base and test relative changes.
	base := testutil.ToFloat64(c)

	assert.NoError(t, err)
	assert.Equal(t, base, testutil.ToFloat64(c))

	_, _, err = cache.CheckTLS(ctx, "fake")
	assert.NoError(t, err)
	assert.Equal(t, base+1, testutil.ToFloat64(c))

	_, _, err = cache.CheckTLS(ctx, "fake")
	assert.NoError(t, err)
	assert.Equal(t, base+2, testutil.ToFloat64(c))
}

// alwaysInsecureCheckTLS adheres to CheckTLSFunc and always returns insecure.
func alwaysInsecureCheckTLS(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// alwaysSecureCheckTLS adheres to CheckTLSFunc and always returns secure.
func alwaysSecureCheckTLS(_ context.Context, _ string) (bool, error) {
	return true, nil
}

// alwaysFailCheckTLS adheres to CheckTLSFunc and always returns an error.
func alwaysFailCheckTLS(_ context.Context, _ string) (bool, error) {
	return false, errors.New("fake tls failure")
}
