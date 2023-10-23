package registry

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
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

// checkTLS performs a TLS check on a registry or returns the result from a
// previous check. Returns true for skip if there was a previous error and the
// registry should not be upserted into the store.
func (rs *Store) checkTLS(ctx context.Context, registry string) (secure bool, skip bool, err error) {
	result := rs.getCachedTLSCheckResult(registry)
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

	secure, err = rs.doAndCacheTLSCheck(ctx, registry)
	return secure, false, err
}

func (rs *Store) getCachedTLSCheckResult(registry string) tlsCheckResult {
	resultI := rs.tlsCheckResults.Get(registry)
	if resultI == nil {
		return tlsCheckResultUnknown
	}

	return resultI.(tlsCheckResult)
}

func (rs *Store) doAndCacheTLSCheck(ctx context.Context, registry string) (bool, error) {
	secure, err := rs.checkTLSFunc(ctx, registry)
	if err != nil {
		rs.tlsCheckResults.Add(registry, tlsCheckResultError)
		return false, errors.Wrapf(err, "unable to check TLS for registry %q", registry)
	}

	res := tlsCheckResultInsecure
	if secure {
		res = tlsCheckResultSecure
	}

	rs.tlsCheckResults.Add(registry, res)

	return secure, nil
}
