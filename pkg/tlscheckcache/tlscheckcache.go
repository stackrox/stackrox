package tlscheckcache

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/tlscheck"
	"github.com/stackrox/rox/pkg/utils"
)

type tlsCheckResult uint8

const (
	tlsCheckResultUnknown tlsCheckResult = iota
	tlsCheckResultSecure
	tlsCheckResultInsecure
	tlsCheckResultError

	defaultTTL = 15 * time.Minute
)

// CheckTLSFunc defines a function which checks if the given address is using TLS.
// An example implementation of this is tlscheck.CheckTLS.
type CheckTLSFunc func(ctx context.Context, endpoint string) (bool, error)

// Cache orchestrates and holds the results of TLS checks.
type Cache interface {
	// CheckTLS performs a TLS check on a endpoint or returns the result from a
	// previous check. Returns true for skip if there was a previous error.
	CheckTLS(ctx context.Context, endpoint string) (secure bool, skip bool, err error)

	// Cleanup will empty the cache.
	Cleanup()
}

type cacheImpl struct {
	checkTLSFunc CheckTLSFunc
	// metricSubsystem prometheus metrics will be labeled with this subsystem.
	metricSubsystem metrics.Subsystem
	// results holds results from TLS checks. This prevents repeatedly
	// performing checks on the same endpoint within the cache expiry
	// window. An expiring cache is used because the TLS state of a endpoint
	// may change.
	results expiringcache.Cache[string, *cacheEntry]
	// ttl represents the duration before a cached entry expires.
	ttl time.Duration
}

// New creates a cache that will hold results of recent TLS checks.
func New(opts ...CacheOption) Cache {
	cache := &cacheImpl{
		ttl:          defaultTTL,
		checkTLSFunc: tlscheck.CheckTLS,
	}

	for _, opt := range opts {
		opt(cache)
	}

	cache.results = expiringcache.NewExpiringCache[string, *cacheEntry](cache.ttl)

	return cache
}

func (c *cacheImpl) Cleanup() {
	c.results.RemoveAll()
}

func (c *cacheImpl) CheckTLS(ctx context.Context, endpoint string) (secure bool, skip bool, err error) {
	incrementTLSCheckCount(c.metricSubsystem)

	// First check the cache for an entry, and, if found, perform
	// the TLS check. This is an optimization to avoid unnecessary
	// allocations on cache hits.
	entry, ok := c.results.Get(endpoint)
	if ok {
		return entry.checkTLS(ctx, c.metricSubsystem, endpoint, c.checkTLSFunc)
	}

	// Otherwise, create a new cache entry in a semi-coordinated way,
	// this may result in multiple cacheEntry objects being created
	// however only one will enter the cache.
	entry = c.results.GetOrSet(endpoint, &cacheEntry{})
	return entry.checkTLS(ctx, c.metricSubsystem, endpoint, c.checkTLSFunc)
}

// CacheOption modifies the default values of the cache.
type CacheOption func(*cacheImpl)

// WithTTL sets the duration before cached entries expire.
func WithTTL(ttl time.Duration) CacheOption {
	return func(c *cacheImpl) {
		c.ttl = ttl
	}
}

// WithTLSCheckFunc sets function that will be used when performing TLS checks.
func WithTLSCheckFunc(f CheckTLSFunc) CacheOption {
	return func(c *cacheImpl) {
		c.checkTLSFunc = f
	}
}

// WithMetricSubsystem sets the subsystem label that will be applied to prometheus metrics
// tracked by the cache.
func WithMetricSubsystem(metricSubsystem metrics.Subsystem) CacheOption {
	return func(c *cacheImpl) {
		c.metricSubsystem = metricSubsystem
	}
}

type cacheEntry struct {
	once   sync.Once
	result tlsCheckResult
}

// checkTLS performs a TLS check on a endpoint or returns the result from a
// previous check. Returns true for skip if there was a previous error.
func (e *cacheEntry) checkTLS(ctx context.Context, metricSubsystem metrics.Subsystem, endpoint string, checkTLSFunc CheckTLSFunc) (secure bool, skip bool, err error) {
	e.once.Do(func() {
		start := time.Now()
		secure, err = checkTLSFunc(ctx, endpoint)
		observeTLSCheckDuration(metricSubsystem, time.Since(start))

		if err != nil {
			e.result = tlsCheckResultError
			return
		}

		e.result = tlsCheckResultInsecure
		if secure {
			e.result = tlsCheckResultSecure
		}
	})

	if err != nil {
		return false, false, err
	}

	switch e.result {
	case tlsCheckResultSecure:
		return true, false, nil
	case tlsCheckResultInsecure:
		return false, false, nil
	case tlsCheckResultError:
		return false, true, nil
	}

	// Should not be reachable.
	return false, false, utils.ShouldErr(errors.Errorf("Unknown TLS check result: %v", e.result))
}
