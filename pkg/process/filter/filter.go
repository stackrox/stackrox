package filter

import (
	"fmt"
	"hash"
	"strings"
	"unsafe"

	"github.com/cespare/xxhash/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/containerid"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// BinaryHash represents a 64-bit hash for memory-efficient key storage.
// Using uint64 directly avoids conversion overhead and provides faster map operations.
// This follows the pattern from network flow dedupers (PR #17040).
type BinaryHash uint64

/***
This filter is a rudimentary filter that prevents a container from spamming Central

Parameters:
### How `ROX_PROCESS_FILTER_FAN_OUT_LEVELS` works

For a given **(deployment, container, execFilePath)**, the filter builds an argument “tree”:

- **Level 0** = first argument token
- **Level 1** = second token
- **Level 2** = third token
- …
- Each `ROX_PROCESS_FILTER_FAN_OUT_LEVELS[i]` is the **max number of distinct children** allowed at that arg position under the same parent.
- If fan-out is exceeded at some level, **new variations are rejected** (filtered).
- If there are more arg tokens than levels, deeper tokens are **not distinguished** (they share the same leaf counter for max-exact matches).

The fan out value should decrease at each level.

Referred to as `fanOut` in the code.

### Configuring
Fan-out limits per argument level as comma-separated integers within brackets
Each value represents the maximum number of unique children at that level
Example: "[10,8,6,4]" increases first-level fan-out to 10
Empty value "" results in default value [8,6,4,2]
Empty array "[]" results in only tracking unique processes without arguments

---

### Example: `ROX_PROCESS_FILTER_FAN_OUT_LEVELS=[3,2]`

Meaning:

- **Level 0 fan-out = 3**: three distinct first arguments are allowed
- **Level 1 fan-out = 2**: two distinct seconds arguments per exec path and first argument.
- Third+ args aren’t used to create more levels (they share the same leaf).

Concrete sequence (same exec file path, same container):

1. /usr/bin/myexec arg1a arg2a -> accepted
2. /usr/bin/myexec arg1b arg2a -> accepted
3. /usr/bin/myexec arg1c arg2a -> accepted
4. /usr/bin/myexec arg1d arg2a -> rejected Only three unique first arguments are allowed
5. /usr/bin/myexec arg1a arg2b -> accepted
6. /usr/bin/myexec arg1a arg2c -> rejected Only two unique second arguments are allowed

### How `ROX_PROCESS_FILTER_MAX_EXACT_PATH_MATCHES` works
Maximum number of times an exact path (same deployment+container+process+args) can appear before being filtered.
Referred to as `maxExactPathMatches` in the code.

### How `ROX_PROCESS_FILTER_MAX_PROCESS_PATHS` works
Maximum number of unique process executable paths per container.
Referred to as `maxUniqueProcesses` in the code.

### Logic
	Keyed on deployment -> container ID, define a root level. Take the exec file path and retrieve or create
	the level for that specific process. No process exec file paths are limited because we want to see all new binaries.
	Then recursively sift through the args and create a level for each argument (up to len(fanOut)) that has a parent of the
 previous argument. If the fan out or the number of maxExactPathMatches has been exceeded, then return false. Otherwise, return true

***/

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

type child struct {
	hash BinaryHash
	node *level
}

type level struct {
	hits     int
	children []child
}

func newLevel() *level {
	return &level{}
}

func (l *level) findChild(h BinaryHash) *level {
	for _, c := range l.children {
		if c.hash == h {
			return c.node
		}
	}
	return nil
}

// containerNode is the per-container root of the filter tree, keyed by process
// executable path. The process layer is unbounded (Sensor sets maxUniqueProcesses
// to math.MaxInt), so a map keeps lookup and insertion O(1); argument levels below
// each process are bounded by fanOut and use slices (see level).
type containerNode = map[BinaryHash]*level

type filterImpl struct {
	maxExactPathMatches int     // maximum number of exact path (same pod + container and same process and args) matches to tolerate
	maxUniqueProcesses  int     // maximum number of unique process exec file paths
	maxFanOut           []uint8 // maximum fan out starting at the process level

	containersInDeployment map[string]map[string]containerNode
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

	nextLevel := level.findChild(argHash)
	if nextLevel == nil {
		// If this level has already hit its max fan out then return false
		if len(level.children) >= int(f.maxFanOut[levelNum]) {
			return false
		}
		nextLevel = newLevel()
		level.children = append(level.children, child{hash: argHash, node: nextLevel})
	}

	return f.siftNoLock(nextLevel, args[1:], levelNum+1)
}

// NewFilter returns an empty filter to start loading processes into
func NewFilter(maxExactPathMatches, maxUniqueProcesses int, fanOut []int) Filter {
	maxFanOut := make([]uint8, len(fanOut))
	for i, v := range fanOut {
		if v < 1 {
			// Guard the boundary: uint8(negative) wraps to a large value (e.g. -1 -> 255).
			// Mirror the env layer's MinimumElementValue(1) so invalid values cannot become max fan out.
			utils.Should(fmt.Errorf("fanOut[%d]=%d is below minimum, clamping to 1", i, v))
			v = 1
		}
		if v > 255 {
			utils.Should(fmt.Errorf("ROX_PROCESS_FILTER_FAN_OUT_LEVELS[%d]=%d exceeds uint8 max, clamping to 255", i, v))
			v = 255
		}
		maxFanOut[i] = uint8(v)
	}
	return &filterImpl{
		maxExactPathMatches: maxExactPathMatches,
		maxUniqueProcesses:  maxUniqueProcesses,
		maxFanOut:           maxFanOut,

		containersInDeployment: make(map[string]map[string]containerNode),
		h:                      xxhash.New(),
	}
}

func (f *filterImpl) getOrAddRootLevelNoLock(indicator *storage.ProcessIndicator) containerNode {
	containerMap := f.containersInDeployment[indicator.GetDeploymentId()]
	if containerMap == nil {
		containerMap = make(map[string]containerNode)
		f.containersInDeployment[indicator.GetDeploymentId()] = containerMap
	}

	rootLevel := containerMap[indicator.GetSignal().GetContainerId()]
	if rootLevel == nil {
		rootLevel = make(containerNode)
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
	processLevel := rootLevel[execFilePathHash]
	if processLevel == nil {
		if len(rootLevel) >= f.maxUniqueProcesses {
			return false
		}
		processLevel = newLevel()
		rootLevel[execFilePathHash] = processLevel
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
