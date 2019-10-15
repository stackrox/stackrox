package processwhitelist

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

// The EvaluationMode specifies when to treat a whitelist as being locked.
type EvaluationMode int

// This block enumerates all valid evaluation modes.
const (
	RoxLocked EvaluationMode = iota
	UserLocked
	RoxOrUserLocked
	RoxAndUserLocked

	ContainerStartupDuration = time.Minute
)

var (
	log = logging.LoggerForModule()
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

// LockedUnderMode checks whether a whitelist is locked under the given evaluation mode.
func LockedUnderMode(whitelist *storage.ProcessWhitelist, mode EvaluationMode) bool {
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
	utils.Should(errors.Errorf("invalid evaluation mode: %v", mode))
	return false
}

// Processes returns the set of whitelisted processes from the whitelist.
// It returns nil if the whitelist is not locked under the passed EvaluationMode --
// if it returns nil, it means that all processes are whitelisted under the given mode.
func Processes(whitelist *storage.ProcessWhitelist, mode EvaluationMode) *set.StringSet {
	if !LockedUnderMode(whitelist, mode) {
		return nil
	}
	processes := set.NewStringSet()
	for _, element := range whitelist.GetElements() {
		processes.Add(element.GetElement().GetProcessName())
	}
	return &processes
}

// WhitelistItemFromProcess returns what we whitelist for a given process.
// It exists to make sure that we're using the same thing in every place (name vs execfilepath).
func WhitelistItemFromProcess(process *storage.ProcessIndicator) string {
	return process.GetSignal().GetExecFilePath()
}

// IsStartupProcess determines if the process is a startup process
// A process is considered a startup process if it happens within the first ContainerStartupDuration and was not scraped
// but instead pulled from exec
func IsStartupProcess(process *storage.ProcessIndicator) bool {
	if process.ContainerStartTime == nil {
		return false
	}
	durationBetweenProcessAndContainerStart := protoutils.Sub(process.GetSignal().GetTime(), process.GetContainerStartTime())
	return durationBetweenProcessAndContainerStart < ContainerStartupDuration
}
