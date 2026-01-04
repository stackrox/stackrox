package filter

import (
	"hash"
	"strings"
	"unsafe"

	"github.com/cespare/xxhash"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/containerid"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
)

// BinaryHash represents a 64-bit hash for memory-efficient key storage.
// Using uint64 directly avoids conversion overhead and provides faster map operations.
// This follows the pattern from network flow dedupers (PR #17040).
type BinaryHash uint64

// This filter is a rudimentary filter that prevents a container from spamming Central
//
// Parameters:
// maxExactPathMatch:
// 	The maximum number of times a complete path has been taken
// 	e.g. The exact path "bash -c nmap" will only be passed through at most maxExactPathMatch
//
// fanOut:
// 	The degree of fan out of each level, generally decreasing
//
// Logic:
// 	Keyed on deployment -> container ID, define a root level. Take the exec file path and retrieve or create
// 	the level for that specific process. No process exec file paths are limited because we want to see all new binaries.
// 	Then recursively sift through the args and create a level for each argument (up to len(fanOut)) that has a parent of the
//  previous argument. If the fan out or the number of maxExactPathMatches has been exceeded, then return false. Otherwise, return true
//
// Configuration:
// The filter can be configured via environment variables:
//   - ROX_PROCESS_FILTER_MAX_EXACT_PATH_MATCHES
//     Maximum number of times an exact path (same deployment+container+process+args) can appear before being filtered
//   - ROX_PROCESS_FILTER_FAN_OUT_LEVELS
//     Fan-out limits per argument level as comma-separated integers within brackets
//     Each value represents the maximum number of unique children at that level
//     Example: "[10,8,6,4]" increases first-level fan-out to 10
//     Empty value "" results in default value [8,6,4,2]
//     Empty array "[]" results in only tracking unique processes without arguments
//   - ROX_PROCESS_FILTER_MAX_PROCESS_PATHS
//     Maximum number of unique process executable paths per container

const (
	maxArgSize = 16
)

// Filter takes in a process indicator via add and determines if should be filtered or not
//
//go:generate mockgen-wrapper
type Filter interface {
	Add(indicator *storage.ProcessIndicator) bool
	UpdateByPod(pod *storage.Pod)
	UpdateByGivenContainers(deploymentID string, liveContainerSet set.StringSet)
	Delete(deploymentID string)
	DeleteByPod(pod *storage.Pod)
}

type level struct {
	hits     int
	children map[BinaryHash]*level
}

func newLevel() *level {
	return &level{
		children: make(map[BinaryHash]*level),
	}
}

type filterImpl struct {
	maxExactPathMatches int   // maximum number of exact path (same pod + container and same process and args) matches to tolerate
	maxUniqueProcesses  int   // maximum number of unique process exec file paths
	maxFanOut           []int // maximum fan out starting at the process level

	containersInDeployment map[string]map[string]*level
	rootLock               sync.Mutex

	// Hash instance for computing BinaryHash keys
	// Reused across Add() calls to avoid allocations
	h hash.Hash64
}

func (f *filterImpl) siftNoLock(level *level, args []string, levelNum int) bool {
	if len(args) == 0 || levelNum >= len(f.maxFanOut) {
		// If we have hit this point with this exact level structure, maxExactPathMatch number of times
		// then return false. Otherwise increment the levels hits and return true
		if level.hits >= f.maxExactPathMatches {
			return false
		}
		level.hits++
		return true
	}
	// Truncate the current argument to the max size to avoid large arguments taking up a lot of space

	truncated := stringutils.Truncate(args[0], maxArgSize)

	// Hash the truncated arguments to solve 2 problems:
	// 1. Holding references to the original string data received from the DB scan
	// 2. Using BinaryHash as map key is reducing memory requirements for the filter
	argHash := hashString(f.h, truncated)

	nextLevel := level.children[argHash]
	if nextLevel == nil {
		// If this level has already hit its max fan out then return false
		if len(level.children) >= f.maxFanOut[levelNum] {
			return false
		}
		nextLevel = newLevel()
		level.children[argHash] = nextLevel
	}

	return f.siftNoLock(nextLevel, args[1:], levelNum+1)
}

// NewFilter returns an empty filter to start loading processes into
func NewFilter(maxExactPathMatches, maxUniqueProcesses int, fanOut []int) Filter {
	return &filterImpl{
		maxExactPathMatches: maxExactPathMatches,
		maxUniqueProcesses:  maxUniqueProcesses,
		maxFanOut:           fanOut,

		containersInDeployment: make(map[string]map[string]*level),
		h:                      xxhash.New(),
	}
}

func (f *filterImpl) getOrAddRootLevelNoLock(indicator *storage.ProcessIndicator) *level {
	containerMap := f.containersInDeployment[indicator.GetDeploymentId()]
	if containerMap == nil {
		containerMap = make(map[string]*level)
		f.containersInDeployment[indicator.GetDeploymentId()] = containerMap
	}

	rootLevel := containerMap[indicator.GetSignal().GetContainerId()]
	if rootLevel == nil {
		rootLevel = newLevel()
		containerMap[indicator.GetSignal().GetContainerId()] = rootLevel
	}

	return rootLevel
}

func (f *filterImpl) Add(indicator *storage.ProcessIndicator) bool {
	f.rootLock.Lock()
	defer f.rootLock.Unlock()

	rootLevel := f.getOrAddRootLevelNoLock(indicator)

	execFilePath := indicator.GetSignal().GetExecFilePath()
	// Hash the execFilePath to solve 2 problems:
	// 1. Holding references to the original string data received from the DB scan
	// 2. Using BinaryHash as map key is reducing memory requirements for the filter
	execFilePathHash := hashString(f.h, execFilePath)

	// Handle the process level independently as we will never reject a new process
	processLevel := rootLevel.children[execFilePathHash]
	if processLevel == nil {
		if len(rootLevel.children) >= f.maxUniqueProcesses {
			return false
		}
		processLevel = newLevel()
		rootLevel.children[execFilePathHash] = processLevel
	}

	return f.siftNoLock(processLevel, strings.Fields(indicator.GetSignal().GetArgs()), 0)
}

func (f *filterImpl) UpdateByPod(pod *storage.Pod) {
	f.rootLock.Lock()
	defer f.rootLock.Unlock()

	liveContainerSet := set.NewStringSet()
	for _, instance := range pod.GetLiveInstances() {
		liveContainerSet.Add(containerid.ShortContainerIDFromInstance(instance))
	}

	f.updateByGivenContainersNoLock(pod.GetDeploymentId(), liveContainerSet)
}

func (f *filterImpl) updateByGivenContainersNoLock(deploymentID string, liveContainerSet set.StringSet) {
	containersMap := f.containersInDeployment[deploymentID]
	for k := range containersMap {
		if !liveContainerSet.Contains(k) {
			delete(containersMap, k)
		}
	}
}

func (f *filterImpl) UpdateByGivenContainers(deploymentID string, liveContainerSet set.StringSet) {
	f.rootLock.Lock()
	defer f.rootLock.Unlock()

	f.updateByGivenContainersNoLock(deploymentID, liveContainerSet)
}

func (f *filterImpl) Delete(deploymentID string) {
	f.rootLock.Lock()
	defer f.rootLock.Unlock()

	delete(f.containersInDeployment, deploymentID)
}

func (f *filterImpl) DeleteByPod(pod *storage.Pod) {
	f.rootLock.Lock()
	defer f.rootLock.Unlock()

	containerSet := set.NewStringSet()
	for _, instance := range pod.GetLiveInstances() {
		containerSet.Add(containerid.ShortContainerIDFromInstance(instance))
	}

	containersMap := f.containersInDeployment[pod.GetDeploymentId()]
	for k := range containersMap {
		if containerSet.Contains(k) {
			delete(containersMap, k)
		}
	}
}

// hashString creates a hash from a single string.
// Convenience wrapper for hashStrings with a single argument.
func hashString(h hash.Hash64, s string) BinaryHash {
	if len(s) == 0 {
		return BinaryHash(0)
	}

	h.Reset()
	// Use zero-copy conversion from string to []byte using unsafe to avoid allocation.
	// This is safe because:
	// 1. h.Write() doesn't modify data (io.Writer contract)
	// 2. xxhash doesn't retain references
	// 3. string s remains alive during the call
	//#nosec G103 -- Audited: zero-copy string-to-bytes conversion for performance
	_, _ = h.Write(unsafe.Slice(unsafe.StringData(s), len(s)))
	return BinaryHash(h.Sum64())
}
