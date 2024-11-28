package clusterentities

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/exp/maps"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
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
	OnAdded(ip net.IPAddress)
	OnRemoved(ip net.IPAddress)
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

	ipRefCountMutex    sync.RWMutex
	publicIPRefCounts  map[net.IPAddress]*int
	publicIPsListeners map[PublicIPsListener]struct{}
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

// StartDebugServer starts HTTP server that allows to look inside the clusterentities store.
// This blocks and should be always started in a goroutine!
func (e *Store) StartDebugServer() {
	http.HandleFunc("/debug/clusterentities/state.json", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n, err := fmt.Fprintf(w, "%s\n", e.Debug())
		log.Debugf("Serving debug http endpoint: n=%d, err=%v", n, err)
	})
	err := http.ListenAndServe(":8099", nil)
	if err != nil {
		log.Error(errors.Wrap(err, "unable to start cluster entities store debug server"))
	}
}

func (e *Store) initMaps() {
	e.ipRefCountMutex.Lock()
	defer e.ipRefCountMutex.Unlock()
	e.publicIPRefCounts = make(map[net.IPAddress]*int)
	e.publicIPsListeners = make(map[PublicIPsListener]struct{})
	e.trace = make(map[string]string)
	if !e.debugMode {
		concurrency.WithLock(&e.traceMutex, func() {
			e.trace["init"] = "events trace disabled in non-debug mode"
		})
	}
}

func (e *Store) resetMaps() {
	concurrency.WithLock(&e.ipRefCountMutex, func() {
		if e.memorySize == 0 {
			e.publicIPRefCounts = make(map[net.IPAddress]*int)
		}
		// Call to e.podIPsStore.resetMaps() will move all IPs to history, so we do not reset the publicIPRefCounts.
		// publicIPsListeners should not be reset at all, as we have no guarantee that the listeners will be re-added.
	})
	e.endpointsStore.resetMaps()
	e.podIPsStore.resetMaps()
	e.containerIDsStore.resetMaps()
}

// Cleanup deletes all entries from store
func (e *Store) Cleanup() {
	e.resetMaps()
}

// Apply applies an update to the store. If incremental is true, data will be added; otherwise, data for each deployment
// that is a key in the map will be replaced (or deleted).
func (e *Store) Apply(updates map[string]*EntityData, incremental bool, auxInfo ...string) {
	for id, data := range updates {
		e.track("add-deployment (%s) overwrite=%t ID=%s, data=%v", auxInfo, !incremental, id, data.String())
	}
	// We track the number of references to Public IPs.
	// Each operation may cause the counter to be incremented or decremented.

	// Order matters: Endpoints must be applied before IPs, as the IP store may query the endpoints store to check
	// whether a given IP is used by other endpoints.
	preIPs := e.currentlyStoredPublicIPs()
	e.endpointsStore.Apply(updates, incremental)
	e.podIPsStore.Apply(updates, incremental)
	postIPs := e.currentlyStoredPublicIPs()

	for ip := range postIPs.Difference(preIPs) {
		e.incPublicIPRef(ip)
	}
	for ip := range preIPs.Difference(postIPs) {
		e.decPublicIPRef(ip)
	}

	callbacks := e.containerIDsStore.Apply(updates, incremental)
	if callbacks != nil {
		if e.callbackChannel != nil && len(callbacks) > 0 {
			go sendMetadataCallbacks(e.callbackChannel, callbacks)
		}
	}
}

// currentlyStoredPublicIPs is an easy (but computationally costly) method to get all public IPs stored in the store.
// Implementing smarter way of counting the IPs is a bit tricky and requires many set operations,
// thus this computationally-expensive method may be not so expensive in general.
func (e *Store) currentlyStoredPublicIPs() set.Set[net.IPAddress] {
	s := set.NewSet[net.IPAddress]()
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
	return s
}

// RecordTick records the information that a unit of time (1 tick) has passed
func (e *Store) RecordTick() {
	e.track("Tick")
	// There may be public pod IP addresses expiring in this tick, and we may need to decrement the counters for them.
	preIPs := e.currentlyStoredPublicIPs()
	e.podIPsStore.RecordTick()
	e.endpointsStore.RecordTick()
	postIPs := e.currentlyStoredPublicIPs()
	for ip := range preIPs.Difference(postIPs) {
		e.decPublicIPRef(ip)
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
	e.ipRefCountMutex.Lock()
	defer e.ipRefCountMutex.Unlock()

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
		return append(current, historical...)
	}
	// TODO: cover this if-case with tests!
	if len(ipLookup)+len(ipLookupHistorical) > 0 {
		e.track("LookupByEndpoint(%s): found=true, foundIn=ipLookup", endpoint.String())
		return append(ipLookupHistorical, ipLookup...)
	}
	e.track("LookupByEndpoint(%s): found=false", endpoint.String())
	return []LookupResult{}
}

// LookupByContainerID retrieves the deployment ID by a container ID.
func (e *Store) LookupByContainerID(containerID string) (result ContainerMetadata, found bool) {
	result, found, _ = e.containerIDsStore.lookupByContainer(containerID)
	e.track("LookupByContainerID(%s): found=%t", containerID, found)
	return result, found
}

// RegisterPublicIPsListener registers a listener that listens on changes to the set of public IP addresses.
// It returns a boolean indicating whether the listener was actually unregistered (i.e., a return value of false
// indicates that the listener was already registered).
func (e *Store) RegisterPublicIPsListener(listener PublicIPsListener) bool {
	// This ipRefCountMutex is pretty broad in scope, but since registering listeners occurs rarely, it's better than adding
	// another ipRefCountMutex that would need to get locked separately.
	e.ipRefCountMutex.Lock()
	defer e.ipRefCountMutex.Unlock()

	oldLen := len(e.publicIPsListeners)
	e.publicIPsListeners[listener] = struct{}{}

	return len(e.publicIPsListeners) > oldLen
}

// UnregisterPublicIPsListener unregisters a previously registered listener for public IP events. It returns a boolean
// indicating whether the listener was actually unregistered (i.e., a return value of false indicates that the listener
// was not registered in the first place).
func (e *Store) UnregisterPublicIPsListener(listener PublicIPsListener) bool {
	e.ipRefCountMutex.Lock()
	defer e.ipRefCountMutex.Unlock()

	oldLen := len(e.publicIPsListeners)
	delete(e.publicIPsListeners, listener)

	return len(e.publicIPsListeners) < oldLen
}

func (e *Store) incPublicIPRef(addr net.IPAddress) {
	e.ipRefCountMutex.Lock()
	defer e.ipRefCountMutex.Unlock()
	refCnt := e.publicIPRefCounts[addr]
	if refCnt == nil {
		refCnt = new(int)
		e.publicIPRefCounts[addr] = refCnt
		e.notifyPublicIPsListenersNoLock(PublicIPsListener.OnAdded, addr)
	}
	*refCnt++
	log.Debugf("Increasing count for %s: now is %d", addr.String(), *refCnt)
}

func (e *Store) decPublicIPRef(addr net.IPAddress) {
	e.ipRefCountMutex.Lock()
	defer e.ipRefCountMutex.Unlock()
	refCnt := e.publicIPRefCounts[addr]
	if refCnt == nil {
		utils.Should(fmt.Errorf("public IP %s has zero refcount already", addr))
		return
	}
	*refCnt--
	log.Debugf("Decreasing count for %s: now is %d", addr.String(), *refCnt)
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
