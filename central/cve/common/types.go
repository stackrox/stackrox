package common

import "time"

// CVESuppressionCache holds suppressed vulnerabilities' information.
type CVESuppressionCache map[string]SuppressionCacheEntry

// SuppressionCacheEntry represents cache entry for suppressed resources.
type SuppressionCacheEntry struct {
	SuppressActivation *time.Time
	SuppressExpiry     *time.Time
}
