package baseline

import (
	//"crypto/sha256"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/processbaseline"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"

	"github.com/cespare/xxhash"
)

// optimizedBaselineEvaluator implements memory-optimized baseline evaluation using process set deduplication
type optimizedBaselineEvaluator struct {
	// deployment -> container name -> content hash (direct access)
	deploymentBaselines map[string]map[string]XXHashKey
	// content hash -> reference count and StringSet for deduplication
	processSets map[XXHashKey]*processSetEntry
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
		deploymentBaselines: make(map[string]map[string]XXHashKey),
		processSets:         make(map[XXHashKey]*processSetEntry),
	}
}

// removeReference decrements reference count and cleans up if necessary
func (oe *optimizedBaselineEvaluator) removeReference(contentHash XXHashKey) {
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

type XXHashKey uint64

func computeProcessSetXXHash(processes set.StringSet) XXHashKey {
        processSlice := processes.AsSlice()
        slices.Sort(processSlice)
        content := strings.Join(processSlice, "\n")
        return XXHashKey(xxhash.Sum64([]byte(content)))
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
		oe.deploymentBaselines[deploymentID] = make(map[string]XXHashKey)
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
func (oe *optimizedBaselineEvaluator) findOrCreateProcessSet(processes set.StringSet) XXHashKey {
	contentHash := computeProcessSetXXHash(processes)

	// Check if we already have this process set
	if entry, exists := oe.processSets[contentHash]; exists {
		// Check for hash collision and verify content actually matches
		if !entry.processes.Equal(processes) {
			utils.Should(errors.Errorf("SHA256 hash collision detected for process set %v vs existing %v",
				processes.AsSlice(), entry.processes.AsSlice()))
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
