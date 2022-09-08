package common

import (
	"context"

	"github.com/gogo/protobuf/types"
)

// CVESuppressManager handles cve suppress and unsuppress workflow.
type CVESuppressManager interface {
	Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, ids ...string) error
	Unsuppress(ctx context.Context, ids ...string) error
}

// CVESuppressionCache holds suppressed vulnerabilities' information.
type CVESuppressionCache map[string]SuppressionCacheEntry

// SuppressionCacheEntry represents cache entry for suppressed resources.
type SuppressionCacheEntry struct {
	SuppressActivation *types.Timestamp
	SuppressExpiry     *types.Timestamp
}
