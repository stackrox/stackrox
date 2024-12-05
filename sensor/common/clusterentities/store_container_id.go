package clusterentities

import (
	"fmt"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/clusterentities/metrics"
	"golang.org/x/exp/maps"
)

type containerIDsStore struct {
	mutex sync.RWMutex
	// memorySize defines how many ticks old endpoint data should be remembered after removal request
	// Set to 0 to disable memory
	memorySize uint16

	// containerIDMap maps container IDs to container metadata
	containerIDMap map[string]ContainerMetadata
	// reverseContainerIDMap maps deploymentID to a set of container IDs associated with this deployment.
	reverseContainerIDMap map[string]set.StringSet

	// historicalContainerIDs is mimicking containerIDMap: container IDs -> container metadata -> historyStatus
	historicalContainerIDs map[string]map[ContainerMetadata]*entityStatus
}

func newContainerIDsStoreWithMemory(numTicks uint16) *containerIDsStore {
	store := &containerIDsStore{memorySize: numTicks}
	concurrency.WithLock(&store.mutex, func() {
		store.initMapsNoLock()
	})
	return store
}

func (e *containerIDsStore) initMapsNoLock() {
	e.containerIDMap = make(map[string]ContainerMetadata)
	e.reverseContainerIDMap = make(map[string]set.StringSet)
	e.historicalContainerIDs = make(map[string]map[ContainerMetadata]*entityStatus)
}

func (e *containerIDsStore) resetMaps() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	// Maps holding historical data must not be wiped on reset! Instead, all entities must be marked as historical.
	// Must be called before the respective source maps are wiped!
	// Performance optimization: no need to handle history if history is disabled
	if !e.historyEnabled() {
		e.initMapsNoLock()
		return
	}
	for s, metadata := range e.containerIDMap {
		e.addToHistory(s, metadata)
	}
	e.containerIDMap = make(map[string]ContainerMetadata)
	e.reverseContainerIDMap = make(map[string]set.StringSet)
	e.updateMetricsNoLock()
}

func (e *containerIDsStore) historyEnabled() bool {
	return e.memorySize > 0
}

// RecordTick records a tick
func (e *containerIDsStore) RecordTick() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	for id, metaMap := range e.historicalContainerIDs {
		for metadata, status := range metaMap {
			status.recordTick()
			// Remove all historical entries that expired in this tick.
			e.removeFromHistoryIfExpired(id, metadata)
		}
	}
}

func (e *containerIDsStore) Apply(updates map[string]*EntityData, incremental bool) []ContainerMetadata {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	var metadata []ContainerMetadata
	if !incremental {
		for deploymentID := range updates {
			e.purgeNoLock(deploymentID)
		}
	}
	for deploymentID, data := range updates {
		if data.isDeleteOnly() {
			// A call to Apply() with empty payload of the updates map (no values) is meant to be a delete operation.
			continue
		}
		r := e.applySingleNoLock(deploymentID, *data)
		metadata = append(metadata, r...)
	}
	e.updateMetricsNoLock()
	return metadata
}

func (e *containerIDsStore) purgeNoLock(deploymentID string) {
	for containerID := range e.reverseContainerIDMap[deploymentID] {
		if meta, found := e.containerIDMap[containerID]; found {
			if e.historyEnabled() {
				e.addToHistory(containerID, meta)
			} else {
				// Relevant when disabling history during runtime
				e.removeFromHistoryIfExpired(containerID, meta)
			}
		}
		delete(e.containerIDMap, containerID)
	}
	delete(e.reverseContainerIDMap, deploymentID)
}

func (e *containerIDsStore) applySingleNoLock(deploymentID string, data EntityData) []ContainerMetadata {
	cSet, found := e.reverseContainerIDMap[deploymentID]
	if cSet == nil || !found {
		cSet = set.NewStringSet()
	}

	mdsForCallback := make([]ContainerMetadata, 0, len(data.containerIDs))
	for containerID, metadata := range data.containerIDs {
		cSet.Add(containerID)
		e.containerIDMap[containerID] = metadata
		// We must unmark if the container was previously marked as historical, otherwise it will expire
		e.deleteFromHistory(containerID, metadata)
		mdsForCallback = append(mdsForCallback, metadata)
	}
	e.reverseContainerIDMap[deploymentID] = cSet
	return mdsForCallback
}

func (e *containerIDsStore) updateMetricsNoLock() {
	metrics.UpdateNumberOfContainerIDs(len(e.containerIDMap), len(e.historicalContainerIDs))
}

// lookupByContainerIDNoLock retrieves the deployment ID by a container ID from the non-historical data in the map.
func (e *containerIDsStore) lookupByContainer(containerID string) (data ContainerMetadata, found, isHistorical bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	if metadata, ok := e.containerIDMap[containerID]; ok {
		return metadata, true, false
	}
	if metaHistory, ok := e.historicalContainerIDs[containerID]; ok {
		// The metaHistory map contains 0 or 1 elements
		for metadata := range metaHistory {
			return metadata, true, true
		}
	}
	return ContainerMetadata{}, false, false
}

func (e *containerIDsStore) addToHistory(contID string, meta ContainerMetadata) {
	if _, ok := e.historicalContainerIDs[contID]; !ok {
		e.historicalContainerIDs[contID] = make(map[ContainerMetadata]*entityStatus)
	}
	e.historicalContainerIDs[contID][meta] = newHistoricalEntity(e.memorySize)
}

// deleteFromHistory deletes containerID from history
func (e *containerIDsStore) deleteFromHistory(contID string, meta ContainerMetadata) {
	delete(e.historicalContainerIDs[contID], meta)
	if len(e.historicalContainerIDs[contID]) == 0 {
		delete(e.historicalContainerIDs, contID)
	}
}

// removeFromHistoryIfExpired calls deleteFromHistory if the entry is expired
func (e *containerIDsStore) removeFromHistoryIfExpired(contID string, meta ContainerMetadata) {
	if status, ok := e.historicalContainerIDs[contID][meta]; ok && status.IsExpired() {
		e.deleteFromHistory(contID, meta)
	}
}

func (e *containerIDsStore) String() string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return fmt.Sprintf("Current: %s\n Historical: %s",
		maps.Keys(e.containerIDMap),
		prettyPrintHistoricalData(e.historicalContainerIDs))
}
