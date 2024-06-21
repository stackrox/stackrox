package registry

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/utils"
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
	// performing checks on the same registry. For example, on Sensor startup
	// if a cluster has 10 pull secrets referencing quay.io, that would normally
	// result in ten connections to quay.io to test TLS, by caching the results
	// the number of connections can be reduced to one.
	//
	// An expiring cache is used because the TLS state of a registry may change.
	results expiringcache.Cache

	checkTLSFunc CheckTLS
}

func NewTLSCheckCache(checkTLSFunc CheckTLS) *tlsCheckCacheImpl {
	return &tlsCheckCacheImpl{
		results:      expiringcache.NewExpiringCache(tlsCheckTTL),
		checkTLSFunc: checkTLSFunc,
	}
}

func (c *tlsCheckCacheImpl) Cleanup() {
	c.results.RemoveAll()
}

// checkTLS performs a TLS check on a registry or returns the result from a
// previous check. Returns true for skip if there was a previous error and the
// registry should not be upserted into the store.
func (c *tlsCheckCacheImpl) CheckTLS(ctx context.Context, registry string) (secure bool, skip bool, err error) {
	result := c.getCachedTLSCheckResult(registry)
	switch result {
	case tlsCheckResultUnknown:
		// Do nothing (will proceed to after switch block).
	case tlsCheckResultSecure:
		return true, false, nil
	case tlsCheckResultInsecure:
		return false, false, nil
	case tlsCheckResultError:
		return false, true, nil
	default:
		utils.Should(errors.Errorf("Unsupported TLS check result: %v", result))
	}

	secure, err = c.doAndCacheTLSCheck(ctx, registry)
	return secure, false, err
}

func (rs *Store) getCachedTLSCheckResult(registry string) tlsCheckResult {
	resultI, ok := rs.tlsCheckResults.Get(registry)
	if !ok {
		return tlsCheckResultUnknown
	}

	return resultI
}

func (c *tlsCheckCacheImpl) doAndCacheTLSCheck(ctx context.Context, registry string) (bool, error) {
	secure, err := c.checkTLSFunc(ctx, registry)
	if err != nil {
		c.results.Add(registry, tlsCheckResultError)
		return false, errors.Wrapf(err, "unable to check TLS for registry %q", registry)
	}

	res := tlsCheckResultInsecure
	if secure {
		res = tlsCheckResultSecure
	}

	c.results.Add(registry, res)

	return secure, nil
}
