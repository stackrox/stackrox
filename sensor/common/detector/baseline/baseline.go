package baseline

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/processbaseline"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stackrox/stackrox/pkg/sync"
)

// Evaluator encapsulates the interface to the baseline evaluator
type Evaluator interface {
	RemoveDeployment(id string)
	AddBaseline(baseline *storage.ProcessBaseline)
	IsOutsideLockedBaseline(pi *storage.ProcessIndicator) bool
}

type baselineEvaluator struct {
	// deployment -> container name -> exec file paths within baseline
	baselines    map[string]map[string]set.StringSet
	baselineLock sync.RWMutex
}

// NewBaselineEvaluator creates a new baseline evaluator
func NewBaselineEvaluator() Evaluator {
	return &baselineEvaluator{
		baselines: make(map[string]map[string]set.StringSet),
	}
}

// RemoveDeployment removes the baselines for this specific deployment
func (w *baselineEvaluator) RemoveDeployment(id string) {
	w.baselineLock.Lock()
	defer w.baselineLock.Unlock()

	delete(w.baselines, id)
}

// AddBaseline adds a baseline to the store
// If the baseline is unlocked, then we remove the baseline references because for the purposes
// of this package, an unlocked baseline has no impact. Locked baselines will have all of the processes
// added to a map
func (w *baselineEvaluator) AddBaseline(baseline *storage.ProcessBaseline) {
	// We'll get this msg with an unlocked baseline if a user unlocks a baseline
	// so we need to purge it from the baseline
	if baseline.GetUserLockedTimestamp() == nil {
		w.baselineLock.Lock()
		defer w.baselineLock.Unlock()

		delete(w.baselines[baseline.GetKey().GetDeploymentId()], baseline.GetKey().GetContainerName())
		return
	}

	// Create the baseline and overwrite the value in the map
	// We'll receive this message for all user locked baselines
	baselineSet := set.NewStringSet()
	for _, elem := range baseline.GetElements() {
		if process := elem.GetElement().GetProcessName(); process != "" {
			baselineSet.Add(process)
		}
	}

	w.baselineLock.Lock()
	defer w.baselineLock.Unlock()

	containerNameMap := w.baselines[baseline.GetKey().GetDeploymentId()]
	if containerNameMap == nil {
		containerNameMap = make(map[string]set.StringSet)
		w.baselines[baseline.GetKey().GetDeploymentId()] = containerNameMap
	}
	containerNameMap[baseline.GetKey().GetContainerName()] = baselineSet
}

// IsInBaseline checks if the process indicator is within a locked baseline
// If the baseline does not exist, then we return true
func (w *baselineEvaluator) IsOutsideLockedBaseline(pi *storage.ProcessIndicator) bool {
	w.baselineLock.RLock()
	defer w.baselineLock.RUnlock()

	baseline := w.baselines[pi.GetDeploymentId()][pi.GetContainerName()]
	// If there is no baseline, then we are counting it as if it's within the baseline
	return baseline != nil && !baseline.Contains(processbaseline.BaselineItemFromProcess(pi))
}
