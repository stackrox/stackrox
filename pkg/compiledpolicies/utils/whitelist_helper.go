package utils

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
)

// WhitelistIsExpired returns true when the input whitelist is expired.
func WhitelistIsExpired(whitelist *storage.Whitelist) bool {
	if whitelist.GetExpiration() == nil {
		return false
	}
	now := time.Now()
	return protoconv.ConvertTimestampToTimeOrNow(whitelist.GetExpiration()).Before(now)
}
