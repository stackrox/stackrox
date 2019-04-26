package processwhitelist

import (
	"github.com/gogo/protobuf/types"
)

// IsLocked checks whether a timestamp represents a locked whitelist true = locked, false = unlocked
func IsLocked(lockTime *types.Timestamp) bool {
	return lockTime != nil && types.TimestampNow().Compare(lockTime) >= 0
}
