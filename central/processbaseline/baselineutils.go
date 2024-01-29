package processbaseline

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

// The EvaluationMode specifies when to treat a process baseline as being locked.
type EvaluationMode int

// This block enumerates all valid evaluation modes.
const (
	RoxLocked EvaluationMode = iota
	UserLocked
	RoxOrUserLocked
	RoxAndUserLocked

	ContainerStartupDuration = time.Minute
)

// locked checks whether a timestamp represents a locked process baseline true = locked, false = unlocked
func locked(lockTime *types.Timestamp) bool {
	return lockTime != nil && types.TimestampNow().Compare(lockTime) >= 0
}

// IsRoxLocked checks whether a process baseline is StackRox locked.
func IsRoxLocked(baseline *storage.ProcessBaseline) bool {
	return locked(baseline.GetStackRoxLockedTimestamp())
}

// IsUserLocked checks whether a process baseline is user locked.
func IsUserLocked(baseline *storage.ProcessBaseline) bool {
	return locked(baseline.GetUserLockedTimestamp())
}

// LockedUnderMode checks whether a process baseline is locked under the given evaluation mode.
func LockedUnderMode(baseline *storage.ProcessBaseline, mode EvaluationMode) bool {
	switch mode {
	case RoxLocked:
		return IsRoxLocked(baseline)
	case UserLocked:
		return IsUserLocked(baseline)
	case RoxOrUserLocked:
		return IsRoxLocked(baseline) || IsUserLocked(baseline)
	case RoxAndUserLocked:
		return IsRoxLocked(baseline) && IsUserLocked(baseline)
	}
	utils.Should(errors.Errorf("invalid evaluation mode: %v", mode))
	return false
}

// Processes returns the set of processes that are in the baseline.
// It returns nil if the process baseline is not locked under the passed EvaluationMode --
// if it returns nil, it means that all processes are in baseline under the given mode.
func Processes(baseline *storage.ProcessBaseline, mode EvaluationMode) *set.StringSet {
	if !LockedUnderMode(baseline, mode) {
		return nil
	}
	processes := set.NewStringSet()
	for _, element := range baseline.GetElements() {
		processes.Add(element.GetElement().GetProcessName())
	}
	return &processes
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
