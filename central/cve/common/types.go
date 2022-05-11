package common

import "github.com/gogo/protobuf/types"

// CVESuppressionCache holds suppressed vulnerabilities' information.
type CVESuppressionCache map[string]SuppressionCacheEntry

// SuppressionCacheEntry represents cache entry for suppressed resources.
type SuppressionCacheEntry struct {
	Suppressed         bool
	SuppressActivation *types.Timestamp
	SuppressExpiry     *types.Timestamp
}
