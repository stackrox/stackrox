package processbaseline

import (
	"github.com/stackrox/rox/central/processindicator/views"
	"github.com/stackrox/rox/generated/storage"
)

// BaselineItemFromProcess returns what we baseline for a given process.
// It exists to make sure that we're using the same thing in every place (name vs execfilepath).
func BaselineItemFromProcess(process *storage.ProcessIndicator) string {
	return process.GetSignal().GetExecFilePath()
}

// BaselineItemFromProcessView returns what we baseline for a given process.
// It exists to make sure that we're using the same thing in every place (name vs execfilepath).
func BaselineItemFromProcessView(process *views.ProcessIndicatorRiskView) string {
	return process.ExecFilePath
}
