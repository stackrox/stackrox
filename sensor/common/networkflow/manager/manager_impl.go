package manager

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
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
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/externalsrcs"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/metrics"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	// Wait at least this long before determining that an unresolvable IP is "outside of the cluster".
	clusterEntityResolutionWaitPeriod = 10 * time.Second
	// Wait at least this long before giving up on resolving the container for a connection
	maxContainerResolutionWaitPeriod = 2 * time.Minute

	connectionDeletionGracePeriod = 5 * time.Minute
)

var (
	emptyProcessInfo = processInfo{}
	tickerTime       = time.Second * 30
)

type hostConnections struct {
	hostname           string
	connections        map[connection]*connStatus
	endpoints          map[containerEndpoint]*connStatus
	lastKnownTimestamp timestamp.MicroTS

	// connectionsSequenceID is the sequence ID of the current active connections state
	connectionsSequenceID int64
	// currentSequenceID is the sequence ID of the most recent `Register` call
	currentSequenceID int64

	pendingDeletion *time.Timer

	mutex sync.Mutex
}

type connStatus struct {
	firstSeen timestamp.MicroTS
	lastSeen  timestamp.MicroTS
	// used keeps track of if an endpoint has been used by the networkgraph path.
	used bool
	// usedProcess keeps track of if an endpoint has been used by the processes listening on
	// ports path. If processes listening on ports is used, both must be true to delete the
	// endpoint. Otherwise the endpoint will not be available to process listening on ports
	// and it won't report endpoints that it doesn't have access to.
	usedProcess bool
	// rotten implies we expected to correlate the flow with a container, but were unable to
	rotten bool
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

// connection is an instance of a connection as reported by collector
type connection struct {
	local       net.NetworkPeerID
	remote      net.NumericEndpoint
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

type containerEndpoint struct {
	endpoint    net.NumericEndpoint
	containerID string
	processKey  processInfo
}

func (e *containerEndpoint) String() string {
	return fmt.Sprintf("%s %s: %s", e.containerID, e.processKey, e.endpoint)
}

type Option func(*networkFlowManager)

// WithTicker overrides the default ticker
func WithTicker(ticker <-chan time.Time) Option {
	return func(manager *networkFlowManager) {
		if ticker != nil {
			manager.tickerC = ticker
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
	enricherTicker := time.NewTicker(tickerTime)
	mgr := &networkFlowManager{
		connectionsByHost: make(map[string]*hostConnections),
		clusterEntities:   clusterEntities,
		publicIPs:         newPublicIPsManager(),
		externalSrcs:      externalSrcs,
		policyDetector:    policyDetector,
		enricherTicker:    enricherTicker,
		tickerC:           enricherTicker.C,
		initialSync:       &atomic.Bool{},
		activeConnections: make(map[connection]*networkConnIndicator),
		activeEndpoints:   make(map[containerEndpoint]*containerEndpointIndicator),
		stopper:           concurrency.NewStopper(),
		pubSub:            pubSub,
	}

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

type networkFlowManager struct {
	connectionsByHost      map[string]*hostConnections
	connectionsByHostMutex sync.Mutex

	clusterEntities EntityStore
	externalSrcs    externalsrcs.Store

	lastSentStateMutex             sync.RWMutex
	enrichedConnsLastSentState     map[networkConnIndicator]timestamp.MicroTS
	enrichedEndpointsLastSentState map[containerEndpointIndicator]timestamp.MicroTS
	enrichedProcessesLastSentState map[processListeningIndicator]timestamp.MicroTS

	activeConnections map[connection]*networkConnIndicator
	activeEndpoints   map[containerEndpoint]*containerEndpointIndicator

	sensorUpdates chan *message.ExpiringMessage
	centralReady  concurrency.Signal

	ctxMutex    sync.Mutex
	cancelCtx   context.CancelFunc
	pipelineCtx context.Context
	initialSync *atomic.Bool

	enricherTicker *time.Ticker
	tickerC        <-chan time.Time

	publicIPs *publicIPsManager

	policyDetector detector.Detector

	stopper concurrency.Stopper
	pubSub  *internalmessage.MessageSubscriber
}

func (m *networkFlowManager) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (m *networkFlowManager) Start() error {
	go m.enrichConnections(m.tickerC)
	go m.publicIPs.Run(m.stopper.LowLevel().GetStopRequestSignal(), m.clusterEntities)
	return nil
}

func (m *networkFlowManager) Stop(_ error) {
	if !m.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = m.stopper.Client().Stopped().Wait()
		}()
	}
	m.stopper.Client().Stop()
}

func (m *networkFlowManager) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (m *networkFlowManager) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventResourceSyncFinished:
		if features.SensorCapturesIntermediateEvents.Enabled() {
			if m.initialSync.CompareAndSwap(false, true) {
				m.enricherTicker.Reset(tickerTime)
			}
			return
		}
		m.resetContext()
		m.resetLastSentState()
		m.centralReady.Signal()
		m.enricherTicker.Reset(tickerTime)
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
	m.pipelineCtx, m.cancelCtx = context.WithCancel(context.Background())
}

func (m *networkFlowManager) sendToCentral(msg *central.MsgFromSensor) bool {
	if features.SensorCapturesIntermediateEvents.Enabled() {
		select {
		case <-m.stopper.Flow().StopRequested():
			return false
		case m.sensorUpdates <- message.New(msg):
			log.Infof("=============== lvm message sent to central flows len(%d) endpoints len(%d)", len(msg.GetNetworkFlowUpdate().GetUpdated()), len(msg.GetNetworkFlowUpdate().GetUpdatedEndpoints()))
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
			log.Info("=== lvm manager tick")
			if !features.SensorCapturesIntermediateEvents.Enabled() && !m.centralReady.IsDone() {
				log.Info("Sensor is in offline mode: skipping enriching until connection is back up")
				continue
			}
			m.enrichAndSend()
			// Measuring number of calls to `enrichAndSend` (ticks) for remembering historical endpoints
			m.clusterEntities.RecordTick()
			if env.ProcessesListeningOnPort.BooleanSetting() {
				m.enrichAndSendProcesses()
			}
		}
	}
}

func (m *networkFlowManager) getCurrentContext() context.Context {
	m.ctxMutex.Lock()
	defer m.ctxMutex.Unlock()
	return m.pipelineCtx
}

func (m *networkFlowManager) enrichAndSend() {
	log.Info("=== lvm (enrichAndSend) start")
	currentConns, currentEndpoints := m.currentEnrichedConnsAndEndpoints()

	// DEBUG OUTPUT
	log.Infof("=== lvm (enrichAndSend) current conn len(%d) current endpoints len(%d)", len(currentConns), len(currentEndpoints))
	for indicator, ts := range currentConns {
		log.Infof("=== lvm (enrichAndSend) current conn %s: %d",
			indicator.srcEntity.ID+"->"+indicator.dstEntity.ID, ts.UnixSeconds())
	}
	for indicator, ts := range currentEndpoints {
		log.Infof("=== lvm (enrichAndSend) current endpoint %s: %d",
			indicator.entity.ID, ts.UnixSeconds())
	}

	updatedConns := computeUpdatedConns(currentConns, m.enrichedConnsLastSentState, &m.lastSentStateMutex)
	updatedEndpoints := computeUpdatedEndpoints(currentEndpoints, m.enrichedEndpointsLastSentState, &m.lastSentStateMutex)

	log.Infof("=== lvm updated conn len(%d) updated endpoints len(%d)", len(updatedConns), len(updatedEndpoints))

	// DEBUG OUTPUT
	for _, c := range updatedConns {
		log.Infof("====== lvm (enrichAndSend) updated conn %s -> %s",
			c.GetProps().GetSrcEntity().GetId(), c.GetProps().GetDstEntity().GetId())
	}
	for _, e := range updatedEndpoints {
		log.Infof("====== lvm (enrichAndSend) updated endpoint %s, ts=%s",
			e.GetProps().String(), e.GetLastActiveTimestamp())
	}

	if len(updatedConns)+len(updatedEndpoints) == 0 {
		defer log.Infof("=== lvm (enrichAndSend) exiting early - nothing to update")
		return
	}

	protoToSend := &central.NetworkFlowUpdate{
		Updated:          updatedConns,
		UpdatedEndpoints: updatedEndpoints,
		Time:             protocompat.TimestampNow(),
	}

	var detectionContext context.Context
	if features.SensorCapturesIntermediateEvents.Enabled() {
		detectionContext = context.Background()
	} else {
		detectionContext = m.getCurrentContext()
	}
	// Before sending, run the flows through policies asynchronously (ProcessNetworkFlow creates a new goroutine for each call)
	for _, flow := range updatedConns {
		m.policyDetector.ProcessNetworkFlow(detectionContext, flow)
	}

	log.Debugf("Flow update : %v", protoToSend)
	if m.sendToCentral(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_NetworkFlowUpdate{
			NetworkFlowUpdate: protoToSend,
		},
	}) {
		m.updateConnectionStates(currentConns, currentEndpoints)
		metrics.IncrementTotalNetworkFlowsSentCounter(len(protoToSend.Updated))
		metrics.IncrementTotalNetworkEndpointsSentCounter(len(protoToSend.UpdatedEndpoints))
	}
	metrics.SetNetworkFlowBufferSizeGauge(len(m.sensorUpdates))
}

func (m *networkFlowManager) enrichAndSendProcesses() {
	currentProcesses := m.currentEnrichedProcesses()

	updatedProcesses := computeUpdatedProcesses(currentProcesses, m.enrichedProcessesLastSentState, &m.lastSentStateMutex)

	if len(updatedProcesses) == 0 {
		return
	}

	processesToSend := &central.ProcessListeningOnPortsUpdate{
		ProcessesListeningOnPorts: updatedProcesses,
		Time:                      protocompat.TimestampNow(),
	}

	if m.sendToCentral(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ProcessListeningOnPortUpdate{
			ProcessListeningOnPortUpdate: processesToSend,
		},
	}) {
		m.updateProcessesState(currentProcesses)
		metrics.IncrementTotalProcessesSentCounter(len(processesToSend.ProcessesListeningOnPorts))
	}
}

func (m *networkFlowManager) handleContainerNotFound(conn *connection, status *connStatus, enrichedConnections map[networkConnIndicator]timestamp.MicroTS) error {
	timeElapsedSinceFirstSeen := timestamp.Now().ElapsedSince(status.firstSeen)
	failReason := fmt.Errorf("ContainerID %s unknown", conn.containerID)
	if timeElapsedSinceFirstSeen <= maxContainerResolutionWaitPeriod {
		return multierror.Append(failReason, fmt.Errorf("time for container resolution (%s) not elapsed yet", maxContainerResolutionWaitPeriod))
	}

	activeConn, found := m.activeConnections[*conn]
	if !found {
		// Expire the connection if the container cannot be found within the clusterEntityResolutionWaitPeriod
		status.rotten = true
		// Only increment metric once the connection is marked rotten
		flowMetrics.ContainerIDMisses.Inc()
		log.Debugf("Can't find deployment information for container %s", conn.containerID)
		return failReason
	}
	// Active connection found - enrichment can be done.
	enrichedConnections[*activeConn] = timestamp.Now()
	delete(m.activeConnections, *conn)
	flowMetrics.SetActiveFlowsTotalGauge(len(m.activeConnections))
	return nil
}

func formatMultiErrorOneline(errs []error) string {
	elems := make([]string, len(errs))
	for i, err := range errs {
		// The error is used in debug logs and is much nicer to read with the first letter capitalized.
		// Writing the error message with first capital in the code raises a style warning.
		msg := cases.Title(language.English, cases.NoLower).String(err.Error())
		elems[i] = fmt.Sprintf("(%d) %s", i+1, msg)
	}
	return strings.Join(elems, ", ")
}

func logReasonForAggregatingNetGraphFlow(conn *connection, contNs, contName, entitiesName string, port uint16, failReason *multierror.Error) {
	reasonStr := ""
	if failReason != nil {
		failReason.ErrorFormat = formatMultiErrorOneline
		reasonStr = failReason.Error()
	}
	// No need to produce complex chain of reasons, if there is one simple explanation
	if conn.remote.IsConsideredExternal() {
		reasonStr = "Collector did not report the IP address to Sensor - the remote part is the Internet"
	}
	if conn.incoming {
		// Keep internal wording even if central lacks `NetworkGraphInternalEntitiesSupported` capability.
		log.Debugf("Marking incoming connection to container %s/%s from %s:%s as '%s' in the network graph: %s.",
			contNs, contName, conn.remote.IPAndPort.String(),
			strconv.Itoa(int(port)), entitiesName, reasonStr)
	} else {
		log.Debugf("Marking outgoing connection from container %s/%s to %s as '%s' in the network graph: %s.",
			contNs, contName, conn.remote.IPAndPort.String(),
			entitiesName, reasonStr)
	}
}

func (m *networkFlowManager) enrichConnection(conn *connection, status *connStatus, enrichedConnections map[networkConnIndicator]timestamp.MicroTS) {
	timeElapsedSinceFirstSeen := timestamp.Now().ElapsedSince(status.firstSeen)
	isFresh := timeElapsedSinceFirstSeen < clusterEntityResolutionWaitPeriod
	var netGraphFailReason *multierror.Error

	container, ok := m.clusterEntities.LookupByContainerID(conn.containerID)
	if !ok {
		// There is an incoming connection to a container that Sensor does not recognize.
		// 90% of the cases that container is Sensor itself before being restarted.
		if err := m.handleContainerNotFound(conn, status, enrichedConnections); err != nil {
			log.Debugf("Enrichment failed: %v", err)
		}
		return
	}
	netGraphFailReason = multierror.Append(netGraphFailReason, errors.New("ContainerID lookup successful"))

	var lookupResults []clusterentities.LookupResult
	var isInternet = false

	// Check if the remote address represents the de-facto INTERNET entity.
	if conn.remote.IsConsideredExternal() {
		isFresh = false
		isInternet = true
		netGraphFailReason = multierror.Append(netGraphFailReason,
			errors.New("Remote part of the connection is the Internet"))
	} else {
		// Otherwise, check if the remote entity is actually a cluster entity.
		lookupResults = m.clusterEntities.LookupByEndpoint(conn.remote)
	}

	var port uint16
	var direction string
	if conn.incoming {
		direction = "ingress"
		port = conn.local.Port
	} else {
		direction = "egress"
		port = conn.remote.IPAndPort.Port
	}

	metricDirection := prometheus.Labels{
		"direction": direction,
		"namespace": container.Namespace,
	}

	if len(lookupResults) == 0 {
		netGraphFailReason = multierror.Append(netGraphFailReason,
			fmt.Errorf("lookup in clusterEntitiesStore failed for endpoint %s", conn.remote.String()))
		// If the address is set and is not resolvable, we want to we wait for `clusterEntityResolutionWaitPeriod` time
		// before associating it to a known network or INTERNET.
		if isFresh && conn.remote.IPAndPort.Address.IsValid() {
			return
		}

		extSrc := m.externalSrcs.LookupByNetwork(conn.remote.IPAndPort.IPNetwork)
		if extSrc != nil {
			isFresh = false
		}

		if isFresh {
			log.Debugf("Enrichment aborted: Connection is fresh")
			return
		}

		defer func() {
			status.used = true
		}()

		if extSrc == nil {
			netGraphFailReason = multierror.Append(netGraphFailReason,
				fmt.Errorf("lookup by network in externalSrcsStore failed for network %+v", conn.remote.IPAndPort))
			entityType := networkgraph.InternetEntity()
			isExternal, err := conn.IsExternal()
			if err != nil {
				// IP is malformed or unknown - do not show on the graph and log the info
				// TODO(ROX-22388): Change log level back to warning when potential Collector issue is fixed
				log.Debugf("Enrichment aborted: Not showing flow on the network graph: %v", err)
				return
			}
			if isExternal {
				// If Central does not handle DiscoveredExternalEntities, report an Internet entity as it used to be.
				if !isInternet && centralcaps.Has(centralsensor.NetworkGraphDiscoveredExternalEntitiesSupported) {
					entityType = networkgraph.DiscoveredExternalEntity(net.IPNetworkFromNetworkPeerID(conn.remote.IPAndPort))
				} else {
					netGraphFailReason = multierror.Append(netGraphFailReason,
						errors.New("Central lacks capability to display discovered external entities"))
				}
			} else if centralcaps.Has(centralsensor.NetworkGraphInternalEntitiesSupported) {
				// Central without the capability would crash the UI if we make it display "Internal Entities".
				entityType = networkgraph.InternalEntities()
			} else {
				netGraphFailReason = multierror.Append(netGraphFailReason,
					errors.New("Central lacks capability to display 'Internal Entities' in the UI"))
			}

			// Fake a lookup result. This shows "External Entities" or "Internal Entities" in the network graph
			lookupResults = []clusterentities.LookupResult{
				{
					Entity:         entityType,
					ContainerPorts: []uint16{port},
				},
			}
			entitiesName := "Internal Entities"
			if isExternal {
				entitiesName = "External Entities"
			}
			logReasonForAggregatingNetGraphFlow(conn, container.Namespace, container.ContainerName, entitiesName, port, netGraphFailReason)

			if !status.used {
				// Count internal metrics even if central lacks `NetworkGraphInternalEntitiesSupported` capability.
				if isExternal {
					flowMetrics.ExternalFlowCounter.With(metricDirection).Inc()
				} else {
					flowMetrics.InternalFlowCounter.With(metricDirection).Inc()
				}
			}
		} else {
			if !status.used {
				flowMetrics.NetworkEntityFlowCounter.With(metricDirection).Inc()
			}
			lookupResults = []clusterentities.LookupResult{
				{
					Entity:         networkgraph.EntityFromProto(extSrc),
					ContainerPorts: []uint16{port},
				},
			}
		}
	} else {
		if !status.used {
			flowMetrics.NetworkEntityFlowCounter.With(metricDirection).Inc()
		}
		status.used = true
		if conn.incoming {
			// Only report incoming connections from outside the cluster. These are already taken care of by the
			// corresponding outgoing connection from the other end.
			return
		}
	}

	for _, lookupResult := range lookupResults {
		for _, port := range lookupResult.ContainerPorts {
			indicator := networkConnIndicator{
				dstPort:  port,
				protocol: conn.remote.L4Proto.ToProtobuf(),
			}

			if conn.incoming {
				indicator.srcEntity = lookupResult.Entity
				indicator.dstEntity = networkgraph.EntityForDeployment(container.DeploymentID)
			} else {
				indicator.srcEntity = networkgraph.EntityForDeployment(container.DeploymentID)
				indicator.dstEntity = lookupResult.Entity
			}

			// Multiple connections from a collector can result in a single enriched connection
			// hence update the timestamp only if we have a more recent connection than the one we have already enriched.
			if oldTS, found := enrichedConnections[indicator]; !found || oldTS < status.lastSeen {
				enrichedConnections[indicator] = status.lastSeen
				if features.SensorCapturesIntermediateEvents.Enabled() {
					if status.lastSeen == timestamp.InfiniteFuture {
						m.activeConnections[*conn] = &indicator
						flowMetrics.SetActiveFlowsTotalGauge(len(m.activeConnections))
					} else {
						delete(m.activeConnections, *conn)
						flowMetrics.SetActiveFlowsTotalGauge(len(m.activeConnections))
					}
				}
			}
		}
	}
}

func (m *networkFlowManager) enrichContainerEndpoint(ep *containerEndpoint, status *connStatus, enrichedEndpoints map[containerEndpointIndicator]timestamp.MicroTS) {
	timeElapsedSinceFirstSeen := timestamp.Now().ElapsedSince(status.firstSeen)
	isFresh := timeElapsedSinceFirstSeen < clusterEntityResolutionWaitPeriod
	if !isFresh {
		status.used = true
	}

	container, ok := m.clusterEntities.LookupByContainerID(ep.containerID)
	if !ok {
		// Expire the connection if the container cannot be found within the clusterEntityResolutionWaitPeriod
		if timeElapsedSinceFirstSeen > maxContainerResolutionWaitPeriod {
			if activeEp, found := m.activeEndpoints[*ep]; found {
				enrichedEndpoints[*activeEp] = timestamp.Now()
				delete(m.activeEndpoints, *ep)
				flowMetrics.SetActiveEndpointsTotalGauge(len(m.activeEndpoints))
				return
			}
			status.rotten = true
			// Only increment metric once the connection is marked rotten
			flowMetrics.ContainerIDMisses.Inc()
			log.Debugf("Unable to fetch deployment information for container %s: no deployment found", ep.containerID)
		}
		return
	}

	status.used = true

	indicator := containerEndpointIndicator{
		entity:   networkgraph.EntityForDeployment(container.DeploymentID),
		port:     ep.endpoint.IPAndPort.Port,
		protocol: ep.endpoint.L4Proto.ToProtobuf(),
	}

	// Multiple endpoints from a collector can result in a single enriched endpoint,
	// hence update the timestamp only if we have a more recent endpoint than the one we have already enriched.
	if oldTS, found := enrichedEndpoints[indicator]; !found || oldTS < status.lastSeen {
		enrichedEndpoints[indicator] = status.lastSeen
		if features.SensorCapturesIntermediateEvents.Enabled() {
			if status.lastSeen == timestamp.InfiniteFuture {
				m.activeEndpoints[*ep] = &indicator
				flowMetrics.SetActiveEndpointsTotalGauge(len(m.activeEndpoints))
			} else {
				delete(m.activeEndpoints, *ep)
				flowMetrics.SetActiveEndpointsTotalGauge(len(m.activeEndpoints))
			}
		}
	}
}

func (m *networkFlowManager) enrichProcessListening(ep *containerEndpoint, status *connStatus, processesListening map[processListeningIndicator]timestamp.MicroTS) {
	timeElapsedSinceFirstSeen := timestamp.Now().ElapsedSince(status.firstSeen)
	isFresh := timeElapsedSinceFirstSeen < clusterEntityResolutionWaitPeriod
	if !isFresh {
		status.usedProcess = true
	}

	container, ok := m.clusterEntities.LookupByContainerID(ep.containerID)
	if !ok {
		// Expire the process if the container cannot be found within the clusterEntityResolutionWaitPeriod
		if timeElapsedSinceFirstSeen > maxContainerResolutionWaitPeriod {
			status.rotten = true
			// Only increment metric once the connection is marked rotten
			flowMetrics.ContainerIDMisses.Inc()
			log.Debugf("Unable to fetch deployment information for container %s: no deployment found", ep.containerID)
		}
		return
	}

	status.usedProcess = true

	indicator := processListeningIndicator{
		key: processUniqueKey{
			podID:         container.PodID,
			containerName: container.ContainerName,
			deploymentID:  container.DeploymentID,
			process:       ep.processKey,
		},
		port:      ep.endpoint.IPAndPort.Port,
		protocol:  ep.endpoint.L4Proto.ToProtobuf(),
		podUID:    container.PodUID,
		namespace: container.Namespace,
	}

	processesListening[indicator] = status.lastSeen
}

func (m *networkFlowManager) enrichHostConnections(hostConns *hostConnections, enrichedConnections map[networkConnIndicator]timestamp.MicroTS) {
	hostConns.mutex.Lock()
	defer hostConns.mutex.Unlock()

	prevSize := len(hostConns.connections)
	for conn, status := range hostConns.connections {
		m.enrichConnection(&conn, status, enrichedConnections)
		if status.rotten || (status.used && status.lastSeen != timestamp.InfiniteFuture) {
			// connections that are no longer active and have already been used can be deleted.
			delete(hostConns.connections, conn)
		}
	}
	flowMetrics.HostConnectionsRemoved.Add(float64(prevSize - len(hostConns.connections)))
}

func (m *networkFlowManager) enrichHostContainerEndpoints(hostConns *hostConnections, enrichedEndpoints map[containerEndpointIndicator]timestamp.MicroTS) {
	hostConns.mutex.Lock()
	defer hostConns.mutex.Unlock()

	prevSize := len(hostConns.endpoints)
	for ep, status := range hostConns.endpoints {
		m.enrichContainerEndpoint(&ep, status, enrichedEndpoints)
		// If processes listening on ports is enabled, it has to be used there as well before being deleted.
		used := status.used && (status.usedProcess || !env.ProcessesListeningOnPort.BooleanSetting())
		if status.rotten || (used && status.lastSeen != timestamp.InfiniteFuture) {
			// endpoints that are no longer active and have already been used can be deleted.
			delete(hostConns.endpoints, ep)
		}
	}
	flowMetrics.HostEndpointsRemoved.Add(float64(prevSize - len(hostConns.endpoints)))
}

func (m *networkFlowManager) enrichProcessesListening(hostConns *hostConnections, processesListening map[processListeningIndicator]timestamp.MicroTS) {
	hostConns.mutex.Lock()
	defer hostConns.mutex.Unlock()

	prevSize := len(hostConns.endpoints)
	for ep, status := range hostConns.endpoints {
		if ep.processKey == emptyProcessInfo {
			// No way to update a process if the data isn't there
			continue
		}

		m.enrichProcessListening(&ep, status, processesListening)
		if status.rotten || (status.used && status.usedProcess && status.lastSeen != timestamp.InfiniteFuture) {
			// endpoints that are no longer active and have already been used can be deleted.
			// Before deleting it must be used here and in enrichContainerEndpoints.
			delete(hostConns.endpoints, ep)
		}
	}
	flowMetrics.HostProcessesRemoved.Add(float64(prevSize - len(hostConns.endpoints)))
}

func (m *networkFlowManager) currentEnrichedConnsAndEndpoints() (map[networkConnIndicator]timestamp.MicroTS, map[containerEndpointIndicator]timestamp.MicroTS) {
	allHostConns := m.getAllHostConnections()

	enrichedConnections := make(map[networkConnIndicator]timestamp.MicroTS)
	enrichedEndpoints := make(map[containerEndpointIndicator]timestamp.MicroTS)
	for _, hostConns := range allHostConns {
		m.enrichHostConnections(hostConns, enrichedConnections)
		m.enrichHostContainerEndpoints(hostConns, enrichedEndpoints)
	}

	return enrichedConnections, enrichedEndpoints
}

func (m *networkFlowManager) currentEnrichedProcesses() map[processListeningIndicator]timestamp.MicroTS {
	allHostConns := m.getAllHostConnections()

	enrichedProcesses := make(map[processListeningIndicator]timestamp.MicroTS)
	for _, hostConns := range allHostConns {
		m.enrichProcessesListening(hostConns, enrichedProcesses)
	}

	return enrichedProcesses
}

func computeUpdatedConns(current map[networkConnIndicator]timestamp.MicroTS, previous map[networkConnIndicator]timestamp.MicroTS, previousMutex *sync.RWMutex) []*storage.NetworkFlow {
	previousMutex.RLock()
	defer previousMutex.RUnlock()
	var updates []*storage.NetworkFlow

	for conn, currTS := range current {
		prevTS, ok := previous[conn]
		if !ok || currTS > prevTS {
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
		prevTS, ok := previous[ep]
		if !ok || currTS > prevTS {
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
	previousMutex.RLock()
	defer previousMutex.RUnlock()
	var updates []*storage.ProcessListeningOnPortFromSensor

	for pl, currTS := range current {
		prevTS, ok := previous[pl]
		if !ok || currTS > prevTS || (prevTS == timestamp.InfiniteFuture && currTS != timestamp.InfiniteFuture) {
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
	m.connectionsByHostMutex.Lock()
	defer m.connectionsByHostMutex.Unlock()

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

	conns.mutex.Lock()
	defer conns.mutex.Unlock()

	if conns.pendingDeletion != nil {
		// Note that we don't need to check the return value, since `deleteHostConnections` needs to acquire
		// m.connectionsByHostMutex. It can therefore only proceed once this function returns, in which case it will be
		// a no-op due to `pendingDeletion` being `nil`.
		conns.pendingDeletion.Stop()
		conns.pendingDeletion = nil
	}

	conns.currentSequenceID++

	return conns, conns.currentSequenceID
}

func (m *networkFlowManager) deleteHostConnections(hostname string) {
	m.connectionsByHostMutex.Lock()
	defer m.connectionsByHostMutex.Unlock()

	conns := m.connectionsByHost[hostname]
	if conns == nil {
		return
	}

	conns.mutex.Lock()
	defer conns.mutex.Unlock()

	if conns.pendingDeletion == nil {
		return
	}
	flowMetrics.HostConnectionsRemoved.Add(float64(len(conns.connections)))
	delete(m.connectionsByHost, hostname)
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
		// Skip deletion if there has been a more recent Register call than the corresponding Unregister call
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
	updatedConnections := getUpdatedConnections(h.hostname, networkInfo)
	updatedEndpoints := getUpdatedContainerEndpoints(h.hostname, networkInfo)
	for c, ts := range updatedConnections {
		if ts == timestamp.InfiniteFuture {
			log.Infof("=== lvm (Process) collector-updated conn %s: +Inf (%d)", c.String(), ts.UnixSeconds())
		} else {
			log.Infof("=== lvm (Process) collector-updated conn %s: %d", c.String(), ts.UnixSeconds())
		}
	}
	for c, ts := range updatedEndpoints {
		if ts == timestamp.InfiniteFuture {
			log.Infof("=== lvm (Process) collector-updated endpoint %s: +Inf (%d)", c.String(), ts.UnixSeconds())
		} else {
			log.Infof("=== lvm (Process) collector-updated endpoint %s: %d", c.String(), ts.UnixSeconds())
		}
	}
	collectorTS := timestamp.FromProtobuf(networkInfo.GetTime())
	tsOffset := nowTimestamp - collectorTS

	h.mutex.Lock()
	defer h.mutex.Unlock()

	if sequenceID != h.currentSequenceID {
		log.Infof("=== lvm (Process) conn replaced by newer connection")
		return errors.New("replaced by newer connection")
	} else if sequenceID != h.connectionsSequenceID {
		log.Infof("=== lvm (Process) observing first message of a new connection")
		// This is the first message of the new connection.
		for _, status := range h.connections {
			// Mark all past connections as closed, as this is the first update
			// after a connection went down and came back up again.
			status.lastSeen = h.lastKnownTimestamp
		}
		for _, status := range h.endpoints {
			status.lastSeen = h.lastKnownTimestamp
		}
		h.connectionsSequenceID = sequenceID
	}

	{
		prevSize := len(h.connections)
		for c, t := range updatedConnections {
			// timestamp = zero implies the connection is newly added. Add new connections, update existing ones to mark them closed
			if t != timestamp.InfiniteFuture { // adjust timestamp if not zero.
				t += tsOffset
			}
			status := h.connections[c]
			if status == nil {
				status = &connStatus{
					firstSeen: timestamp.Now(),
				}
				if t < status.firstSeen {
					status.firstSeen = t
				}
				h.connections[c] = status
			}
			status.lastSeen = t
			log.Infof("=============== lvm conn last seen is set %v", t)
		}

		h.lastKnownTimestamp = nowTimestamp
		flowMetrics.HostConnectionsAdded.Add(float64(len(h.connections) - prevSize))
	}

	{
		prevSize := len(h.endpoints)
		for ep, t := range updatedEndpoints {
			// timestamp = zero implies the endpoint is newly added. Add new endpoints, update existing ones to mark them closed
			if t != timestamp.InfiniteFuture { // adjust timestamp if not zero.
				t += tsOffset
			}
			status := h.endpoints[ep]
			if status == nil {
				status = &connStatus{
					firstSeen: timestamp.Now(),
				}
				if t < status.firstSeen {
					status.firstSeen = t
				}
				h.endpoints[ep] = status
			}
			status.lastSeen = t
			log.Infof("=============== lvm endpoint last seen is set %v", t)
		}

		h.lastKnownTimestamp = nowTimestamp
		flowMetrics.HostEndpointsAdded.Add(float64(len(h.endpoints) - prevSize))
	}

	return nil
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
func getUpdatedConnections(hostname string, networkInfo *sensor.NetworkConnectionInfo) map[connection]timestamp.MicroTS {
	updatedConnections := make(map[connection]timestamp.MicroTS)

	flowMetrics.NetworkFlowMessagesPerNode.With(prometheus.Labels{"Hostname": hostname}).Inc()

	for _, conn := range networkInfo.GetUpdatedConnections() {
		c, err := processConnection(conn)
		if err != nil {
			log.Warnf("Failed to process connection: %s", err)
			continue
		}

		connType := "outgoing"
		if c.incoming {
			connType = "incoming"
		}

		flowMetrics.NetworkFlowsPerNodeByType.With(prometheus.Labels{"Hostname": hostname, "Type": connType, "Protocol": conn.Protocol.String()}).Inc()

		// timestamp will be set to close timestamp for closed connections, and zero for newly added connection.
		ts := timestamp.FromProtobuf(conn.CloseTimestamp)
		if ts == 0 {
			ts = timestamp.InfiniteFuture
		}
		updatedConnections[*c] = ts
	}

	return updatedConnections
}

func getUpdatedContainerEndpoints(hostname string, networkInfo *sensor.NetworkConnectionInfo) map[containerEndpoint]timestamp.MicroTS {
	updatedEndpoints := make(map[containerEndpoint]timestamp.MicroTS)

	flowMetrics.NetworkFlowMessagesPerNode.With(prometheus.Labels{"Hostname": hostname}).Inc()

	for _, endpoint := range networkInfo.GetUpdatedEndpoints() {
		normalize.NetworkEndpoint(endpoint)

		flowMetrics.ContainerEndpointsPerNode.With(prometheus.Labels{"Hostname": hostname, "Protocol": endpoint.Protocol.String()}).Inc()

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
