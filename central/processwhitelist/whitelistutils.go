package processwhitelist

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/set"
)

// The EvaluationMode specifies when to treat a whitelist as being locked.
type EvaluationMode int

// This block enumerates all valid evaluation modes.
const (
	RoxLocked EvaluationMode = iota
	UserLocked
	RoxOrUserLocked
	RoxAndUserLocked
)

// locked checks whether a timestamp represents a locked whitelist true = locked, false = unlocked
func locked(lockTime *types.Timestamp) bool {
	return lockTime != nil && types.TimestampNow().Compare(lockTime) >= 0
}

// IsRoxLocked checks whether a whitelist is StackRox locked.
func IsRoxLocked(whitelist *storage.ProcessWhitelist) bool {
	return locked(whitelist.GetStackRoxLockedTimestamp())
}

// IsUserLocked checks whether a whitelist is user locked.
func IsUserLocked(whitelist *storage.ProcessWhitelist) bool {
	return locked(whitelist.GetUserLockedTimestamp())
}

func lockedUnderMode(whitelist *storage.ProcessWhitelist, mode EvaluationMode) bool {
	switch mode {
	case RoxLocked:
		return IsRoxLocked(whitelist)
	case UserLocked:
		return IsUserLocked(whitelist)
	case RoxOrUserLocked:
		return IsRoxLocked(whitelist) || IsUserLocked(whitelist)
	case RoxAndUserLocked:
		return IsRoxLocked(whitelist) && IsUserLocked(whitelist)
	}
	errorhelpers.PanicOnDevelopmentf("invalid evaluation mode: %v", mode)
	return false
}

// Processes returns the set of whitelisted processes from the whitelist.
// It returns nil if the whitelist is not locked under the passed EvaluationMode --
// if it returns nil, it means that all processes are whitelisted under the given mode.
func Processes(whitelist *storage.ProcessWhitelist, mode EvaluationMode) *set.StringSet {
	if !lockedUnderMode(whitelist, mode) {
		return nil
	}
	processes := set.NewStringSet()
	for _, element := range whitelist.GetElements() {
		processes.Add(element.GetElement().GetProcessName())
	}
	return &processes
}
