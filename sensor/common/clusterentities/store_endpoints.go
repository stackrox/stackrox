package clusterentities

import (
	"fmt"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/clusterentities/metrics"
)

type endpointsStore struct {
	mutex sync.RWMutex
	// endpointMap maps endpoints to a (deployment id -> endpoint target info) mapping.
	endpointMap map[net.NumericEndpoint]map[string]set.Set[EndpointTargetInfo]
	// reverseEndpointMap maps deployment ids to sets of endpoints associated with this deployment.
	reverseEndpointMap map[string]set.Set[net.NumericEndpoint]

	// memorySize defines how many ticks old endpoint data should be remembered after removal request
	// Set to 0 to disable memory
	memorySize uint16
	// historicalEndpoints is mimicking endpointMap: endpoints -> deployment id -> endpoint target info -> historyStatus
	historicalEndpoints map[net.NumericEndpoint]map[string]map[EndpointTargetInfo]*entityStatus
	// reverseHistoricalEndpoints is mimicking reverseEndpointMap: deploymentID -> endpointInfo -> historyStatus
	reverseHistoricalEndpoints map[string]map[net.NumericEndpoint]*entityStatus
}

func newEndpointsStoreWithMemory(numTicks uint16) *endpointsStore {
	store := &endpointsStore{memorySize: numTicks}
	store.mutex.Lock()
	defer unlockWithMetric(&store.mutex, time.Now(), "endpoints", "init")
	store.initMapsNoLock()
	return store
}

func (e *endpointsStore) initMapsNoLock() {
	e.endpointMap = make(map[net.NumericEndpoint]map[string]set.Set[EndpointTargetInfo])
	e.reverseEndpointMap = make(map[string]set.Set[net.NumericEndpoint])

	e.reverseHistoricalEndpoints = make(map[string]map[net.NumericEndpoint]*entityStatus)
	e.historicalEndpoints = make(map[net.NumericEndpoint]map[string]map[EndpointTargetInfo]*entityStatus)
}

func (e *endpointsStore) resetMaps() {
	e.mutex.Lock()
	defer unlockWithMetric(&e.mutex, time.Now(), "endpoints", "reset_maps")
	// Maps holding historical data must not be wiped on reset! Instead, all entities must be marked as historical.
	// Must be called before the respective source maps are wiped!
	// Performance optimization: no need to handle history if history is disabled
	if !e.historyEnabled() {
		e.initMapsNoLock()
		return
	}
	// Add all endpoints to history before wiping the current state.
	for ep, m1 := range e.endpointMap {
		for deplID := range m1 {
			e.addToHistory(deplID, ep)
		}
	}

	e.endpointMap = make(map[net.NumericEndpoint]map[string]set.Set[EndpointTargetInfo])
	e.reverseEndpointMap = make(map[string]set.Set[net.NumericEndpoint])
	e.updateMetricsNoLock()
}

func (e *endpointsStore) historyEnabled() bool {
	return e.memorySize > 0
}

func (e *endpointsStore) updateMetricsNoLock() {
	metrics.UpdateNumberOfEndpoints(len(e.endpointMap), len(e.historicalEndpoints))
}

// RecordTick records a tick and returns true if
// there was any endpoint in the history expired in this tick with public IP address.
func (e *endpointsStore) RecordTick() bool {
	e.mutex.Lock()
	defer unlockWithMetric(&e.mutex, time.Now(), "endpoints", "record_tick")
	defer e.updateMetricsNoLock()
	removedPublic := false
	for endpoint, m1 := range e.historicalEndpoints {
		for deploymentID, m2 := range m1 {
			for _, status := range m2 {
				status.recordTick()
			}
			e.reverseHistoricalEndpoints[deploymentID][endpoint].recordTick()
			// Remove all historical entries that expired in this tick.
			removed := e.removeFromHistoryIfExpired(deploymentID, endpoint)
			removedPublic = removedPublic || removed && endpoint.IPAndPort.Address.IsPublic()
		}
	}
	return removedPublic
}

func (e *endpointsStore) Apply(updates map[string]*EntityData, incremental bool) {
	e.mutex.Lock()
	defer unlockWithMetric(&e.mutex, time.Now(), "endpoints", "apply")
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
	for ep := range endpointsSet {
		e.moveToHistory(deploymentID, ep)
	}
}

func (e *endpointsStore) applySingleNoLock(deploymentID string, data EntityData) {
	dSet, deploymentFound := e.reverseEndpointMap[deploymentID]
	if !deploymentFound || dSet == nil {
		dSet = set.NewSet[net.NumericEndpoint]()
	}

	for ep, targetInfos := range data.endpoints {
		dSet.Add(ep)
		e.reverseEndpointMap[deploymentID] = dSet

		deploymentsOnThisEp, epFound := e.endpointMap[ep]
		if !epFound {
			e.endpointMap[ep] = make(map[string]set.Set[EndpointTargetInfo])
		} else if !deploymentFound {
			// New deployment, but the endpoint exists - the new deployment takes over the already existing endpoint
			e.endpointMap[ep][deploymentID] = set.NewSet[EndpointTargetInfo]()
			// Mark all other deployments having with this endpoint as historical
			for otherDeploymentID := range deploymentsOnThisEp {
				// Currently added deployment is already in the map, so do not mark it historical
				if otherDeploymentID != deploymentID {
					e.moveToHistory(otherDeploymentID, ep)
				}
			}
		}
		etiSet, targetFound := e.endpointMap[ep][deploymentID]
		if !targetFound {
			etiSet = set.NewSet[EndpointTargetInfo]()
		}
		for _, tgtInfo := range targetInfos {
			etiSet.Add(tgtInfo)
		}
		e.endpointMap[ep][deploymentID] = etiSet
		// Endpoints previously marked as historical may need to be restored.
		e.deleteFromHistory(deploymentID, ep)
	}
}

type netAddrLookupper interface {
	LookupByNetAddr(ip net.IPAddress, port uint16) (results, historical []LookupResult)
}

func (e *endpointsStore) lookupEndpoint(endpoint net.NumericEndpoint, netLookup netAddrLookupper) (current, historical, ipLookup, ipLookupHistorical []LookupResult) {
	e.mutex.RLock()
	defer runlockWithMetric(&e.mutex, time.Now(), "endpoints", "lookup_endpoint")
	// Phase 1: Search in the current map
	current = doLookupEndpoint(endpoint, e.endpointMap)
	// Phase 2: Search in the historical map
	historical = doLookupEndpoint(endpoint, e.historicalEndpoints)
	if len(current)+len(historical) > 0 {
		return current, historical, ipLookup, ipLookupHistorical
	}
	// Phase 3: Search by network address
	ipLookup, ipLookupHistorical = netLookup.LookupByNetAddr(endpoint.IPAndPort.Address, endpoint.IPAndPort.Port)
	return current, historical, ipLookup, ipLookupHistorical
}

type Map[T any] interface {
	~map[EndpointTargetInfo]T
}

func doLookupEndpoint[M Map[T], T any](ep net.NumericEndpoint, src map[net.NumericEndpoint]map[string]M) (results []LookupResult) {
	for deploymentID, targetInfoSet := range src[ep] {
		result := LookupResult{
			Entity:         networkgraph.EntityForDeployment(deploymentID),
			ContainerPorts: make([]uint16, 0),
		}
		for tgtInfo := range targetInfoSet {
			result.ContainerPorts = append(result.ContainerPorts, tgtInfo.ContainerPort)
			if tgtInfo.PortName != "" {
				result.PortNames = append(result.PortNames, tgtInfo.PortName)
			}
		}
		results = append(results, result)
	}
	return results
}

// removeFromHistoryIfExpired iterates over all historical entries and deletes all that are expired
func (e *endpointsStore) removeFromHistoryIfExpired(deploymentID string, ep net.NumericEndpoint) bool {
	// Assumption: If an entry in reverseHistoricalMap is expired,
	// then the respective entry in historicalEndpoints should also be expired
	if status, ok := e.reverseHistoricalEndpoints[deploymentID][ep]; ok && status.IsExpired() {
		return e.deleteFromHistory(deploymentID, ep)
	}
	return false
}

// moveToHistory is a convenience function that removes data from the current map and adds it to history.
// If history is disabled, it just deletes the data from the current map.
func (e *endpointsStore) moveToHistory(deploymentID string, ep net.NumericEndpoint) {
	if e.historyEnabled() {
		e.addToHistory(deploymentID, ep)
	}
	e.deleteFromCurrent(deploymentID, ep)
}

// deleteFromHistory marks previously marked historical endpoint as no longer historical
func (e *endpointsStore) deleteFromHistory(deploymentID string, ep net.NumericEndpoint) bool {
	_, foundDepl := e.reverseHistoricalEndpoints[deploymentID][ep]
	_, foundEp := e.historicalEndpoints[ep][deploymentID]

	delete(e.reverseHistoricalEndpoints[deploymentID], ep)
	if len(e.reverseHistoricalEndpoints[deploymentID]) == 0 {
		delete(e.reverseHistoricalEndpoints, deploymentID)
	}
	delete(e.historicalEndpoints[ep], deploymentID)
	if len(e.historicalEndpoints[ep]) == 0 {
		delete(e.historicalEndpoints, ep)
	}
	return foundDepl || foundEp
}

// deleteFromCurrent is a helper that removes data from the current map, but does not manipulate history
func (e *endpointsStore) deleteFromCurrent(deploymentID string, ep net.NumericEndpoint) {
	delete(e.endpointMap[ep], deploymentID)
	if len(e.endpointMap[ep]) == 0 {
		delete(e.endpointMap, ep)
	}

	dSet, found := e.reverseEndpointMap[deploymentID]
	if found {
		dSet.Remove(ep)
		if dSet.Cardinality() == 0 {
			delete(e.reverseEndpointMap, deploymentID)
		}
	}
}

// addToHistory records history for one <deployment, endpoint> pair in linear time relative
// to the endpoint's target-info cardinality.
//
// Complexity:
//   - O(T) for one call, where T = number of EndpointTargetInfo entries for this endpoint.
//   - During purge of a deployment with M endpoints, total work scales as O(sum(T_i)) plus O(M)
//     reverse-map updates, avoiding any M-by-M scan.
//
// This routine is performance-critical for large clusters with many nodes and NodePort/LoadBalancer
// service expansions, where endpoint cardinality can become very high. Its complexity directly
// impacts how long endpointsStore mutex is held during Apply(), and therefore affects Sensor
// throughput and event pipeline latency.
func (e *endpointsStore) addToHistory(deploymentID string, ep net.NumericEndpoint) {
	// Prepare maps if empty
	if _, ok := e.historicalEndpoints[ep]; !ok {
		e.historicalEndpoints[ep] = make(map[string]map[EndpointTargetInfo]*entityStatus)
	}
	if _, ok := e.historicalEndpoints[ep][deploymentID]; !ok {
		e.historicalEndpoints[ep][deploymentID] = make(map[EndpointTargetInfo]*entityStatus)
	}
	for info := range e.endpointMap[ep][deploymentID] {
		e.historicalEndpoints[ep][deploymentID][info] = newHistoricalEntity(e.memorySize)
	}

	if _, ok := e.reverseHistoricalEndpoints[deploymentID]; !ok {
		e.reverseHistoricalEndpoints[deploymentID] = make(map[net.NumericEndpoint]*entityStatus)
	}
	e.reverseHistoricalEndpoints[deploymentID][ep] = newHistoricalEntity(e.memorySize)
}

func (e *endpointsStore) String() string {
	e.mutex.RLock()
	defer runlockWithMetric(&e.mutex, time.Now(), "endpoints", "string")
	currentStr := "map is empty"
	if len(e.endpointMap) > 0 {
		fragments1 := make([]string, 0, len(e.endpointMap))
		for netAddr, m1 := range e.endpointMap {
			for deplID := range m1 {
				fragments1 = append(fragments1,
					fmt.Sprintf("[ID=%s, net=%s]", deplID, netAddr.String()))
			}
		}
		currentStr = strings.Join(fragments1, "\n")
	}

	historyStr := "history is empty"
	if len(e.historicalEndpoints) > 0 {
		fragments2 := make([]string, 0, len(e.historicalEndpoints))
		for netAddr, m1 := range e.historicalEndpoints {
			subtree := prettyPrintHistoricalData(m1)
			fragments2 = append(fragments2, fmt.Sprintf("Net=%s %s", netAddr.String(), subtree))
		}
		historyStr = strings.Join(fragments2, "\n")
	}
	return fmt.Sprintf("Current: %s\nHistorical: %s", currentStr, historyStr)
}
