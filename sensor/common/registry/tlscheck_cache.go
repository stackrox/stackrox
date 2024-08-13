package registry

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/registry/metrics"
)

type tlsCheckResult uint8

const (
	tlsCheckResultUnknown tlsCheckResult = iota
	tlsCheckResultSecure
	tlsCheckResultInsecure
	tlsCheckResultError
)

var (
	tlsCheckTTL = env.RegistryTLSCheckTTL.DurationSetting()
)

type tlsCheckCacheImpl struct {
	// results holds results from TLS checks. This prevents repeatedly
	// performing checks on the same registry within the cache expiry
	// window. An expiring cache is used because the TLS state of a registry
	// may change.
	results      expiringcache.Cache[string, *cacheEntry]
	checkTLSFunc CheckTLS
}

type cacheEntry struct {
	once   sync.Once
	result tlsCheckResult
}

// checkTLS performs a TLS check on a registry or returns the result from a
// previous check. Returns true for skip if there was a previous error.
func (e *cacheEntry) checkTLS(ctx context.Context, registry string, checkTLSFunc CheckTLS) (secure bool, skip bool, err error) {
	e.once.Do(func() {
		start := time.Now()
		secure, err = checkTLSFunc(ctx, registry)
		metrics.ObserveTLSCheckDuration(time.Since(start))

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

func newTLSCheckCache(checkTLSFunc CheckTLS) *tlsCheckCacheImpl {
	return &tlsCheckCacheImpl{
		results:      expiringcache.NewExpiringCache[string, *cacheEntry](tlsCheckTTL),
		checkTLSFunc: checkTLSFunc,
	}
}

func (c *tlsCheckCacheImpl) Cleanup() {
	c.results.RemoveAll()
}

// CheckTLS performs a TLS check on a registry or returns the result from a
// previous check. Returns true for skip if there was a previous error.
func (c *tlsCheckCacheImpl) CheckTLS(ctx context.Context, registry string) (secure bool, skip bool, err error) {
	// First check the cache for an entry, and, if found, perform
	// the TLS check. This is an optimization to avoid unnecessary
	// allocations on cache hits.
	entry, ok := c.results.Get(registry)
	if ok {
		metrics.IncrementTLSCheckCacheHitCount()
		return entry.checkTLS(ctx, registry, c.checkTLSFunc)
	}

	// Otherwise, create a new cache entry in a semi-coordinated way,
	// this may result in multiple cacheEntry objects being created
	// however only one will enter the cache.
	entry = c.results.GetOrSet(registry, &cacheEntry{})
	return entry.checkTLS(ctx, registry, c.checkTLSFunc)
}
