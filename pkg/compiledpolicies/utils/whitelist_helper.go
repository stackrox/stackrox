package utils

import (
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv"
)

// WhitelistIsExpired returns true when the input whitelist is expired.
func WhitelistIsExpired(whitelist *v1.Whitelist) bool {
	if whitelist.GetExpiration() == nil {
		return false
	}
	now := time.Now()
	return protoconv.ConvertTimestampToTimeOrNow(whitelist.GetExpiration()).Before(now)
}
