package clusterentities

import (
	"fmt"
	"hash"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/clusterentities/metrics"
)

type endpointsStore struct {
	mutex sync.RWMutex
	// endpointMap maps endpoint hashes to a (deployment id -> container ports) mapping.
	endpointMap map[net.BinaryHash]map[string]set.Set[uint16]
	// reverseEndpointMap maps deployment ids to sets of endpoint hashes associated with this deployment.
	reverseEndpointMap map[string]set.Set[net.BinaryHash]

	// memorySize defines how many ticks old endpoint data should be remembered after removal request
	// Set to 0 to disable memory
	memorySize uint16
	// historicalEndpoints is mimicking endpointMap: endpoint hashes -> deployment id -> container port -> historyStatus
	historicalEndpoints map[net.BinaryHash]map[string]map[uint16]*entityStatus
	// reverseHistoricalEndpoints is mimicking reverseEndpointMap: deploymentID -> endpoint hash -> historyStatus
	reverseHistoricalEndpoints map[string]map[net.BinaryHash]*entityStatus

	// h is the xxhash instance used for hashing endpoints
	h hash.Hash64
	// hashBuf is a reusable buffer for hashing (16 bytes for IPv6 addresses)
	hashBuf [16]byte
}

func newEndpointsStoreWithMemory(numTicks uint16) *endpointsStore {
	store := &endpointsStore{
		memorySize: numTicks,
		h:          xxhash.New(),
	}
	concurrency.WithLock(&store.mutex, func() {
		store.initMapsNoLock()
	})
	return store
}

func (e *endpointsStore) initMapsNoLock() {
	e.endpointMap = make(map[net.BinaryHash]map[string]set.Set[uint16])
	e.reverseEndpointMap = make(map[string]set.Set[net.BinaryHash])

	e.reverseHistoricalEndpoints = make(map[string]map[net.BinaryHash]*entityStatus)
	e.historicalEndpoints = make(map[net.BinaryHash]map[string]map[uint16]*entityStatus)
}

func (e *endpointsStore) resetMaps() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	// Maps holding historical data must not be wiped on reset! Instead, all entities must be marked as historical.
	// Must be called before the respective source maps are wiped!
	// Performance optimization: no need to handle history if history is disabled
	if !e.historyEnabled() {
		e.initMapsNoLock()
		return
	}
	// Add all endpoints to history before wiping the current state.
	for epHash, m1 := range e.endpointMap {
		for deplID := range m1 {
			e.addToHistory(deplID, epHash)
		}
	}

	e.endpointMap = make(map[net.BinaryHash]map[string]set.Set[uint16])
	e.reverseEndpointMap = make(map[string]set.Set[net.BinaryHash])
	e.updateMetricsNoLock()
}

func (e *endpointsStore) historyEnabled() bool {
	return e.memorySize > 0
}

func (e *endpointsStore) updateMetricsNoLock() {
	metrics.UpdateNumberOfEndpoints(len(e.endpointMap), len(e.historicalEndpoints))
}

// RecordTick records a tick.
// Returns false since we cannot determine if expired endpoints had public IPs when using hashes.
// This is acceptable as public IP tracking is a best-effort optimization.
func (e *endpointsStore) RecordTick() bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	for epHash, m1 := range e.historicalEndpoints {
		for deploymentID, m2 := range m1 {
			for _, status := range m2 {
				status.recordTick()
			}

			e.reverseHistoricalEndpoints[deploymentID][epHash].recordTick()
			e.removeFromHistoryIfExpired(deploymentID, epHash)
		}
	}
	return false
}

func (e *endpointsStore) Apply(updates map[string]*EntityData, incremental bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.applyNoLock(updates, incremental)
}

func (e *endpointsStore) applyNoLock(updates map[string]*EntityData, incremental bool) {
	defer e.updateMetricsNoLock()
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
		e.applySingleNoLock(deploymentID, *data)
	}
}

func (e *endpointsStore) purgeNoLock(deploymentID string) {
	// We will be manipulating reverseEndpointMap when calling deleteFromCurrent or moveToHistory,
	// so let's make a temporary copy.
	endpointsSet := e.reverseEndpointMap[deploymentID]
	for epHash := range endpointsSet {
		if e.historyEnabled() {
			e.moveToHistory(deploymentID, epHash)
		} else {
			e.deleteFromCurrent(deploymentID, epHash)
		}
	}
}

func (e *endpointsStore) applySingleNoLock(deploymentID string, data EntityData) {
	dSet, deploymentFound := e.reverseEndpointMap[deploymentID]
	if !deploymentFound || dSet == nil {
		dSet = make(set.Set[net.BinaryHash], len(data.endpoints))
		e.reverseEndpointMap[deploymentID] = dSet
	}

	for ep, targetInfos := range data.endpoints {
		epHash := ep.BinaryKey(e.h, &e.hashBuf)
		dSet.Add(epHash)

		deploymentsOnThisEp, epFound := e.endpointMap[epHash]
		if !epFound {
			// New endpoint - create map with initial capacity
			e.endpointMap[epHash] = map[string]set.Set[uint16]{
				deploymentID: make(set.Set[uint16], len(targetInfos)),
			}
		} else if !deploymentFound {
			// New deployment takes over existing endpoint
			e.endpointMap[epHash][deploymentID] = make(set.Set[uint16], len(targetInfos))
			// Mark other deployments using this endpoint as historical
			for otherDeploymentID := range deploymentsOnThisEp {
				if otherDeploymentID != deploymentID {
					e.moveToHistory(otherDeploymentID, epHash)
				}
			}
		} else {
			// Ensure port set exists
			if _, targetFound := e.endpointMap[epHash][deploymentID]; !targetFound {
				e.endpointMap[epHash][deploymentID] = make(set.Set[uint16], len(targetInfos))
			}
		}

		// Add container ports to the set
		portSet := e.endpointMap[epHash][deploymentID]
		for _, tgtInfo := range targetInfos {
			portSet.Add(tgtInfo.ContainerPort)
		}

		e.deleteFromHistory(deploymentID, epHash)
	}
}

type netAddrLookupper interface {
	LookupByNetAddr(ip net.IPAddress, port uint16) (results, historical []LookupResult)
}

func (e *endpointsStore) lookupEndpoint(endpoint net.NumericEndpoint, netLookup netAddrLookupper) (current, historical, ipLookup, ipLookupHistorical []LookupResult) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	epHash := endpoint.BinaryKey(e.h, &e.hashBuf)

	// Phase 1: Search in the current map
	current = doLookupEndpoint(epHash, e.endpointMap)
	// Phase 2: Search in the historical map
	historical = doLookupEndpoint(epHash, e.historicalEndpoints)
	if len(current)+len(historical) > 0 {
		return current, historical, ipLookup, ipLookupHistorical
	}
	// Phase 3: Search by network address
	ipLookup, ipLookupHistorical = netLookup.LookupByNetAddr(endpoint.IPAndPort.Address, endpoint.IPAndPort.Port)
	return current, historical, ipLookup, ipLookupHistorical
}

type Map[T any] interface {
	~map[uint16]T
}

func doLookupEndpoint[M Map[T], T any](epHash net.BinaryHash, src map[net.BinaryHash]map[string]M) (results []LookupResult) {
	for deploymentID, portSet := range src[epHash] {
		result := LookupResult{
			Entity:         networkgraph.EntityForDeployment(deploymentID),
			ContainerPorts: make([]uint16, 0, len(portSet)),
		}
		for port := range portSet {
			result.ContainerPorts = append(result.ContainerPorts, port)
		}
		results = append(results, result)
	}
	return results
}

// removeFromHistoryIfExpired deletes historical entries that have expired.
func (e *endpointsStore) removeFromHistoryIfExpired(deploymentID string, epHash net.BinaryHash) bool {
	if status, ok := e.reverseHistoricalEndpoints[deploymentID][epHash]; ok && status.IsExpired() {
		return e.deleteFromHistory(deploymentID, epHash)
	}
	return false
}

// moveToHistory removes data from the current map and adds it to history.
func (e *endpointsStore) moveToHistory(deploymentID string, epHash net.BinaryHash) {
	e.addToHistory(deploymentID, epHash)
	e.deleteFromCurrent(deploymentID, epHash)
}

// deleteFromHistory removes an endpoint from the historical map.
func (e *endpointsStore) deleteFromHistory(deploymentID string, epHash net.BinaryHash) bool {
	_, foundDepl := e.reverseHistoricalEndpoints[deploymentID][epHash]
	_, foundEp := e.historicalEndpoints[epHash][deploymentID]

	delete(e.reverseHistoricalEndpoints[deploymentID], epHash)
	if len(e.reverseHistoricalEndpoints[deploymentID]) == 0 {
		delete(e.reverseHistoricalEndpoints, deploymentID)
	}
	delete(e.historicalEndpoints[epHash], deploymentID)
	if len(e.historicalEndpoints[epHash]) == 0 {
		delete(e.historicalEndpoints, epHash)
	}
	return foundDepl || foundEp
}

// deleteFromCurrent removes data from the current map without affecting history.
func (e *endpointsStore) deleteFromCurrent(deploymentID string, epHash net.BinaryHash) {
	delete(e.endpointMap[epHash], deploymentID)
	if len(e.endpointMap[epHash]) == 0 {
		delete(e.endpointMap, epHash)
	}

	dSet, found := e.reverseEndpointMap[deploymentID]
	if found {
		dSet.Remove(epHash)
		if dSet.Cardinality() == 0 {
			delete(e.reverseEndpointMap, deploymentID)
		}
	}
}

// addToHistory adds endpoint data to history without removing it from the current map.
func (e *endpointsStore) addToHistory(deploymentID string, epHash net.BinaryHash) {
	if _, ok := e.historicalEndpoints[epHash]; !ok {
		e.historicalEndpoints[epHash] = make(map[string]map[uint16]*entityStatus)
	}
	if _, ok := e.historicalEndpoints[epHash][deploymentID]; !ok {
		capacity := len(e.endpointMap[epHash][deploymentID])
		e.historicalEndpoints[epHash][deploymentID] = make(map[uint16]*entityStatus, capacity)
	}

	histMap := e.historicalEndpoints[epHash][deploymentID]
	for port := range e.endpointMap[epHash][deploymentID] {
		histMap[port] = newHistoricalEntity(e.memorySize)
	}

	if _, ok := e.reverseHistoricalEndpoints[deploymentID]; !ok {
		e.reverseHistoricalEndpoints[deploymentID] = make(map[net.BinaryHash]*entityStatus, len(e.reverseEndpointMap[deploymentID]))
	}

	revHistMap := e.reverseHistoricalEndpoints[deploymentID]
	for epHash := range e.reverseEndpointMap[deploymentID] {
		revHistMap[epHash] = newHistoricalEntity(e.memorySize)
	}
}

func (e *endpointsStore) String() string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	currentStr := "map is empty"
	if len(e.endpointMap) > 0 {
		fragments1 := make([]string, 0, len(e.endpointMap))
		for epHash, m1 := range e.endpointMap {
			for deplID := range m1 {
				fragments1 = append(fragments1,
					fmt.Sprintf("[ID=%s, hash=0x%x]", deplID, epHash))
			}
		}
		currentStr = strings.Join(fragments1, "\n")
	}

	historyStr := "history is empty"
	if len(e.historicalEndpoints) > 0 {
		fragments2 := make([]string, 0, len(e.historicalEndpoints))
		for epHash, m1 := range e.historicalEndpoints {
			subtree := prettyPrintHistoricalData(m1)
			fragments2 = append(fragments2, fmt.Sprintf("Hash=0x%x %s", epHash, subtree))
		}
		historyStr = strings.Join(fragments2, "\n")
	}
	return fmt.Sprintf("Current: %s\nHistorical: %s", currentStr, historyStr)
}
