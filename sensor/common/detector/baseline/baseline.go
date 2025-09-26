package baseline

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/processbaseline"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
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

// NewBaselineEvaluator creates a new baseline evaluator, using optimized implementation if feature flag is enabled
func NewBaselineEvaluator() Evaluator {
	if features.OptimizedBaselineMemory.Enabled() {
		return newOptimizedBaselineEvaluator()
	}
	return newBaselineEvaluator()
}

// newBaselineEvaluator creates the original baseline evaluator implementation
func newBaselineEvaluator() Evaluator {
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
		log.Debugf("Deleted process baseline %s", baseline.GetId())
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

	log.Debugf("Successfully added process baseline %s", baseline.GetId())
}

// IsInBaseline checks if the process indicator is within a locked baseline
// If the baseline does not exist, then we return true
func (w *baselineEvaluator) IsOutsideLockedBaseline(pi *storage.ProcessIndicator) bool {
	if pi == nil {
		return false // Treat nil process as within baseline
	}

	w.baselineLock.RLock()
	defer w.baselineLock.RUnlock()

	baseline := w.baselines[pi.GetDeploymentId()][pi.GetContainerName()]
	// If there is no baseline, then we are counting it as if it's within the baseline
	return baseline != nil && !baseline.Contains(processbaseline.BaselineItemFromProcess(pi))
}

// optimizedBaselineEvaluator implements memory-optimized baseline evaluation using process set deduplication
type optimizedBaselineEvaluator struct {
	// deployment -> container name -> content hash (direct access)
	deploymentBaselines map[string]map[string]string
	// content hash -> reference count and StringSet for deduplication
	processSets map[string]*processSetEntry
	// lock for thread safety
	lock sync.RWMutex
}

type processSetEntry struct {
	refCount  int
	processes set.StringSet
}

// newOptimizedBaselineEvaluator creates the optimized baseline evaluator implementation
func newOptimizedBaselineEvaluator() Evaluator {
	return &optimizedBaselineEvaluator{
		deploymentBaselines: make(map[string]map[string]string),
		processSets:         make(map[string]*processSetEntry),
	}
}

// removeReference decrements reference count and cleans up if necessary
func (oe *optimizedBaselineEvaluator) removeReference(contentHash string) {
	entry, exists := oe.processSets[contentHash]
	if !exists {
		return // Entry doesn't exist or is nil
	}

	if entry != nil {
		// Decrement reference count
		entry.refCount--
	}

	// Clean up if nil or no longer referenced
	if entry == nil || entry.refCount <= 0 {
		delete(oe.processSets, contentHash)
	}
}

// computeProcessSetHash creates a deterministic hash for a process set
func computeProcessSetHash(processes set.StringSet) string {
	// Convert to sorted slice for deterministic hashing
	processSlice := processes.AsSlice()
	slices.Sort(processSlice)

	// Create hash of concatenated processes
	content := strings.Join(processSlice, "\n")
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// RemoveDeployment removes a deployment from the optimized baseline evaluator
func (oe *optimizedBaselineEvaluator) RemoveDeployment(id string) {
	oe.lock.Lock()
	defer oe.lock.Unlock()

	containerMap := oe.deploymentBaselines[id]
	if containerMap == nil {
		return
	}

	// Decrement reference counts for all process sets used by this deployment
	for _, contentHash := range containerMap {
		oe.removeReference(contentHash)
	}

	delete(oe.deploymentBaselines, id)
	log.Debugf("Successfully removed deployment %s from baseline", id)
}

// AddBaseline adds a process baseline to the optimized evaluator
func (oe *optimizedBaselineEvaluator) AddBaseline(baseline *storage.ProcessBaseline) {
	oe.lock.Lock()
	defer oe.lock.Unlock()

	deploymentID := baseline.GetKey().GetDeploymentId()
	containerName := baseline.GetKey().GetContainerName()

	// Check if baseline should be unlocked (has UserLockedTimestamp = nil)
	if baseline.GetUserLockedTimestamp() == nil {
		oe.removeBaseline(baseline)
		return
	}

	// Locked baseline - process normally
	baselineSet := set.NewStringSet()
	for _, elem := range baseline.GetElements() {
		if process := elem.GetElement().GetProcessName(); process != "" {
			baselineSet.Add(process)
		}
	}

	// Find existing process set with same content or create new one
	contentHash := oe.findOrCreateProcessSet(baselineSet)

	// Update deployment mapping
	if oe.deploymentBaselines[deploymentID] == nil {
		oe.deploymentBaselines[deploymentID] = make(map[string]string)
	}

	// If this deployment/container already has a process set, decrement its ref count
	if oldContentHash, exists := oe.deploymentBaselines[deploymentID][containerName]; exists {
		oe.removeReference(oldContentHash)
	}

	oe.deploymentBaselines[deploymentID][containerName] = contentHash
	log.Debugf("Successfully added locked process baseline %s", baseline.GetId())
}

func (oe *optimizedBaselineEvaluator) removeBaseline(baseline *storage.ProcessBaseline) {
	log.Debugf("Removing (id:%s, UserLockedTimestamp:%v, elements:%v)", baseline.GetId(), baseline.GetUserLockedTimestamp(), baseline.GetElements())
	deploymentID := baseline.GetKey().GetDeploymentId()
	containerName := baseline.GetKey().GetContainerName()
	if oe.deploymentBaselines[deploymentID] != nil {
		if oldContentHash, exists := oe.deploymentBaselines[deploymentID][containerName]; exists {
			oe.removeReference(oldContentHash)
			delete(oe.deploymentBaselines[deploymentID], containerName)
			if len(oe.deploymentBaselines[deploymentID]) == 0 {
				delete(oe.deploymentBaselines, deploymentID)
			}
		} else {
			log.Debugf("Baseline for container name %s does not exist", containerName)
		}
	} else {
		log.Debugf("Baseline for deployment ID %s does not exist", deploymentID)
	}
}

// findOrCreateProcessSet finds an existing process set with the same content or creates a new one
func (oe *optimizedBaselineEvaluator) findOrCreateProcessSet(processes set.StringSet) string {
	contentHash := computeProcessSetHash(processes)

	// Check if we already have this process set
	if entry, exists := oe.processSets[contentHash]; exists {
		// Check for hash collision and verify content actually matches
		if !entry.processes.Equal(processes) {
			log.Panic("SHA256 hash collision detected for process set %v vs existing %v",
				processes.AsSlice(), entry.processes.AsSlice())
		}
		entry.refCount++
		return contentHash
	}

	// Create new process set
	oe.processSets[contentHash] = &processSetEntry{
		refCount:  1,
		processes: processes,
	}
	return contentHash
}

// IsOutsideLockedBaseline checks if the process indicator is within a locked baseline using optimized lookup
func (oe *optimizedBaselineEvaluator) IsOutsideLockedBaseline(pi *storage.ProcessIndicator) bool {
	if pi == nil {
		return false // Treat nil process as within baseline
	}

	oe.lock.RLock()
	defer oe.lock.RUnlock()

	containerMap := oe.deploymentBaselines[pi.GetDeploymentId()]
	if containerMap == nil {
		return false // No baseline exists, consider within baseline
	}

	contentHash, exists := containerMap[pi.GetContainerName()]
	if !exists {
		return false // No baseline exists, consider within baseline
	}

	entry := oe.processSets[contentHash]
	if entry == nil {
		return false // No baseline exists, consider within baseline
	}

	// Check if process is in the baseline
	return !entry.processes.Contains(processbaseline.BaselineItemFromProcess(pi))
}
