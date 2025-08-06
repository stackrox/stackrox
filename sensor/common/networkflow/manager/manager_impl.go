package manager

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/process/normalize"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sensor/queue"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/externalsrcs"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/metrics"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
	"github.com/stackrox/rox/sensor/common/trace"
	"github.com/stackrox/rox/sensor/common/unimplemented"
)

const (
	connectionDeletionGracePeriod = 5 * time.Minute
	loggingRateLimiter            = "plop-feature-disabled"
)

var (
	emptyProcessInfo = processInfo{}
	enricherCycle    = time.Second * 30
)

type hostConnections struct {
	hostname    string
	mutex       sync.Mutex
	connections map[connection]*connStatus
	endpoints   map[containerEndpoint]*connStatus

	lastKnownTimestamp timestamp.MicroTS
	// connectionsSequenceID is the sequence ID of the current active connections state
	connectionsSequenceID int64
	// currentSequenceID is the sequence ID of the most recent `Register` call
	currentSequenceID int64
	pendingDeletion   *time.Timer
}

type networkConnIndicatorWithAge struct {
	networkConnIndicator
	lastUpdate timestamp.MicroTS
}

type networkConnIndicator struct {
	srcEntity networkgraph.Entity
	dstEntity networkgraph.Entity
	dstPort   uint16
	protocol  storage.L4Protocol
}

func (i *networkConnIndicator) toProto(ts timestamp.MicroTS) *storage.NetworkFlow {
	proto := &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity:  i.srcEntity.ToProto(),
			DstEntity:  i.dstEntity.ToProto(),
			DstPort:    uint32(i.dstPort),
			L4Protocol: i.protocol,
		},
	}

	if ts != timestamp.InfiniteFuture {
		proto.LastSeenTimestamp = protoconv.ConvertMicroTSToProtobufTS(ts)
	}
	return proto
}

type containerEndpointIndicatorWithAge struct {
	containerEndpointIndicator
	lastUpdate timestamp.MicroTS
}

// containerEndpointIndicator is a key in Sensor's maps that track active endpoints. It's set of fields should be minimal.
type containerEndpointIndicator struct {
	entity   networkgraph.Entity
	port     uint16
	protocol storage.L4Protocol
}

func (i *containerEndpointIndicator) toProto(ts timestamp.MicroTS) *storage.NetworkEndpoint {
	proto := &storage.NetworkEndpoint{
		Props: &storage.NetworkEndpointProperties{
			Entity:     i.entity.ToProto(),
			Port:       uint32(i.port),
			L4Protocol: i.protocol,
		},
	}

	if ts != timestamp.InfiniteFuture {
		proto.LastActiveTimestamp = protoconv.ConvertMicroTSToProtobufTS(ts)
	}
	return proto
}

type processUniqueKey struct {
	podID         string
	containerName string
	deploymentID  string
	process       processInfo
}

type processListeningIndicator struct {
	key       processUniqueKey
	port      uint16
	protocol  storage.L4Protocol
	podUID    string
	namespace string
}

func (i *processListeningIndicator) toProto(ts timestamp.MicroTS) *storage.ProcessListeningOnPortFromSensor {
	proto := &storage.ProcessListeningOnPortFromSensor{
		Port:     uint32(i.port),
		Protocol: i.protocol,
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               i.key.podID,
			ContainerName:       i.key.containerName,
			ProcessName:         i.key.process.processName,
			ProcessExecFilePath: i.key.process.processExec,
			ProcessArgs:         i.key.process.processArgs,
		},
		DeploymentId: i.key.deploymentID,
		PodUid:       i.podUID,
		Namespace:    i.namespace,
	}

	if ts != timestamp.InfiniteFuture {
		proto.CloseTimestamp = protoconv.ConvertMicroTSToProtobufTS(ts)
	}

	return proto
}

// connection is an instance of a connection as reported by collector.
// Fields are ordered for memory alignment optimization (as described in https://goperf.dev/01-common-patterns/fields-alignment/)
type connection struct {
	remote      net.NumericEndpoint
	local       net.NetworkPeerID
	containerID string
	incoming    bool
}

func (c *connection) String() string {
	var arrow string
	if c.incoming {
		arrow = "<-"
	} else {
		arrow = "->"
	}
	return fmt.Sprintf("%s: %s %s %s", c.containerID, c.local, arrow, c.remote)
}

// IsExternal returns true when IPv4 does not belong to the private IP addresses; false otherwise.
// Error is returned when IP address is malformed
func (c *connection) IsExternal() (bool, error) {
	addr, err := c.getRemoteIPAddress()
	if err != nil {
		return true, errors.Wrap(err, "unable to determine if flow is external or internal")
	}
	if addr.IsLoopback() {
		return false, errors.New("connection with localhost")
	}
	return addr.IsPublic(), nil
}

// getIPAddress returns the IP address of the connection remote.
// If that IP is unset, it returns the address of the IP Network to which the remote belongs.
// If both are unavaliable, an error is returned.
// This check of both is required, because Collector reports the IP addresses
// either as IPAndPort.Address, or IPAndPort.IPNetwork. The former is used in most cases, but sometimes
// (usually on OCP) the latter is provided. Analyzing only one of those two sources may lead to incorrectly reporting
// a connection as external on the network graph.
func (c *connection) getRemoteIPAddress() (net.IPAddress, error) {
	if c.remote.IPAndPort.IsAddressValid() {
		return c.remote.IPAndPort.Address, nil
	}
	if c.remote.IPAndPort.IPNetwork.IsValid() {
		return c.remote.IPAndPort.IPNetwork.IP(), nil
	}
	return net.IPAddress{}, fmt.Errorf("remote has invalid IP address %q", c.remote.IPAndPort.String())
}

type processInfo struct {
	processName string
	processArgs string
	processExec string
}

func (p *processInfo) String() string {
	return fmt.Sprintf("%s: %s %s", p.processExec, p.processName, p.processArgs)
}

// containerEndpoint represents a container endpoint with fields ordered for memory alignment optimization
// (as described in https://goperf.dev/01-common-patterns/fields-alignment/)
type containerEndpoint struct {
	processKey  processInfo
	endpoint    net.NumericEndpoint
	containerID string
}

func (e *containerEndpoint) String() string {
	return fmt.Sprintf("%s %s: %s", e.containerID, e.processKey, e.endpoint)
}

type Option func(*networkFlowManager)

// WithEnrichTicker overrides the default enrichment ticker
func WithEnrichTicker(ticker <-chan time.Time) Option {
	return func(manager *networkFlowManager) {
		if ticker != nil {
			manager.enricherTickerC = ticker
		}
	}
}

// NewManager creates a new instance of network flow manager
func NewManager(
	clusterEntities EntityStore,
	externalSrcs externalsrcs.Store,
	policyDetector detector.Detector,
	pubSub *internalmessage.MessageSubscriber,
	opts ...Option,
) Manager {
	enricherTicker := time.NewTicker(enricherCycle)

	mgr := &networkFlowManager{
		connectionsByHost: make(map[string]*hostConnections),
		clusterEntities:   clusterEntities,
		publicIPs:         newPublicIPsManager(),
		externalSrcs:      externalSrcs,
		policyDetector:    policyDetector,
		enricherTicker:    enricherTicker,
		enricherTickerC:   enricherTicker.C,
		initialSync:       &atomic.Bool{},
		activeConnections: make(map[connection]*networkConnIndicatorWithAge),
		activeEndpoints:   make(map[containerEndpoint]*containerEndpointIndicatorWithAge),
		stopper:           concurrency.NewStopper(),
		pubSub:            pubSub,
	}
	maxAgeSetting := env.EnrichmentPurgerTickerMaxAge.DurationSetting()
	if maxAgeSetting > 0 && maxAgeSetting <= enricherCycle {
		log.Warnf("ROX_ENRICHMENT_PURGER_MAX_AGE (%s) must be higher than enricher cycle (%s). "+
			"Applying default of 4 hours", maxAgeSetting, enricherCycle)
		maxAgeSetting = 4 * time.Hour
	}
	mgr.purger = NewNetworkFlowPurger(clusterEntities, maxAgeSetting, WithManager(mgr))

	enricherTicker.Stop()
	if features.SensorCapturesIntermediateEvents.Enabled() {
		mgr.sensorUpdates = make(chan *message.ExpiringMessage, queue.ScaleSizeOnNonDefault(env.NetworkFlowBufferSize))
	} else {
		mgr.sensorUpdates = make(chan *message.ExpiringMessage)
	}

	if err := mgr.pubSub.Subscribe(internalmessage.SensorMessageResourceSyncFinished, func(msg *internalmessage.SensorInternalMessage) {
		if msg.IsExpired() {
			return
		}
		// Since we need to have the logic to transition to offline mode if `SensorCapturesIntermediateEvents` is disabled.
		// We call `Notify` here to keep the logic to transition offline/online in the same place.
		mgr.Notify(common.SensorComponentEventResourceSyncFinished)
	}); err != nil {
		log.Errorf("unable to subscribe to %s: %+v", internalmessage.SensorMessageResourceSyncFinished, err)
	}
	for _, o := range opts {
		o(mgr)
	}
	return mgr
}

// networkFlowComponent represents a sub-component of the networkFlowManager
type networkFlowComponent interface {
	Start() error
	Stop()
	Notify(common.SensorComponentEvent)
}

type networkFlowManager struct {
	unimplemented.Receiver

	connectionsByHost      map[string]*hostConnections
	connectionsByHostMutex sync.RWMutex

	clusterEntities EntityStore
	externalSrcs    externalsrcs.Store

	lastSentStateMutex             sync.RWMutex
	enrichedConnsLastSentState     map[networkConnIndicator]timestamp.MicroTS
	enrichedEndpointsLastSentState map[containerEndpointIndicator]timestamp.MicroTS
	enrichedProcessesLastSentState map[processListeningIndicator]timestamp.MicroTS

	activeConnectionsMutex sync.RWMutex
	// activeConnections tracks all connections reported by Collector that are believed to be active.
	// A connection is active until Collector sends a NetworkConnectionInfo message with `lastSeen` set to a non-nil value,
	// or until Sensor decides that such message may never arrive and decides that a given connection is no longer active.
	activeConnections    map[connection]*networkConnIndicatorWithAge
	activeEndpointsMutex sync.RWMutex
	// An endpoint is active until Collector sends a NetworkConnectionInfo message with `lastSeen` set to a non-nil value,
	// or until Sensor decides that such message may never arrive and decides that a given endpoint is no longer active.
	activeEndpoints map[containerEndpoint]*containerEndpointIndicatorWithAge

	sensorUpdates chan *message.ExpiringMessage
	centralReady  concurrency.Signal

	ctxMutex    sync.Mutex
	cancelCtx   context.CancelFunc
	pipelineCtx context.Context
	initialSync *atomic.Bool

	enricherTicker  *time.Ticker
	enricherTickerC <-chan time.Time

	publicIPs *publicIPsManager

	policyDetector detector.Detector

	stopper concurrency.Stopper
	purger  networkFlowComponent
	pubSub  *internalmessage.MessageSubscriber
}

func (m *networkFlowManager) Name() string {
	return "networkflow.manager.networkFlowManager"
}

func (m *networkFlowManager) Start() error {
	go m.enrichConnections(m.enricherTickerC)
	go m.publicIPs.Run(m.stopper.LowLevel().GetStopRequestSignal(), m.clusterEntities)
	if m.purger != nil {
		if err := m.purger.Start(); err != nil {
			log.Warnf("Not starting network flow purger: %s", err)
		}
	}
	return nil
}

func (m *networkFlowManager) Stop() {
	if !m.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = m.stopper.Client().Stopped().Wait()
		}()
	}
	m.stopper.Client().Stop()
	if m.purger != nil {
		m.purger.Stop()
	}
}

func (m *networkFlowManager) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (m *networkFlowManager) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, "NetworkFlowManager"))
	// Ensure that the sub-components are notified after this manager processes the notification.
	defer func() {
		if m.purger != nil {
			m.purger.Notify(e)
		}
	}()
	switch e {
	case common.SensorComponentEventResourceSyncFinished:
		if features.SensorCapturesIntermediateEvents.Enabled() {
			if m.initialSync.CompareAndSwap(false, true) {
				m.enricherTicker.Reset(enricherCycle)
			}
			return
		}
		m.resetContext()
		m.resetLastSentState()
		m.centralReady.Signal()
		m.enricherTicker.Reset(enricherCycle)
	case common.SensorComponentEventOfflineMode:
		if features.SensorCapturesIntermediateEvents.Enabled() {
			return
		}
		m.centralReady.Reset()
		m.enricherTicker.Stop()
	}
}

func (m *networkFlowManager) ResponsesC() <-chan *message.ExpiringMessage {
	return m.sensorUpdates
}

func (m *networkFlowManager) resetContext() {
	m.ctxMutex.Lock()
	defer m.ctxMutex.Unlock()
	if m.cancelCtx != nil {
		m.cancelCtx()
	}
	m.pipelineCtx, m.cancelCtx = context.WithCancel(trace.Background())
}

func (m *networkFlowManager) sendToCentral(msg *central.MsgFromSensor) bool {
	if features.SensorCapturesIntermediateEvents.Enabled() {
		select {
		case <-m.stopper.Flow().StopRequested():
			return false
		case m.sensorUpdates <- message.New(msg):
			return true
		default:
			// If the m.sensorUpdates queue is full, we bounce the Network Flow update.
			// They will still be processed by the detection engine for newer entities, but
			// sensor will not keep ordered updates indefinitely in memory.
			return false
		}
	} else {
		ctx := m.getCurrentContext()
		select {
		case <-m.stopper.Flow().StopRequested():
			return false
		case m.sensorUpdates <- message.NewExpiring(ctx, msg):
			return true
		}
	}
}

func (m *networkFlowManager) resetLastSentState() {
	m.lastSentStateMutex.Lock()
	defer m.lastSentStateMutex.Unlock()
	m.enrichedConnsLastSentState = nil
	m.enrichedEndpointsLastSentState = nil
	m.enrichedProcessesLastSentState = nil
}

func (m *networkFlowManager) updateConnectionStates(newConns map[networkConnIndicator]timestamp.MicroTS, newEndpoints map[containerEndpointIndicator]timestamp.MicroTS) {
	m.lastSentStateMutex.Lock()
	defer m.lastSentStateMutex.Unlock()
	m.enrichedConnsLastSentState = newConns
	m.enrichedEndpointsLastSentState = newEndpoints
}

func (m *networkFlowManager) updateProcessesState(newProcesses map[processListeningIndicator]timestamp.MicroTS) {
	m.lastSentStateMutex.Lock()
	defer m.lastSentStateMutex.Unlock()
	m.enrichedProcessesLastSentState = newProcesses
}

func (m *networkFlowManager) enrichConnections(tickerC <-chan time.Time) {
	defer m.stopper.Flow().ReportStopped()
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		case <-tickerC:
			if !features.SensorCapturesIntermediateEvents.Enabled() && !m.centralReady.IsDone() {
				log.Info("Sensor is in offline mode: skipping enriching until connection is back up")
				continue
			}
			m.enrichAndSend()
			// Measuring number of calls to `enrichAndSend` (ticks) for remembering historical endpoints
			m.clusterEntities.RecordTick()
		}
	}
}

func (m *networkFlowManager) getCurrentContext() context.Context {
	m.ctxMutex.Lock()
	defer m.ctxMutex.Unlock()
	return m.pipelineCtx
}

func (m *networkFlowManager) enrichAndSend() {
	// Takes host connections & endpoints and updates them by enriching with additional data.
	// Updates m.activeEndpoints and m.activeConnections if lastSeen was reported as null by the Collector.
	currentConns, currentEndpoints, currentProcesses := m.currentEnrichedConnsAndEndpoints()

	// Compares currently enriched connections & endpoints with those enriched in the previous cycle.
	// The new changes are sent to Central.
	updatedConns := computeUpdatedConns(currentConns, m.enrichedConnsLastSentState, &m.lastSentStateMutex)
	updatedEndpoints := computeUpdatedEndpoints(currentEndpoints, m.enrichedEndpointsLastSentState, &m.lastSentStateMutex)
	updatedProcesses := computeUpdatedProcesses(currentProcesses, m.enrichedProcessesLastSentState, &m.lastSentStateMutex)
	flowMetrics.NumUpdatesSentToCentral.WithLabelValues("connections").Add(float64(len(updatedConns)))
	flowMetrics.NumUpdatesSentToCentral.WithLabelValues("endpoints").Add(float64(len(updatedEndpoints)))
	flowMetrics.NumUpdatesSentToCentral.WithLabelValues("processes").Add(float64(len(updatedProcesses)))

	if len(updatedConns)+len(updatedEndpoints) > 0 {
		if sent := m.sendConnsEps(updatedConns, updatedEndpoints); sent {
			m.updateConnectionStates(currentConns, currentEndpoints)
		}
		metrics.SetNetworkFlowBufferSizeGauge(len(m.sensorUpdates))
	}
	if env.ProcessesListeningOnPort.BooleanSetting() && len(updatedProcesses) > 0 {
		if sent := m.sendProcesses(updatedProcesses); sent {
			m.updateProcessesState(currentProcesses)
		}
	}
}
func (m *networkFlowManager) sendConnsEps(conns []*storage.NetworkFlow, eps []*storage.NetworkEndpoint) bool {
	protoToSend := &central.NetworkFlowUpdate{
		Updated:          conns,
		UpdatedEndpoints: eps,
		Time:             protocompat.TimestampNow(),
	}

	var detectionContext context.Context
	if features.SensorCapturesIntermediateEvents.Enabled() {
		detectionContext = trace.Background()
	} else {
		detectionContext = m.getCurrentContext()
	}
	// Before sending, run the flows through policies asynchronously (ProcessNetworkFlow creates a new goroutine for each call)
	for _, flow := range conns {
		m.policyDetector.ProcessNetworkFlow(detectionContext, flow)
	}

	log.Debugf("Flow update : %v", protoToSend)
	return m.sendToCentral(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_NetworkFlowUpdate{
			NetworkFlowUpdate: protoToSend,
		},
	})
}

func (m *networkFlowManager) sendProcesses(processes []*storage.ProcessListeningOnPortFromSensor) bool {
	processesToSend := &central.ProcessListeningOnPortsUpdate{
		ProcessesListeningOnPorts: processes,
		Time:                      protocompat.TimestampNow(),
	}
	return m.sendToCentral(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ProcessListeningOnPortUpdate{
			ProcessListeningOnPortUpdate: processesToSend,
		},
	})
}

func (m *networkFlowManager) currentEnrichedConnsAndEndpoints() (
	enrichedConnections map[networkConnIndicator]timestamp.MicroTS,
	enrichedEndpoints map[containerEndpointIndicator]timestamp.MicroTS,
	enrichedProcesses map[processListeningIndicator]timestamp.MicroTS,
) {
	now := timestamp.Now()
	allHostConns := m.getAllHostConnections()

	enrichedConnections = make(map[networkConnIndicator]timestamp.MicroTS)
	enrichedEndpoints = make(map[containerEndpointIndicator]timestamp.MicroTS)
	enrichedProcesses = make(map[processListeningIndicator]timestamp.MicroTS)
	for _, hostConns := range allHostConns {
		m.enrichHostConnections(now, hostConns, enrichedConnections)
		m.enrichHostContainerEndpoints(now, hostConns, enrichedEndpoints, enrichedProcesses)
	}
	return enrichedConnections, enrichedEndpoints, enrichedProcesses
}

// isUpdated determines if a connection/endpoint should be sent to Central based on timestamp comparison.
//
// Timestamp Convention:
// - timestamp.InfiniteFuture = connection/endpoint is OPEN (still active)
// - Any other timestamp value = connection/endpoint is CLOSED (lastSeen/closeTime)
//
// This function detects updates when:
// 1. New connection/endpoint (not seen before)
// 2. More recent activity (newer timestamp)
// 3. State transition from OPEN -> CLOSED (InfiniteFuture -> actual timestamp)
func isUpdated(prevTS, currTS timestamp.MicroTS, seenPreviously bool) bool {
	// Connection has not been seen in the last tick.
	if !seenPreviously {
		return true
	}
	// Collector saw this connection more recently.
	if currTS > prevTS {
		return true
	}
	// Connection was active (unclosed) in the last tick, now it is closed.
	if prevTS == timestamp.InfiniteFuture && currTS != timestamp.InfiniteFuture {
		return true
	}
	return false
}

func computeUpdatedConns(current map[networkConnIndicator]timestamp.MicroTS, previous map[networkConnIndicator]timestamp.MicroTS, previousMutex *sync.RWMutex) []*storage.NetworkFlow {
	previousMutex.RLock()
	defer previousMutex.RUnlock()
	var updates []*storage.NetworkFlow

	for conn, currTS := range current {
		prevTS, seenPreviously := previous[conn]
		if isUpdated(prevTS, currTS, seenPreviously) {
			updates = append(updates, conn.toProto(currTS))
		}
	}

	for conn, prevTS := range previous {
		if _, ok := current[conn]; !ok {
			updates = append(updates, conn.toProto(prevTS))
		}
	}

	return updates
}

func computeUpdatedEndpoints(current map[containerEndpointIndicator]timestamp.MicroTS, previous map[containerEndpointIndicator]timestamp.MicroTS, previousMutex *sync.RWMutex) []*storage.NetworkEndpoint {
	previousMutex.RLock()
	defer previousMutex.RUnlock()
	var updates []*storage.NetworkEndpoint

	for ep, currTS := range current {
		prevTS, seenPreviously := previous[ep]
		if isUpdated(prevTS, currTS, seenPreviously) {
			updates = append(updates, ep.toProto(currTS))
		}
	}

	for ep, prevTS := range previous {
		if _, ok := current[ep]; !ok {
			updates = append(updates, ep.toProto(prevTS))
		}
	}

	return updates
}

func computeUpdatedProcesses(current map[processListeningIndicator]timestamp.MicroTS, previous map[processListeningIndicator]timestamp.MicroTS, previousMutex *sync.RWMutex) []*storage.ProcessListeningOnPortFromSensor {
	if !env.ProcessesListeningOnPort.BooleanSetting() {
		if len(current) > 0 {
			logging.GetRateLimitedLogger().Warn(loggingRateLimiter,
				"Received process while ProcessesListeningOnPort feature is disabled. This may indicate a misconfiguration.", len(current))
		}
		return []*storage.ProcessListeningOnPortFromSensor{}
	}
	previousMutex.RLock()
	defer previousMutex.RUnlock()
	var updates []*storage.ProcessListeningOnPortFromSensor

	for pl, currTS := range current {
		prevTS, seenPreviously := previous[pl]
		if isUpdated(prevTS, currTS, seenPreviously) {
			updates = append(updates, pl.toProto(currTS))
		}
	}

	for ep, prevTS := range previous {
		if _, ok := current[ep]; !ok {
			// This condition means the deployment was removed before we got the
			// close timestamp for the endpoint. Use the current timestamp instead.
			if prevTS == timestamp.InfiniteFuture {
				prevTS = timestamp.Now()
			}
			updates = append(updates, ep.toProto(prevTS))
		}
	}

	return updates
}

func (m *networkFlowManager) getAllHostConnections() []*hostConnections {
	// Get a snapshot of all *hostConnections. This allows us to lock the individual mutexes without having to hold
	// two locks simultaneously.
	// Using RLock here improves the runtime of this function by roughly 13% (benchmarked).
	m.connectionsByHostMutex.RLock()
	defer m.connectionsByHostMutex.RUnlock()

	allHostConns := make([]*hostConnections, 0, len(m.connectionsByHost))
	for _, hostConns := range m.connectionsByHost {
		allHostConns = append(allHostConns, hostConns)
	}
	return allHostConns
}

func (m *networkFlowManager) RegisterCollector(hostname string) (HostNetworkInfo, int64) {
	m.connectionsByHostMutex.Lock()
	defer m.connectionsByHostMutex.Unlock()

	conns := m.connectionsByHost[hostname]

	if conns == nil {
		conns = &hostConnections{
			hostname:    hostname,
			connections: make(map[connection]*connStatus),
			endpoints:   make(map[containerEndpoint]*connStatus),
		}
		m.connectionsByHost[hostname] = conns
	}

	concurrency.WithLock(&conns.mutex, func() {
		if conns.pendingDeletion != nil {
			// Note that we don't need to check the return value, since `deleteHostConnections` needs to acquire
			// m.connectionsByHostMutex. It can therefore only proceed once this function returns, in which case it will be
			// a no-op due to `pendingDeletion` being `nil`.
			conns.pendingDeletion.Stop()
			conns.pendingDeletion = nil
		}

		conns.currentSequenceID++
	})

	return conns, conns.currentSequenceID
}

func (m *networkFlowManager) deleteHostConnections(hostname string) {
	concurrency.WithLock(&m.connectionsByHostMutex, func() {
		conns := m.connectionsByHost[hostname]
		if conns == nil {
			return
		}
		concurrency.WithLock(&conns.mutex, func() {
			if conns.pendingDeletion == nil {
				return
			}
			flowMetrics.HostConnectionsOperations.WithLabelValues("remove", "connections").Add(float64(len(conns.connections)))
			flowMetrics.HostConnectionsOperations.WithLabelValues("remove", "endpoints").Add(float64(len(conns.endpoints)))
		})
		delete(m.connectionsByHost, hostname)
	})
}

func (m *networkFlowManager) UnregisterCollector(hostname string, sequenceID int64) {
	m.connectionsByHostMutex.Lock()
	defer m.connectionsByHostMutex.Unlock()

	conns := m.connectionsByHost[hostname]
	if conns == nil {
		return
	}
	conns.mutex.Lock()
	defer conns.mutex.Unlock()

	if conns.currentSequenceID != sequenceID {
		// Skip deletion if there has been a more recent `Register` call than the corresponding `Unregister` call
		return
	}
	if conns.pendingDeletion != nil {
		// Cancel any pending deletions there might be. See `RegisterCollector` on why we do not need to check for the
		// return value of Stop.
		conns.pendingDeletion.Stop()
	}
	conns.pendingDeletion = time.AfterFunc(connectionDeletionGracePeriod, func() {
		m.deleteHostConnections(hostname)
	})
}

func (h *hostConnections) Process(networkInfo *sensor.NetworkConnectionInfo, nowTimestamp timestamp.MicroTS, sequenceID int64) error {
	flowMetrics.NetworkConnectionInfoMessagesRcvd.With(prometheus.Labels{"Hostname": h.hostname}).Inc()

	updatedConnections := getUpdatedConnections(networkInfo)
	updatedEndpoints := getUpdatedContainerEndpoints(networkInfo)

	flowMetrics.NumUpdated.With(prometheus.Labels{"Hostname": h.hostname, "Type": "Connection"}).Set(float64(len(updatedConnections)))
	flowMetrics.NumUpdated.With(prometheus.Labels{"Hostname": h.hostname, "Type": "Endpoint"}).Set(float64(len(updatedEndpoints)))

	collectorTS := timestamp.FromProtobuf(networkInfo.GetTime())
	tsOffset := nowTimestamp - collectorTS

	if sequenceID != h.currentSequenceID {
		return errors.New("replaced by newer connection")
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if sequenceID != h.connectionsSequenceID {
		// This is the first message of the new connection.
		for _, status := range h.connections {
			// Mark all connections as closed this is the first update
			// after a connection went down and came back up again.
			status.lastSeen = h.lastKnownTimestamp
		}
		for _, status := range h.endpoints {
			status.lastSeen = h.lastKnownTimestamp
		}
		h.connectionsSequenceID = sequenceID
	}
	h.updateConnectionsStatusNoLock(updatedConnections, tsOffset, nowTimestamp)
	h.updateEndpointsStatusNoLock(updatedEndpoints, tsOffset, nowTimestamp)
	h.lastKnownTimestamp = nowTimestamp
	return nil
}

func (h *hostConnections) updateConnectionsStatusNoLock(updatedConnections map[connection]timestamp.MicroTS, tsOffset, nowTimestamp timestamp.MicroTS) {
	updateStatusNoLock(h.connections, updatedConnections, tsOffset, nowTimestamp)
}

func (h *hostConnections) updateEndpointsStatusNoLock(updatedEndpoints map[containerEndpoint]timestamp.MicroTS, tsOffset, nowTimestamp timestamp.MicroTS) {
	updateStatusNoLock(h.endpoints, updatedEndpoints, tsOffset, nowTimestamp)
}

func updateStatusNoLock[T comparable](current map[T]*connStatus, updated map[T]timestamp.MicroTS, tsOffset, nowTimestamp timestamp.MicroTS) {
	for c, t := range updated {
		// timestamp = zero implies the connection/endpoint is newly added.
		// Add new current, update existing ones to mark them closed
		if t != timestamp.InfiniteFuture { // adjust timestamp if not zero.
			t += tsOffset
		}
		status := current[c]
		if status == nil {
			status = &connStatus{
				firstSeen: nowTimestamp,
				tsAdded:   nowTimestamp,
			}
			if t < status.firstSeen {
				status.firstSeen = t
			}
			current[c] = status
		}
		status.lastSeen = t
	}
}

func getProcessKey(originator *storage.NetworkProcessUniqueKey) processInfo {
	if originator == nil {
		return processInfo{}
	}

	return processInfo{
		processName: originator.ProcessName,
		processArgs: originator.ProcessArgs,
		processExec: originator.ProcessExecFilePath,
	}
}

func getIPAndPort(address *sensor.NetworkAddress) net.NetworkPeerID {
	tuple := net.NetworkPeerID{
		// For private address, both address and IPNetwork are expected to be set by Collector.
		// If not set, this will be invalid i.e. `IPNetwork{}`.
		IPNetwork: net.IPNetworkFromCIDRBytes(address.GetIpNetwork()),
		// If not set, this will be invalid i.e. `IPAddress{}`.
		Address: net.IPFromBytes(address.GetAddressData()),
		Port:    uint16(address.GetPort()),
	}
	return tuple
}

func processConnection(conn *sensor.NetworkConnection) (*connection, error) {
	var incoming bool
	switch conn.Role {
	case sensor.ClientServerRole_ROLE_SERVER:
		incoming = true
	case sensor.ClientServerRole_ROLE_CLIENT:
		incoming = false
	default:
		return nil, errors.New("Received connection that was not marked as server or client")
	}

	remote := net.NumericEndpoint{
		IPAndPort: getIPAndPort(conn.GetRemoteAddress()),
		L4Proto:   net.L4ProtoFromProtobuf(conn.GetProtocol()),
	}
	local := getIPAndPort(conn.GetLocalAddress())

	// Special handling for UDP ports - role reported by collector may be unreliable, so look at which port is more
	// likely to be ephemeral. In case a port is set to 0, collector couldn't retrieve this value, we assume the
	// connection works in the direction opposite of this port.
	if remote.L4Proto == net.UDP {
		incoming = netutil.IsEphemeralPort(remote.IPAndPort.Port) > netutil.IsEphemeralPort(local.Port)
	}

	c := &connection{
		local:       local,
		remote:      remote,
		containerID: conn.GetContainerId(),
		incoming:    incoming,
	}
	return c, nil
}

// getUpdatedConnections returns a map of connections to timestamp.
// The timestamp set to +infinity means that the connection is open;
// any other value >0 means that the connection is closed.
func getUpdatedConnections(networkInfo *sensor.NetworkConnectionInfo) map[connection]timestamp.MicroTS {
	updatedConnections := make(map[connection]timestamp.MicroTS)

	for _, conn := range networkInfo.GetUpdatedConnections() {
		c, err := processConnection(conn)
		if err != nil {
			log.Warnf("Failed to process connection: %s", err)
			continue
		}

		// timestamp will be set to close timestamp for closed connections, and zero for newly added connection.
		ts := timestamp.FromProtobuf(conn.CloseTimestamp)
		if ts == 0 {
			ts = timestamp.InfiniteFuture
			flowMetrics.IncomingConnectionsEndpoints.With(prometheus.Labels{"object": "connections", "closedTS": "unset"}).Inc()
		} else {
			flowMetrics.IncomingConnectionsEndpoints.With(prometheus.Labels{"object": "connections", "closedTS": "set"}).Inc()
		}
		updatedConnections[*c] = ts
	}

	return updatedConnections
}

func getUpdatedContainerEndpoints(networkInfo *sensor.NetworkConnectionInfo) map[containerEndpoint]timestamp.MicroTS {
	updatedEndpoints := make(map[containerEndpoint]timestamp.MicroTS)

	for _, endpoint := range networkInfo.GetUpdatedEndpoints() {
		normalize.NetworkEndpoint(endpoint)
		ep := containerEndpoint{
			containerID: endpoint.GetContainerId(),
			endpoint: net.NumericEndpoint{
				IPAndPort: getIPAndPort(endpoint.GetListenAddress()),
				L4Proto:   net.L4ProtoFromProtobuf(endpoint.GetProtocol()),
			},
			processKey: getProcessKey(endpoint.GetOriginator()),
		}

		// timestamp will be set to close timestamp for closed connections, and zero for newly added connection.
		ts := timestamp.FromProtobuf(endpoint.GetCloseTimestamp())
		if ts == 0 {
			ts = timestamp.InfiniteFuture
			flowMetrics.IncomingConnectionsEndpoints.With(prometheus.Labels{"object": "endpoints", "closedTS": "unset"}).Inc()
		} else {
			flowMetrics.IncomingConnectionsEndpoints.With(prometheus.Labels{"object": "endpoints", "closedTS": "set"}).Inc()
		}
		updatedEndpoints[ep] = ts
	}

	return updatedEndpoints
}

func (m *networkFlowManager) PublicIPsValueStream() concurrency.ReadOnlyValueStream[*sensor.IPAddressList] {
	return m.publicIPs.PublicIPsProtoStream()
}

func (m *networkFlowManager) ExternalSrcsValueStream() concurrency.ReadOnlyValueStream[*sensor.IPNetworkList] {
	return m.externalSrcs.ExternalSrcsValueStream()
}
