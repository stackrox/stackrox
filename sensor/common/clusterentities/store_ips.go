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

type ipsStore struct {
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

func newIPsStoreWithMemory(numTicks uint16) *ipsStore {
	store := &ipsStore{memorySize: numTicks}
	concurrency.WithLock(&store.mutex, func() {
		store.initMapsNoLock()
	})
	return store
}

func (e *ipsStore) initMapsNoLock() {
	e.ipMap = make(map[net.IPAddress]set.StringSet)
	e.reverseIPMap = make(map[string]set.FrozenSet[net.IPAddress])
	e.historicalIPs = make(map[net.IPAddress]map[string]*entityStatus)
}

func (e *ipsStore) resetMaps() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	// Maps holding historical data must not be wiped on reset! Instead, all entities must be marked as historical.
	// Must be called before the respective source maps are wiped!
	// Performance optimization: no need to handle history if history is disabled
	if !e.historyEnabled() {
		e.initMapsNoLock()
		return
	}
	e.moveAllToHistory()

	e.ipMap = make(map[net.IPAddress]set.StringSet)
	e.reverseIPMap = make(map[string]set.FrozenSet[net.IPAddress])
	e.updateMetricsNoLock()
}

func (e *ipsStore) historyEnabled() bool {
	return e.memorySize > 0
}

func (e *ipsStore) updateMetricsNoLock() {
	metrics.UpdateNumberOfIPs(len(e.ipMap), len(e.historicalIPs))
}

func (e *ipsStore) RecordTick() set.FrozenSet[net.IPAddress] {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	dec := set.NewFrozenSet[net.IPAddress]()
	for ip, m := range e.historicalIPs {
		for deploymentID, status := range m {
			status.recordTick()
			// Remove all historical entries that expired in this tick.
			dec = dec.Union(e.removeFromHistoryIfExpired(deploymentID, ip))
		}
	}
	return dec
}

func (e *ipsStore) Apply(updates map[string]*EntityData, incremental bool) (dec, inc set.FrozenSet[net.IPAddress]) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.applyNoLock(updates, incremental)
}

func (e *ipsStore) applyNoLock(updates map[string]*EntityData, incremental bool) (dec, inc set.FrozenSet[net.IPAddress]) {
	defer e.updateMetricsNoLock()
	dec = set.NewFrozenSet[net.IPAddress]()
	inc = set.NewFrozenSet[net.IPAddress]()
	if !incremental {
		for deploymentID := range updates {
			dec = dec.Union(e.purgeNoLock(deploymentID))
		}
	}
	for deploymentID, data := range updates {
		if data == nil {
			continue
		}
		decA, incA := e.applySingleNoLock(deploymentID, *data)
		dec = dec.Union(decA)
		inc = inc.Union(incA)
	}
	// All IPs from `inc` will get +1, whereas all from `dec` will get -1. Let's optimize a bit
	common := inc.Intersect(dec)
	dec = dec.Difference(common)
	inc = inc.Difference(common)
	return dec, inc
}

func (e *ipsStore) purgeNoLock(deploymentID string) set.FrozenSet[net.IPAddress] {
	decPublicIPs := set.NewFrozenSet[net.IPAddress]()
	ipSet := e.reverseIPMap[deploymentID]
	for _, ip := range ipSet.AsSlice() {
		if e.historyEnabled() {
			e.moveToHistory(deploymentID, ip)
		} else {
			// If memory is disabled, we should not wait for a tick and delete all historical data immediately.
			// This should not be needed as the entries would never land in history in the first place,
			// but it may be useful if in the future we allow enabling/disabling history during runtime.
			decPublicIPs = decPublicIPs.Union(e.removeFromHistoryIfExpired(deploymentID, ip))
		}
	}
	return decPublicIPs
}

func (e *ipsStore) LookupByNetAddr(ip net.IPAddress, port uint16) (results, historical []LookupResult) {
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

func (e *ipsStore) applySingleNoLock(deploymentID string, data EntityData) (dec, inc set.FrozenSet[net.IPAddress]) {
	incPublicIPs := set.NewSet[net.IPAddress]()
	decPublicIPs := set.NewFrozenSet[net.IPAddress]()
	ipsSet := e.reverseIPMap[deploymentID].Unfreeze()
	for ip := range data.ips {
		ipsSet.Add(ip)

		deplSet := e.ipMap[ip]
		deplSet.Add(deploymentID)
		// This IP has more than one deployment! Interesting, let's record it.
		if deplSet.Cardinality() > 1 {
			metrics.ObserveManyDeploymentsSharingSingleIP(ip.AsNetIP().String(), deplSet.AsSlice())
		}
		e.ipMap[ip] = deplSet
		if ip.IsPublic() {
			incPublicIPs.Add(ip)
		}
		// If the IP being currently added was already in history,
		// we must remove it from the history to prevent expiration after a while.
		decPublicIPs = decPublicIPs.Union(e.deleteFromHistory(deploymentID, ip))
	}

	e.reverseIPMap[deploymentID] = ipsSet.Freeze()

	return decPublicIPs, incPublicIPs.Freeze()
}

// moveToHistory is a convenience function that removes data from the current map and adds it to history
func (e *ipsStore) moveToHistory(deploymentID string, ip net.IPAddress) {
	// Sometimes the call to existsInCurrent is not necessary, but I prefer to have it here, because it is cheap.
	if e.existsInCurrent(deploymentID, ip) {
		e.addToHistory(deploymentID, ip)
		e.deleteFromCurrent(deploymentID, ip)
	}
}

func (e *ipsStore) existsInCurrent(deploymentID string, ip net.IPAddress) bool {
	// If the map has no entry for this ip, then the set of deployment IDs will be empty and Contains returns false
	return e.ipMap[ip].Contains(deploymentID)
}

func (e *ipsStore) addToHistory(deploymentID string, ip net.IPAddress) {
	if _, ok := e.historicalIPs[ip]; !ok {
		e.historicalIPs[ip] = make(map[string]*entityStatus)
	}
	e.historicalIPs[ip][deploymentID] = newHistoricalEntity(e.memorySize)
}

func (e *ipsStore) deleteFromCurrent(deploymentID string, ip net.IPAddress) {
	deployments := e.ipMap[ip]
	deployments.Remove(deploymentID)
	if deployments.Cardinality() == 0 {
		// Usually one IP belongs to maximally one deployment, but let's be on the safe side.
		delete(e.ipMap, ip)
	} else {
		e.ipMap[ip] = deployments
	}

	ips := e.reverseIPMap[deploymentID]
	// To optimize memory allocations, we prevent unfreezing the set unless absolutely necessary
	switch ips.Cardinality() {
	case 0:
		// Deleting an IP from a deployment that has no IPs - this should occur extremely rarely
		delete(e.reverseIPMap, deploymentID)
		return
	case 1:
		if ips.Contains(ip) {
			// The set has one element that we want to remove - let's drop the entire set
			delete(e.reverseIPMap, deploymentID)
			return
		}
	default:
		// Interesting part! This deployment has more IPs, and we are removing only one of them
		us := ips.Unfreeze()
		us.Remove(ip)
		e.reverseIPMap[deploymentID] = us.Freeze()
	}
}

// deleteFromHistory removes all entries matching <deploymentID, IP> from history.
// It does not check whether the historical entry has expired.
func (e *ipsStore) deleteFromHistory(deploymentID string, ip net.IPAddress) set.FrozenSet[net.IPAddress] {
	if _, ok := e.historicalIPs[ip]; !ok {
		// Prevent adding the IP to the decrement list if there is nothing to remove
		return set.NewFrozenSet[net.IPAddress]()
	}
	// In most of the cases, "delete(e.historicalIPs, ip)"
	// should be enough as one IP should belong maximally to one deployment, but let's cover here the general case.
	delete(e.historicalIPs[ip], deploymentID)
	if len(e.historicalIPs[ip]) == 0 {
		delete(e.historicalIPs, ip)
		if ip.IsPublic() {
			return set.NewFrozenSet[net.IPAddress](ip)
		}
	}
	return set.NewFrozenSet[net.IPAddress]()
}

func (e *ipsStore) removeFromHistoryIfExpired(deploymentID string, ip net.IPAddress) set.FrozenSet[net.IPAddress] {
	if status, ok := e.historicalIPs[ip][deploymentID]; ok && status.IsExpired() {
		return e.deleteFromHistory(deploymentID, ip)
	}
	return set.NewFrozenSet[net.IPAddress]()
}

func (e *ipsStore) moveAllToHistory() {
	for ip, set1 := range e.ipMap {
		for deplID := range set1 {
			e.moveToHistory(deplID, ip)
		}
	}
}

func (e *ipsStore) String() string {
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
