package processbaseline

import "github.com/stackrox/stackrox/generated/storage"

// BaselineItemFromProcess returns what we baseline for a given process.
// It exists to make sure that we're using the same thing in every place (name vs execfilepath).
func BaselineItemFromProcess(process *storage.ProcessIndicator) string {
	return process.GetSignal().GetExecFilePath()
}
