package clusterentities

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/clusterentities/metrics"
)

// ContainerMetadata is the container metadata stored per instance
type ContainerMetadata struct {
	DeploymentID  string
	DeploymentTS  int64
	PodID         string
	PodUID        string
	ContainerName string
	ContainerID   string
	Namespace     string
	StartTime     *time.Time
	ImageID       string
}

// PublicIPsListener is an interface for listeners on changes to the set of public IP addresses.
// Note: Implementors of this interface must ensure the methods complete in a very short time/do not block, as they
// get invoked synchronously in a critical section.
type PublicIPsListener interface {
	OnAdded(ip net.IPAddress)
	OnRemoved(ip net.IPAddress)
}

// Store is a store for managing cluster entities (currently deployments only) and allows looking them up by
// endpoint.
type Store struct {
	// ipMap maps ip addresses to sets of deployment ids this IP is associated with.
	ipMap map[net.IPAddress]map[string]struct{}
	// endpointMap maps endpoints to a (deployment id -> endpoint target info) mapping.
	endpointMap map[net.NumericEndpoint]map[string]map[EndpointTargetInfo]struct{}
	// containerIDMap maps container IDs to container metadata
	containerIDMap map[string]ContainerMetadata

	// reverseIpMap maps deployment ids to sets of IP addresses associated with this deployment.
	reverseIPMap map[string]map[net.IPAddress]struct{}
	// reverseEndpointMap maps deployment ids to sets of endpoints associated with this deployment.
	reverseEndpointMap map[string]map[net.NumericEndpoint]struct{}
	// reverseContainerIDMap maps deployment ids to sets of container IDs associated with this deployment.
	reverseContainerIDMap map[string]map[string]struct{}
	// callbackChannel is a channel to send container metadata upon resolution
	callbackChannel chan<- ContainerMetadata

	publicIPRefCounts  map[net.IPAddress]*int
	publicIPsListeners map[PublicIPsListener]struct{}

	mutex sync.RWMutex

	// entitiesMemorySize defines how many ticks old endpoint data should be remembered after removal request
	// Set to 0 to disable memory
	entitiesMemorySize uint16
	// historicalEndpoints is mimicking endpointMap: deploymentID -> endpointInfo -> historyStatus
	historicalEndpoints map[string]map[net.NumericEndpoint]*entityStatus
	// historicalIPs is mimicking ipMap: IP Address -> deploymentID -> historyStatus
	historicalIPs map[net.IPAddress]map[string]*entityStatus
	// historicalContainerIDs is mimicking containerIDMap: container IDs -> container metadata -> historyStatus
	historicalContainerIDs map[string]map[ContainerMetadata]*entityStatus
}

// NewStore creates and returns a new store instance.
func NewStore() *Store {
	return NewStoreWithMemory(0)
}

// NewStoreWithMemory returns store that remembers past IPs of an endpoint for a given number of ticks
func NewStoreWithMemory(numTicks uint16) *Store {
	store := &Store{entitiesMemorySize: numTicks}
	store.initMaps()
	return store
}

func (e *Store) initMaps() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.ipMap = make(map[net.IPAddress]map[string]struct{})
	e.endpointMap = make(map[net.NumericEndpoint]map[string]map[EndpointTargetInfo]struct{})
	e.containerIDMap = make(map[string]ContainerMetadata)
	e.reverseIPMap = make(map[string]map[net.IPAddress]struct{})
	e.reverseEndpointMap = make(map[string]map[net.NumericEndpoint]struct{})
	e.reverseContainerIDMap = make(map[string]map[string]struct{})
	e.publicIPRefCounts = make(map[net.IPAddress]*int)
	e.publicIPsListeners = make(map[PublicIPsListener]struct{})

	e.historicalEndpoints = make(map[string]map[net.NumericEndpoint]*entityStatus)
	e.historicalIPs = make(map[net.IPAddress]map[string]*entityStatus)
	e.historicalContainerIDs = make(map[string]map[ContainerMetadata]*entityStatus)
}

func (e *Store) resetMaps() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	// Maps holding historical data must not be wiped on reset! Instead, all entities must be marked as historical.
	// Must be called before the respective source maps are wiped!
	e.resetHistoricalMapsNoLock()

	e.ipMap = make(map[net.IPAddress]map[string]struct{})
	e.endpointMap = make(map[net.NumericEndpoint]map[string]map[EndpointTargetInfo]struct{})
	e.containerIDMap = make(map[string]ContainerMetadata)
	e.reverseIPMap = make(map[string]map[net.IPAddress]struct{})
	e.reverseEndpointMap = make(map[string]map[net.NumericEndpoint]struct{})
	e.reverseContainerIDMap = make(map[string]map[string]struct{})
	e.publicIPRefCounts = make(map[net.IPAddress]*int)
	e.publicIPsListeners = make(map[PublicIPsListener]struct{})
}

// resetHistoricalMapsNoLock ensures that all current data is moved to history. The history will be wiped after
// configured number of ticks
func (e *Store) resetHistoricalMapsNoLock() {
	// FIXME: This is probably wrong - on transition to offline, we may lose those histories
	e.historicalEndpoints = make(map[string]map[net.NumericEndpoint]*entityStatus)
	e.historicalIPs = make(map[net.IPAddress]map[string]*entityStatus)

	e.markAllContainerIDsHistorical()
}

// EndpointTargetInfo is the target port for an endpoint (container port, service port etc.).
type EndpointTargetInfo struct {
	ContainerPort uint16
	PortName      string
}

// EntityData is a data structure representing the updates to be applied to the store for a given deployment.
type EntityData struct {
	ips          map[net.IPAddress]struct{}
	endpoints    map[net.NumericEndpoint][]EndpointTargetInfo
	containerIDs map[string]ContainerMetadata
}

// AddIP adds an IP address to the set of IP addresses of the respective deployment.
func (ed *EntityData) AddIP(ip net.IPAddress) {
	if ed.ips == nil {
		ed.ips = make(map[net.IPAddress]struct{})
	}
	ed.ips[ip] = struct{}{}
}

// AddEndpoint adds an endpoint along with target info to the endpoints of the respective deployment.
func (ed *EntityData) AddEndpoint(ep net.NumericEndpoint, info EndpointTargetInfo) {
	if ed.endpoints == nil {
		ed.endpoints = make(map[net.NumericEndpoint][]EndpointTargetInfo)
	}
	ed.endpoints[ep] = append(ed.endpoints[ep], info)
}

// AddContainerID adds a container ID to the container IDs of the respective deployment.
func (ed *EntityData) AddContainerID(containerID string, container ContainerMetadata) {
	if ed.containerIDs == nil {
		ed.containerIDs = make(map[string]ContainerMetadata)
	}
	ed.containerIDs[containerID] = container
}

func (e *Store) updateMetrics() {
	metrics.UpdateNumberContainersInEntityStored(len(e.containerIDMap))
	metrics.UpdateNumberHistoricalContainersInEntityStored(len(e.historicalContainerIDs))
}

// Cleanup deletes all entries from store (or marks them as historical)
func (e *Store) Cleanup() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	defer e.updateMetrics()
	e.resetMaps()
}

// Apply applies an update to the store. If incremental is true, data will be added; otherwise, data for each deployment
// that is a key in the map will be replaced (or deleted).
func (e *Store) Apply(updates map[string]*EntityData, incremental bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	defer e.updateMetrics()
	e.applyNoLock(updates, incremental)
}

// RecordTick records the information that a unit of time (one tick) has passed
func (e *Store) RecordTick() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	for deploymentID, m := range e.historicalEndpoints {
		for endpoint, status := range m {
			status.recordTick()
			// Remove all historical entries that expired in this tick.
			e.removeHistoricalExpiredDeploymentEndpoints(deploymentID, endpoint)
		}
	}
	for ip, m := range e.historicalIPs {
		for deploymentID, status := range m {
			status.recordTick()
			// Remove all historical entries that expired in this tick.
			e.removeHistoricalExpiredIPs(deploymentID, ip)
		}
	}
	for id, metaMap := range e.historicalContainerIDs {
		for metadata, status := range metaMap {
			status.recordTick()
			// Remove all historical entries that expired in this tick.
			e.removeHistoricalExpiredContainerIDs(id, metadata)
		}
	}
}

func (e *Store) removeDeploymentEndpoints(deploymentID string, ep net.NumericEndpoint) {
	delete(e.endpointMap[ep], deploymentID)
	if len(e.endpointMap[ep]) == 0 {
		delete(e.endpointMap, ep)
	}

	delete(e.reverseEndpointMap[deploymentID], ep)
	if len(e.reverseEndpointMap[deploymentID]) == 0 {
		delete(e.reverseEndpointMap, deploymentID)
	}
}

func (e *Store) removeHistoricalExpiredDeploymentEndpoints(deploymentID string, ep net.NumericEndpoint) {
	if status, ok := e.historicalEndpoints[deploymentID][ep]; ok && status.IsExpired() {
		e.removeDeploymentEndpoints(deploymentID, ep)
		delete(e.historicalEndpoints[deploymentID], ep)
		if len(e.historicalEndpoints[deploymentID]) == 0 {
			delete(e.historicalEndpoints, deploymentID)
		}
	}
}

// unmarkEndpointHistorical marks previously marked historical endpoint as no longer historical
func (e *Store) unmarkEndpointHistorical(deploymentID string, ep net.NumericEndpoint) {
	if _, ok := e.historicalEndpoints[deploymentID]; !ok {
		return
	}
	delete(e.historicalEndpoints[deploymentID], ep)
	if len(e.historicalEndpoints[deploymentID]) == 0 {
		delete(e.historicalEndpoints, deploymentID)
	}
}

func (e *Store) markEndpointHistorical(deploymentID string, ep net.NumericEndpoint) {
	if _, ok := e.historicalEndpoints[deploymentID]; !ok {
		e.historicalEndpoints[deploymentID] = make(map[net.NumericEndpoint]*entityStatus)
	}
	es := e.historicalEndpoints[deploymentID][ep]
	if es == nil {
		es = newEntityStatus(e.entitiesMemorySize)
	}
	es.markHistorical(e.entitiesMemorySize)
	e.historicalEndpoints[deploymentID][ep] = es
}

// markAllContainerIDsHistorical does a history-preserving version of removeAll
func (e *Store) markAllContainerIDsHistorical() {
	for contID, metadata := range e.containerIDMap {
		e.markContainerIDHistorical(contID, metadata)
	}
}

func (e *Store) markContainerIDHistorical(contID string, meta ContainerMetadata) {
	// Marking something historical should not happen when the memory setting is 0.
	if e.entitiesMemorySize == 0 {
		return
	}
	if _, ok := e.historicalContainerIDs[contID]; !ok {
		e.historicalContainerIDs[contID] = make(map[ContainerMetadata]*entityStatus)
	}
	es := e.historicalContainerIDs[contID][meta]
	if es == nil {
		es = newEntityStatus(e.entitiesMemorySize)
	}
	es.markHistorical(e.entitiesMemorySize)
	e.historicalContainerIDs[contID][meta] = es
}

// unmarkContainerIDHistorical marks previously marked historical containerID as no longer historical
func (e *Store) unmarkContainerIDHistorical(contID string, meta ContainerMetadata) {
	if _, ok := e.historicalContainerIDs[contID]; !ok {
		return
	}
	delete(e.historicalContainerIDs[contID], meta)
	if len(e.historicalContainerIDs[contID]) == 0 {
		delete(e.historicalContainerIDs, contID)
	}
}

func (e *Store) removeDeploymentIP(deploymentID string, ip net.IPAddress) {
	delete(e.ipMap[ip], deploymentID)
	if len(e.ipMap[ip]) == 0 {
		delete(e.ipMap, ip)
	}

	delete(e.reverseIPMap[deploymentID], ip)
	if len(e.reverseIPMap[deploymentID]) == 0 {
		delete(e.reverseIPMap, deploymentID)
	}
}

func (e *Store) removeHistoricalExpiredIPs(deploymentID string, ip net.IPAddress) {
	if status, ok := e.historicalIPs[ip][deploymentID]; ok && status.IsExpired() {
		e.removeDeploymentIP(deploymentID, ip)
		delete(e.historicalIPs[ip], deploymentID)
		if len(e.historicalIPs[ip]) == 0 {
			delete(e.historicalIPs, ip)
		}
	}
}

func (e *Store) removeHistoricalExpiredContainerIDs(contID string, meta ContainerMetadata) {
	if status, ok := e.historicalContainerIDs[contID][meta]; ok && status.IsExpired() {
		// remove in the non-historical storage
		delete(e.containerIDMap, contID)

		delete(e.historicalContainerIDs[contID], meta)
		if len(e.historicalContainerIDs[contID]) == 0 {
			delete(e.historicalContainerIDs, contID)
		}
	}
}

// unmarkHistoricalIP marks previously marked historical IP as no longer historical
func (e *Store) unmarkHistoricalIP(deploymentID string, ip net.IPAddress) {
	if _, ok := e.historicalIPs[ip]; !ok {
		return
	}
	delete(e.historicalIPs[ip], deploymentID)
	if len(e.historicalIPs[ip]) == 0 {
		delete(e.historicalIPs, ip)
	}
}

func (e *Store) markHistoricalIP(deploymentID string, ip net.IPAddress) {
	if _, ok := e.historicalIPs[ip]; !ok {
		e.historicalIPs[ip] = make(map[string]*entityStatus)
	}
	es := e.historicalIPs[ip][deploymentID]
	if es == nil {
		es = newEntityStatus(e.entitiesMemorySize)
	}
	es.markHistorical(e.entitiesMemorySize)
	e.historicalIPs[ip][deploymentID] = es
}

func (e *Store) purgeNoLock(deploymentID string) {
	for ip := range e.reverseIPMap[deploymentID] {
		e.markHistoricalIP(deploymentID, ip)
		// For entitiesMemorySize > 0, the deletion of historical expired entries happens after a tick.
		// If memory is disabled, we should not wait for a tick and delete them immediately.
		e.removeHistoricalExpiredIPs(deploymentID, ip)

		if len(e.ipMap[ip]) == 0 {
			delete(e.ipMap, ip)
			if ip.IsPublic() {
				e.decPublicIPRefNoLock(ip)
			}
		}
	}
	for ep := range e.reverseEndpointMap[deploymentID] {
		e.markEndpointHistorical(deploymentID, ep)
		// For entitiesMemorySize > 0, the deletion of historical expired entries happens after a tick.
		// If memory is disabled, we should delete historical expired entries immediately.
		e.removeHistoricalExpiredDeploymentEndpoints(deploymentID, ep)

		if len(e.endpointMap[ep]) == 0 {
			delete(e.endpointMap, ep)
			if ipAddr := ep.IPAndPort.Address; ipAddr.IsPublic() {
				e.decPublicIPRefNoLock(ipAddr)
			}
		}
	}

	for containerID := range e.reverseContainerIDMap[deploymentID] {
		if meta, found := e.containerIDMap[containerID]; found {
			e.markContainerIDHistorical(containerID, meta)
			// If memorySize is set to 0, this must be called here
			// as it is an equivalent of: delete(e.containerIDMap, containerID)
			e.removeHistoricalExpiredContainerIDs(containerID, meta)
		}
		delete(e.containerIDMap, containerID)
	}

	delete(e.reverseIPMap, deploymentID)
	delete(e.reverseEndpointMap, deploymentID)
	delete(e.reverseContainerIDMap, deploymentID)
}

func (e *Store) applyNoLock(updates map[string]*EntityData, incremental bool) {
	if !incremental {
		for deploymentID := range updates {
			e.purgeNoLock(deploymentID)
		}
	}

	for deploymentID, data := range updates {
		if data == nil {
			continue
		}
		e.applySingleNoLock(deploymentID, *data)
	}
}

func (e *Store) applySingleNoLock(deploymentID string, data EntityData) {
	reverseEPs := e.reverseEndpointMap[deploymentID]
	reverseIPs := e.reverseIPMap[deploymentID]
	reverseContainerIDs := e.reverseContainerIDMap[deploymentID]

	for ep, targetInfos := range data.endpoints {
		if reverseEPs == nil {
			reverseEPs = make(map[net.NumericEndpoint]struct{})
			e.reverseEndpointMap[deploymentID] = reverseEPs
		}
		reverseEPs[ep] = struct{}{}

		epMap := e.endpointMap[ep]
		if epMap == nil {
			epMap = make(map[string]map[EndpointTargetInfo]struct{})
			e.endpointMap[ep] = epMap
			if ipAddr := ep.IPAndPort.Address; ipAddr.IsPublic() {
				e.incPublicIPRefNoLock(ipAddr)
			}
		}
		targetSet := epMap[deploymentID]
		if targetSet == nil {
			targetSet = make(map[EndpointTargetInfo]struct{})
			epMap[deploymentID] = targetSet
		}
		for _, tgtInfo := range targetInfos {
			targetSet[tgtInfo] = struct{}{}
		}
		// Endpoints previously marked as historical would expire soon, so we must mark them as no longer historical.
		e.unmarkEndpointHistorical(deploymentID, ep)
	}

	for ip := range data.ips {
		if reverseIPs == nil {
			reverseIPs = make(map[net.IPAddress]struct{})
			e.reverseIPMap[deploymentID] = reverseIPs
		}
		reverseIPs[ip] = struct{}{}

		ipMap := e.ipMap[ip]
		if ipMap == nil {
			ipMap = make(map[string]struct{})
			e.ipMap[ip] = ipMap
			if ip.IsPublic() {
				e.incPublicIPRefNoLock(ip)
			}
		}
		ipMap[deploymentID] = struct{}{}
		// IP previously marked as historical would expire soon, so we must mark them as no longer historical.
		e.unmarkHistoricalIP(deploymentID, ip)
	}

	mdsForCallback := make([]ContainerMetadata, 0, len(data.containerIDs))
	for containerID, metadata := range data.containerIDs {
		if reverseContainerIDs == nil {
			reverseContainerIDs = make(map[string]struct{})
			e.reverseContainerIDMap[deploymentID] = reverseContainerIDs
		}
		reverseContainerIDs[containerID] = struct{}{}
		e.containerIDMap[containerID] = metadata
		// We must unmark if the container was previously marked as historical, otherwise it will expire
		e.unmarkContainerIDHistorical(containerID, metadata)
		mdsForCallback = append(mdsForCallback, metadata)
	}

	if e.callbackChannel != nil && len(mdsForCallback) > 0 {
		go sendMetadataCallbacks(e.callbackChannel, mdsForCallback)
	}
}

func sendMetadataCallbacks(callbackC chan<- ContainerMetadata, mds []ContainerMetadata) {
	for _, md := range mds {
		callbackC <- md
	}
}

// RegisterContainerMetadataCallbackChannel registers the given channel as the callback channel for container metadata.
// Any previously registered callback channel will get overwritten by repeatedly calling this method. The previous
// callback channel (if any) is returned by this function.
func (e *Store) RegisterContainerMetadataCallbackChannel(callbackChan chan<- ContainerMetadata) chan<- ContainerMetadata {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	oldChan := e.callbackChannel
	e.callbackChannel = callbackChan
	return oldChan
}

// LookupResult contains the result of a lookup operation.
type LookupResult struct {
	Entity         networkgraph.Entity
	ContainerPorts []uint16
	PortNames      []string
}

// LookupByEndpoint returns possible target deployments by endpoint (if any).
func (e *Store) LookupByEndpoint(endpoint net.NumericEndpoint) []LookupResult {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.lookupNoLock(endpoint)
}

// LookupByContainerID retrieves the deployment ID by a container ID.
func (e *Store) LookupByContainerID(containerID string) (ContainerMetadata, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	// Search in the non-historical memory
	if metadata, ok := e.lookupByContainerIDNoLock(containerID); ok {
		return metadata, true
	}
	// Search in history
	return e.lookupByContainerIDInHistoryNoLock(containerID)
}

// lookupByContainerIDNoLock retrieves the deployment ID by a container ID from the non-historical data in the map.
func (e *Store) lookupByContainerIDNoLock(containerID string) (ContainerMetadata, bool) {
	if metadata, ok := e.containerIDMap[containerID]; ok {
		return metadata, true
	}
	return ContainerMetadata{}, false
}

// lookupByContainerIDInHistoryNoLock does the same as LookupByContainerID but searches only among historical entries.
func (e *Store) lookupByContainerIDInHistoryNoLock(containerID string) (ContainerMetadata, bool) {
	if metaHistory, ok := e.historicalContainerIDs[containerID]; ok {
		// The metaHistory map contains 0 or 1 elements
		for metadata := range metaHistory {
			return metadata, true
		}
	}
	return ContainerMetadata{}, false
}

func (e *Store) lookupNoLock(endpoint net.NumericEndpoint) (results []LookupResult) {
	for deploymentID, targetInfoSet := range e.endpointMap[endpoint] {
		result := LookupResult{
			Entity:         networkgraph.EntityForDeployment(deploymentID),
			ContainerPorts: make([]uint16, 0, len(targetInfoSet)),
		}
		for tgtInfo := range targetInfoSet {
			result.ContainerPorts = append(result.ContainerPorts, tgtInfo.ContainerPort)
			if tgtInfo.PortName != "" {
				result.PortNames = append(result.PortNames, tgtInfo.PortName)
			}
		}
		results = append(results, result)
	}

	if len(results) > 0 {
		return
	}

	for deploymentID := range e.ipMap[endpoint.IPAndPort.Address] {
		result := LookupResult{
			Entity:         networkgraph.EntityForDeployment(deploymentID),
			ContainerPorts: []uint16{endpoint.IPAndPort.Port},
		}
		results = append(results, result)
	}

	return
}

// RegisterPublicIPsListener registers a listener that listens to changes to the set of public IP addresses.
// It returns a boolean indicating whether the listener was actually unregistered (i.e., a return value of false
// indicates that the listener was already registered).
func (e *Store) RegisterPublicIPsListener(listener PublicIPsListener) bool {
	// This mutex is pretty broad in scope, but since registering listeners occurs rarely, it's better than adding
	// another mutex that would need to get locked separately.
	e.mutex.Lock()
	defer e.mutex.Unlock()

	oldLen := len(e.publicIPsListeners)
	e.publicIPsListeners[listener] = struct{}{}

	return len(e.publicIPsListeners) > oldLen
}

// UnregisterPublicIPsListener unregisters a previously registered listener for public IP events. It returns a boolean
// indicating whether the listener was actually unregistered (i.e., a return value of false indicates that the listener
// was not registered in the first place).
func (e *Store) UnregisterPublicIPsListener(listener PublicIPsListener) bool {
	e.mutex.Lock()
	defer e.mutex.Lock()

	oldLen := len(e.publicIPsListeners)
	delete(e.publicIPsListeners, listener)

	return len(e.publicIPsListeners) < oldLen
}

func (e *Store) incPublicIPRefNoLock(addr net.IPAddress) {
	refCnt := e.publicIPRefCounts[addr]
	if refCnt == nil {
		refCnt = new(int)
		e.publicIPRefCounts[addr] = refCnt
		e.notifyPublicIPsListenersNoLock(PublicIPsListener.OnAdded, addr)
	}
	*refCnt++
}

func (e *Store) decPublicIPRefNoLock(addr net.IPAddress) {
	refCnt := e.publicIPRefCounts[addr]
	if refCnt == nil {
		utils.Should(errors.New("public IP has zero refcount already"))
		return
	}

	*refCnt--
	if *refCnt == 0 {
		delete(e.publicIPRefCounts, addr)
		e.notifyPublicIPsListenersNoLock(PublicIPsListener.OnRemoved, addr)
	}
}

func (e *Store) notifyPublicIPsListenersNoLock(notifyFunc func(PublicIPsListener, net.IPAddress), ip net.IPAddress) {
	for listener := range e.publicIPsListeners {
		notifyFunc(listener, ip)
	}
}
