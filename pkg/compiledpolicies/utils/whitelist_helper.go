package utils

import (
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
)

// WhitelistIsExpired returns true when the input whitelist is expired.
func WhitelistIsExpired(whitelist *v1.Whitelist) bool {
	if whitelist.GetExpiration() == nil {
		return false
	}
	now := time.Now()
	return protoconv.ConvertTimestampToTimeOrNow(whitelist.GetExpiration()).Before(now)
}
