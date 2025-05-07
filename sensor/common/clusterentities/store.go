package clusterentities

import (
	"fmt"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/exp/maps"
)

// ContainerMetadata is the container metadata that is stored per instance
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
	OnUpdate(ips set.Set[net.IPAddress])
}

// EndpointTargetInfo is the target port for an endpoint (container port, service port, etc.).
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

func (ed *EntityData) String() string {
	if ed == nil {
		return "nil"
	}
	return fmt.Sprintf("ips: %v, endpoints: %v, containerIDs: %v",
		maps.Keys(ed.ips), maps.Keys(ed.endpoints), maps.Keys(ed.containerIDs))
}

// isDeleteOnly prevents from treating a request as ADD with empty values, as such requests should be treated as DELETE
func (ed *EntityData) isDeleteOnly() bool {
	if ed == nil {
		return true
	}
	if len(ed.endpoints)+len(ed.containerIDs)+len(ed.ips) == 0 {
		return true
	}
	return false
}

// AddIP adds an IP address to the set of IP addresses of the respective deployment.
func (ed *EntityData) AddIP(ip net.IPAddress) {
	if ed.ips == nil {
		ed.ips = make(map[net.IPAddress]struct{})
	}
	ed.ips[ip] = struct{}{}
}

// AddEndpoint adds an endpoint along with a target info to the endpoints of the respective deployment.
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

// Store is a store for managing cluster entities (currently deployments only) and allows looking them up by
// endpoint.
type Store struct {
	endpointsStore    *endpointsStore
	podIPsStore       *podIPsStore
	containerIDsStore *containerIDsStore

	publicIPsTrackingMutex sync.RWMutex
	publicIPsListeners     set.Set[PublicIPsListener]
	// callbackChannel is a channel to send container metadata upon resolution
	callbackChannel chan<- ContainerMetadata

	// memorySize defines how many ticks old endpoint data should be remembered after removal request
	// Set to 0 to disable memory
	memorySize uint16

	// list of events for debugging purposes
	debugMode  bool
	traceMutex sync.RWMutex
	trace      map[string]string
}

// NewStore creates and returns a new store instance.
func NewStore() *Store {
	return NewStoreWithMemory(0, false)
}

// NewStoreWithMemory returns store that remembers past IPs of an endpoint for a given number of ticks
func NewStoreWithMemory(numTicks uint16, debugMode bool) *Store {
	store := &Store{
		endpointsStore:    newEndpointsStoreWithMemory(numTicks),
		podIPsStore:       newPodIPsStoreWithMemory(numTicks),
		containerIDsStore: newContainerIDsStoreWithMemory(numTicks),
		memorySize:        numTicks,
		debugMode:         debugMode,
	}
	store.initMaps()
	return store
}

func (e *Store) initMaps() {
	e.publicIPsTrackingMutex.Lock()
	defer e.publicIPsTrackingMutex.Unlock()
	e.publicIPsListeners = set.NewSet[PublicIPsListener]()
	e.trace = make(map[string]string)
	if !e.debugMode {
		concurrency.WithLock(&e.traceMutex, func() {
			e.trace["init"] = "events trace disabled in non-debug mode"
		})
	}
}

func (e *Store) resetMaps() {
	e.endpointsStore.resetMaps()
	e.podIPsStore.resetMaps()
	e.containerIDsStore.resetMaps()
	if e.memorySize == 0 {
		// delete all tracked public IPs
		e.updatePublicIPRefs(set.NewSet[net.IPAddress]())
	}
}

// Cleanup deletes all entries from store
func (e *Store) Cleanup() {
	e.resetMaps()
}

// Apply applies an update to the store. If incremental is true, data will be added; otherwise, data for each deployment
// that is a key in the map will be replaced (or deleted).
func (e *Store) Apply(updates map[string]*EntityData, incremental bool, auxInfo ...string) {
	if e.debugMode {
		for id, data := range updates {
			e.track("add-deployment (%s) overwrite=%t ID=%s, data=%v", auxInfo, !incremental, id, data.String())
		}
	}

	// Order matters: Endpoints must be applied before IPs, as the IP store may query the endpoints store to check
	// whether a given IP is used by other endpoints.
	e.endpointsStore.Apply(updates, incremental)
	e.podIPsStore.Apply(updates, incremental)

	e.updatePublicIPRefs(e.currentlyStoredPublicIPs())

	callbacks := e.containerIDsStore.Apply(updates, incremental)
	if e.callbackChannel != nil && len(callbacks) > 0 {
		go sendMetadataCallbacks(e.callbackChannel, callbacks)
	}
}

// currentlyStoredPublicIPs returns all public IPs currently stored in the store (including history).
func (e *Store) currentlyStoredPublicIPs() set.Set[net.IPAddress] {
	s := set.NewSet[net.IPAddress]()
	concurrency.WithRLock(&e.endpointsStore.mutex, func() {
		for endpoint := range e.endpointsStore.endpointMap {
			if endpoint.IPAndPort.Address.IsPublic() {
				s.Add(endpoint.IPAndPort.Address)
			}
		}
		for endpoint := range e.endpointsStore.historicalEndpoints {
			if endpoint.IPAndPort.Address.IsPublic() {
				s.Add(endpoint.IPAndPort.Address)
			}
		}
	})
	concurrency.WithRLock(&e.podIPsStore.mutex, func() {
		for address := range e.podIPsStore.ipMap {
			if address.IsPublic() {
				s.Add(address)
			}
		}
		for address := range e.podIPsStore.historicalIPs {
			if address.IsPublic() {
				s.Add(address)
			}
		}
	})
	return s
}

// RecordTick records the information that a unit of time (1 tick) has passed
func (e *Store) RecordTick() {
	e.track("Tick")
	// Avoid or-statements like "a.RecordTick() || b.RecordTick()"
	// because there is no guarantee that b.RecordTick() will be called.
	removedPubIP := e.podIPsStore.RecordTick()
	removedEpWithPubIP := e.endpointsStore.RecordTick()
	if removedPubIP || removedEpWithPubIP {
		// If there are any public IPs expiring in this tick, then we need to update the listeners.
		e.updatePublicIPRefs(e.currentlyStoredPublicIPs())
	}
	e.containerIDsStore.RecordTick()
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
	e.publicIPsTrackingMutex.Lock()
	defer e.publicIPsTrackingMutex.Unlock()

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
// If no matching graph Entity is found by Endpoint, the podIP store is searched
func (e *Store) LookupByEndpoint(endpoint net.NumericEndpoint) []LookupResult {
	current, historical, ipLookup, ipLookupHistorical := e.endpointsStore.lookupEndpoint(endpoint, e.podIPsStore)
	// Return early to avoid potential duplicates... not sure if duplicates are bad here.
	if len(current)+len(historical) > 0 {
		e.track("LookupByEndpoint(%s): found=true, foundIn=endpointsStore", endpoint.String())
		log.Debugf("LookupByEndpoint(%s): found=true, foundIn=endpointsStore", endpoint.String())
		return append(current, historical...)
	}
	if len(ipLookup)+len(ipLookupHistorical) > 0 {
		e.track("LookupByEndpoint(%s): found=true, foundIn=ipLookup", endpoint.String())
		log.Debugf("LookupByEndpoint(%s): found=true, foundIn=ipLookup", endpoint.String())
		return append(ipLookupHistorical, ipLookup...)
	}
	e.track("LookupByEndpoint(%s): found=false", endpoint.String())
	log.Debugf("LookupByEndpoint(%s): found=false", endpoint.String())
	return []LookupResult{}
}

func (e *Store) DumpEndpointStore() {
	log.Debug("Dumping e.endpointsStore.endpointMap")
	for numericEndpoint, m := range e.endpointsStore.endpointMap {
		log.Debugf("endpointMap[%s]: %v", numericEndpoint.String(), m)
	}
}

// LookupByContainerID retrieves the deployment ID by a container ID.
func (e *Store) LookupByContainerID(containerID string) (metadata ContainerMetadata, found bool, isHistorical bool) {
	metadata, found, isHistorical = e.containerIDsStore.lookupByContainer(containerID)
	e.track("LookupByContainerID(%s): found=%t", containerID, found)
	return metadata, found, isHistorical
}

// RegisterPublicIPsListener registers a listener that listens on changes to the set of public IP addresses.
// It returns a boolean indicating whether the listener was actually unregistered (i.e., a return value of false
// indicates that the listener was already registered).
func (e *Store) RegisterPublicIPsListener(listener PublicIPsListener) bool {
	// This publicIPsTrackingMutex is pretty broad in scope, but since registering listeners occurs rarely, it's better than adding
	// another publicIPsTrackingMutex that would need to get locked separately.
	e.publicIPsTrackingMutex.Lock()
	defer e.publicIPsTrackingMutex.Unlock()
	return e.publicIPsListeners.Add(listener)
}

// UnregisterPublicIPsListener unregisters a previously registered listener for public IP events. It returns a boolean
// indicating whether the listener was actually unregistered (i.e., a return value of false indicates that the listener
// was not registered in the first place).
func (e *Store) UnregisterPublicIPsListener(listener PublicIPsListener) bool {
	e.publicIPsTrackingMutex.Lock()
	defer e.publicIPsTrackingMutex.Unlock()
	return e.publicIPsListeners.Remove(listener)
}

func (e *Store) updatePublicIPRefs(addrs set.Set[net.IPAddress]) {
	e.notifyPublicIPsListenersNoLock(PublicIPsListener.OnUpdate, addrs)
}

func (e *Store) notifyPublicIPsListenersNoLock(notifyFunc func(PublicIPsListener, set.Set[net.IPAddress]), ips set.Set[net.IPAddress]) {
	e.publicIPsTrackingMutex.RLock()
	defer e.publicIPsTrackingMutex.RUnlock()
	for listener := range e.publicIPsListeners {
		notifyFunc(listener, ips)
	}
}

func prettyPrintHistoricalData[M ~map[K1]map[K2]*entityStatus, K1 comparable, K2 comparable](data M) string {
	if len(data) == 0 {
		return "history is empty"
	}
	fragments := make([]string, 0, len(data))
	for ID, m := range data {
		for _, status := range m {
			fragments = append(fragments,
				fmt.Sprintf("[ID=%v, ticksLeft=%d]", ID, status.ticksLeft))
		}
	}
	return strings.Join(fragments, "\n")
}
