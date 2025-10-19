package baseline

import (
	"slices"

	//"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/processbaseline"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	//"github.com/stackrox/rox/pkg/utils"

	"github.com/cespare/xxhash"
)

// optimizedBaselineEvaluatorMapSize implements memory-optimized baseline evaluation using process set deduplication
type optimizedBaselineEvaluatorMapSize struct {
	// deployment -> container name -> content hash (direct access)
	deploymentBaselines map[string]map[string]XXHashKey
	// content hash -> reference count and StringSet for deduplication
	processSets map[XXHashKey]*processSetEntry
	// lock for thread safety
	lock sync.RWMutex
}

// newOptimizedBaselineEvaluator creates the optimized baseline evaluator implementation
func newOptimizedBaselineEvaluatorMapSize() Evaluator {
	return &optimizedBaselineEvaluatorMapSize{
		deploymentBaselines: make(map[string]map[string]XXHashKey, 50),
		processSets:         make(map[XXHashKey]*processSetEntry, 1000),
	}
}

// removeReference decrements reference count and cleans up if necessary
func (oe *optimizedBaselineEvaluatorMapSize) removeReference(contentHash XXHashKey) {
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

// RemoveDeployment removes a deployment from the optimized baseline evaluator
func (oe *optimizedBaselineEvaluatorMapSize) RemoveDeployment(id string) {
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

func (oe *optimizedBaselineEvaluatorMapSize) AddBaseline(baseline *storage.ProcessBaseline) {
	oe.lock.Lock()
	defer oe.lock.Unlock()

	deploymentID := baseline.GetKey().GetDeploymentId()
	containerName := baseline.GetKey().GetContainerName()

	if baseline.GetUserLockedTimestamp() == nil {
		oe.removeBaseline(baseline)
		return
	}

	// Build sorted slice directly from baseline elements
	processes := make([]string, 0, len(baseline.GetElements()))
	for _, elem := range baseline.GetElements() {
		if process := elem.GetElement().GetProcessName(); process != "" {
			processes = append(processes, process)
		}
	}
	slices.Sort(processes)

	// Compute hash and build deduplicated set in one pass
	contentHash, baselineSet := computeHashAndBuildSetMapSize(processes)

	if _, exists := oe.processSets[contentHash]; !exists {
		oe.processSets[contentHash] = &processSetEntry{
        	        refCount:  1,
        	        processes: baselineSet,
        	}
	}

	// ... rest of the function
	if oe.deploymentBaselines[deploymentID] == nil {
		oe.deploymentBaselines[deploymentID] = make(map[string]XXHashKey, 500)
	}
	if oldContentHash, exists := oe.deploymentBaselines[deploymentID][containerName]; exists {
		oe.removeReference(oldContentHash)
	}
	oe.deploymentBaselines[deploymentID][containerName] = contentHash
}

func computeHashAndBuildSetMapSize(sortedProcesses []string) (XXHashKey, set.StringSet) {
	h := xxhash.New()
	baselineSet := set.NewStringSet()

	var prev string
	for i, process := range sortedProcesses {
		// Skip duplicates (sorted, so they're adjacent)
		if i > 0 && process == prev {
			continue
		}
		prev = process

		h.Write([]byte(process))
		h.Write([]byte{'\n'})
		baselineSet.Add(process)
	}

	hashValue := h.Sum64()

	return XXHashKey(hashValue), baselineSet
}

func (oe *optimizedBaselineEvaluatorMapSize) removeBaseline(baseline *storage.ProcessBaseline) {
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

//// findOrCreateProcessSet finds an existing process set with the same content or creates a new one
//func (oe *optimizedBaselineEvaluatorMapSize) findOrCreateProcessSet(processes set.StringSet) XXHashKey {
//	contentHash, _ := computeHashAndBuildSetMapSize(processes)
//
//	// Check if we already have this process set
//	if entry, exists := oe.processSets[contentHash]; exists {
//		// Check for hash collision and verify content actually matches
//		if !entry.processes.Equal(processes) {
//			utils.Should(errors.Errorf("SHA256 hash collision detected for process set %v vs existing %v",
//				processes.AsSlice(), entry.processes.AsSlice()))
//		}
//		entry.refCount++
//		return contentHash
//	}
//
//	// Create new process set
//	oe.processSets[contentHash] = &processSetEntry{
//		refCount:  1,
//		processes: processes,
//	}
//	return contentHash
//}

// IsOutsideLockedBaseline checks if the process indicator is within a locked baseline using optimized lookup
func (oe *optimizedBaselineEvaluatorMapSize) IsOutsideLockedBaseline(pi *storage.ProcessIndicator) bool {
	if pi == nil {
		return false // Treat nil process as within baseline
	}

	oe.lock.RLock()
	defer oe.lock.RUnlock()

	containerMap, found := oe.deploymentBaselines[pi.GetDeploymentId()]
	if !found || containerMap == nil {
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
