package clusterentities

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/clusterentities/metrics"
)

var log = logging.LoggerForModule()

type podIPsStore struct {
	mutex sync.RWMutex

	// memorySize defines how many ticks old data should be remembered after removal request
	// Set to 0 to disable memory
	memorySize uint16
	// ipMap maps ip addresses to sets of deployment ids this IP is associated with.
	ipMap map[net.IPAddress]set.StringSet
	// reverseIpMap maps deployment ids to sets of IP addresses associated with this deployment.
	reverseIPMap map[string]set.FrozenSet[net.IPAddress]
	// historicalIPs is mimicking ipMap: IP Address -> deploymentID -> historyStatus
	historicalIPs map[net.IPAddress]map[string]*entityStatus
}

func newPodIPsStoreWithMemory(numTicks uint16) *podIPsStore {
	store := &podIPsStore{memorySize: numTicks}
	concurrency.WithLock(&store.mutex, func() {
		store.initMapsNoLock()
	})
	return store
}

func (e *podIPsStore) initMapsNoLock() {
	e.ipMap = make(map[net.IPAddress]set.StringSet)
	e.reverseIPMap = make(map[string]set.FrozenSet[net.IPAddress])
	e.historicalIPs = make(map[net.IPAddress]map[string]*entityStatus)
}

func (e *podIPsStore) resetMaps() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	// Maps holding historical data must not be wiped on reset! Instead, all entities must be marked as historical.
	// Must be called before the respective source maps are wiped!
	// Performance optimization: no need to handle history if history is disabled
	if !e.historyEnabled() {
		e.initMapsNoLock()
		return
	}
	for deplID := range e.reverseIPMap {
		e.moveDeploymentToHistory(deplID)
	}

	e.ipMap = make(map[net.IPAddress]set.StringSet)
	e.reverseIPMap = make(map[string]set.FrozenSet[net.IPAddress])
	e.updateMetricsNoLock()
}

func (e *podIPsStore) historyEnabled() bool {
	return e.memorySize > 0
}

func (e *podIPsStore) updateMetricsNoLock() {
	metrics.UpdateNumberOfIPs(len(e.ipMap), len(e.historicalIPs))
}

// RecordTick records a tick and returns true if any Public IP in the history expired in this tick.
func (e *podIPsStore) RecordTick() bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	removedPublic := false
	for ip, m := range e.historicalIPs {
		for deploymentID, status := range m {
			status.recordTick()
			// Remove all historical entries that expired in this tick.
			removed := e.removeFromHistoryIfExpired(deploymentID, ip)
			removedPublic = removedPublic || removed && ip.IsPublic()
		}
	}
	return removedPublic
}

func (e *podIPsStore) Apply(updates map[string]*EntityData, incremental bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.applyNoLock(updates, incremental)
}

func (e *podIPsStore) applyNoLock(updates map[string]*EntityData, incremental bool) {
	defer e.updateMetricsNoLock()
	if !incremental {
		for deploymentID := range updates {
			e.purgeDeploymentNoLock(deploymentID)
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

func (e *podIPsStore) purgeDeploymentNoLock(deploymentID string) {
	if e.historyEnabled() {
		e.moveDeploymentToHistory(deploymentID)
	} else {
		e.deleteDeploymentFromCurrent(deploymentID)
		// In case we allow in the future to disable history during runtime, we would need to remove here all
		// expired data for deploymentID from the history.
	}
}

func (e *podIPsStore) LookupByNetAddr(ip net.IPAddress, port uint16) (results, historical []LookupResult) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	for deploymentID := range e.ipMap[ip] {
		result := LookupResult{
			Entity:         networkgraph.EntityForDeployment(deploymentID),
			ContainerPorts: []uint16{port},
		}
		results = append(results, result)
	}
	// if there is a match in the map, then there is no need to search in history,
	// as it may contain data about different past deployment using this address
	for histDeploymentID := range e.historicalIPs[ip] {
		result := LookupResult{
			Entity:         networkgraph.EntityForDeployment(histDeploymentID),
			ContainerPorts: []uint16{port},
		}
		historical = append(historical, result)
	}
	return results, historical
}

func (e *podIPsStore) applySingleNoLock(deploymentID string, data EntityData) {
	ipsSet := e.reverseIPMap[deploymentID].Unfreeze()
	for ip := range data.ips {
		ipsSet.Add(ip)

		// Check if this IP already belongs to other deployment.
		// If the `ip` is not in the `ipMap` then `e.ipMap[ip]` returns the zero-value of the StringSet,
		// which is an empty (but initialized) StringSet.
		deplSet := e.ipMap[ip]
		deplSet.Add(deploymentID)
		// This IP has more than one deployment! Interesting, let's record it.
		if deplSet.Cardinality() > 1 {
			metrics.ObserveManyDeploymentsSharingSingleIP(ip.AsNetIP().String(), deplSet.AsSlice())
		}
		e.ipMap[ip] = deplSet
		// If the IP being currently added was already in history,
		// we must remove it from there to prevent unwanted expiration.
		_ = e.deleteFromHistory(deploymentID, ip)
	}
	e.reverseIPMap[deploymentID] = ipsSet.Freeze()
}

// moveDeploymentToHistory is a convenience function that removes data from the current map and adds it to history
func (e *podIPsStore) moveDeploymentToHistory(deploymentID string) {
	e.addToHistory(deploymentID)
	e.deleteDeploymentFromCurrent(deploymentID)
}

func (e *podIPsStore) addToHistory(deploymentID string) {
	ipSet := e.reverseIPMap[deploymentID]
	for _, ip := range ipSet.AsSlice() {
		if _, ok := e.historicalIPs[ip]; !ok {
			e.historicalIPs[ip] = make(map[string]*entityStatus)
		}
		e.historicalIPs[ip][deploymentID] = newHistoricalEntity(e.memorySize)
	}
}

// deleteDeploymentFromCurrent deletes all data for given deployment from the current map
func (e *podIPsStore) deleteDeploymentFromCurrent(deploymentID string) {
	ips := e.reverseIPMap[deploymentID]
	for _, address := range ips.AsSlice() {
		deploymentsHavingIP := e.ipMap[address]
		if deploymentsHavingIP.Cardinality() < 2 {
			delete(e.ipMap, address)
		} else {
			log.Warnf("The same pod IP %s belongs to 2 or more deployments:%v !", address, deploymentsHavingIP.AsSlice())
		}
	}
	delete(e.reverseIPMap, deploymentID)
}

// deleteFromHistory removes all entries matching <deploymentID, IP> from history.
// It does not check whether the historical entry has expired.
func (e *podIPsStore) deleteFromHistory(deploymentID string, ip net.IPAddress) bool {
	if _, ok := e.historicalIPs[ip]; !ok {
		return false // nothing to remove
	}
	// In most of the cases, "delete(e.historicalIPs, ip)"
	// should be enough as one IP should belong maximally to one deployment, but let's cover here the general case.
	delete(e.historicalIPs[ip], deploymentID)
	if len(e.historicalIPs[ip]) == 0 {
		delete(e.historicalIPs, ip)
	}
	return true
}

func (e *podIPsStore) removeFromHistoryIfExpired(deploymentID string, ip net.IPAddress) bool {
	if status, ok := e.historicalIPs[ip][deploymentID]; ok && status.IsExpired() {
		return e.deleteFromHistory(deploymentID, ip)
	}
	return false
}

func (e *podIPsStore) String() string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	currStr := "current map is empty"
	if len(e.ipMap) > 0 {
		fragments := make([]string, 0, len(e.ipMap))
		for address, stringSet := range e.ipMap {
			fragments = append(fragments, fmt.Sprintf("{%q: %s}", address, stringSet.AsSlice()))
		}
		currStr = strings.Join(fragments, "\n")
	}
	histStr := "history is empty"
	if len(e.historicalIPs) > 0 {
		fragments := make([]string, 0, len(e.historicalIPs))
		for address, submap := range e.historicalIPs {
			subfragments := make([]string, 0, len(submap))
			for deplID, status := range submap {
				subfragments = append(subfragments, fmt.Sprintf("[ID=%s, ticksLeft=%d]", deplID, status.ticksLeft))
			}
			fragments = append(fragments, fmt.Sprintf("{%q: %s}", address, strings.Join(subfragments, ",")))

		}
		histStr = strings.Join(fragments, "\n")
	}
	return fmt.Sprintf("Current: %v\nHistorical: %s", currStr, histStr)
}
