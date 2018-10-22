package manager

import (
	"errors"
	"sync"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/metrics"
)

type hostConnections struct {
	connections        map[connection]timestamp.MicroTS
	lastKnownTimestamp timestamp.MicroTS
	sequenceID         int64

	mutex sync.Mutex
}

type networkConnIndicator struct {
	srcDeploymentID string
	dstDeploymentID string
	dstPort         uint16
	protocol        v1.L4Protocol
}

func (i networkConnIndicator) toProto(ts timestamp.MicroTS) *v1.NetworkFlow {
	proto := &v1.NetworkFlow{
		Props: &v1.NetworkFlowProperties{
			SrcDeploymentId: i.srcDeploymentID,
			DstDeploymentId: i.dstDeploymentID,
			DstPort:         uint32(i.dstPort),
			L4Protocol:      i.protocol,
		},
	}

	if ts != timestamp.InfiniteFuture {
		proto.LastSeenTimestamp = ts.GogoProtobuf()
	}
	return proto
}

// connection is an instance of a connection as reported by collector
type connection struct {
	srcAddr        net.IPAddress
	srcContainerID string
	dest           net.NumericEndpoint
}

type networkFlowManager struct {
	connectionsByHost      map[string]*hostConnections
	connectionsByHostMutex sync.Mutex

	clusterEntities *clusterentities.Store

	enrichedLastSentState map[networkConnIndicator]timestamp.MicroTS

	done        concurrency.Signal
	flowUpdates chan *central.NetworkFlowUpdate
}

func (m *networkFlowManager) Start() {
	go m.enrichConnections()
}

func (m *networkFlowManager) Stop() {
	m.done.Signal()
}

func (m *networkFlowManager) FlowUpdates() <-chan *central.NetworkFlowUpdate {
	return m.flowUpdates
}

func (m *networkFlowManager) enrichConnections() {
	ticker := time.NewTicker(time.Second * 30)

	for {
		select {
		case <-m.done.WaitC():
			return
		case <-ticker.C:
			m.enrichAndSend()
		}
	}
}

func computeUpdateMessage(current map[networkConnIndicator]timestamp.MicroTS, previous map[networkConnIndicator]timestamp.MicroTS) *central.NetworkFlowUpdate {
	var updates []*v1.NetworkFlow

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

	if len(updates) == 0 {
		return nil
	}

	return &central.NetworkFlowUpdate{
		Updated: updates,
		Time:    timestamp.Now().GogoProtobuf(),
	}
}

func (m *networkFlowManager) enrichAndSend() {
	current := m.currentEnrichedConns()

	protoToSend := computeUpdateMessage(current, m.enrichedLastSentState)
	m.enrichedLastSentState = current

	if protoToSend == nil {
		return
	}

	metrics.IncrementTotalNetworkFlowsSentCounter(env.ClusterID.Setting(), len(protoToSend.Updated))
	log.Debugf("Flow update : %v", protoToSend)
	select {
	case <-m.done.Done():
		return
	case m.flowUpdates <- protoToSend:
		return
	}
}

func (m *networkFlowManager) currentEnrichedConns() map[networkConnIndicator]timestamp.MicroTS {
	conns := m.getAllConnections()

	enrichedConnections := make(map[networkConnIndicator]timestamp.MicroTS)
	for conn, ts := range conns {
		container, ok := m.clusterEntities.LookupByContainerID(conn.srcContainerID)
		if !ok {
			log.Errorf("Unable to fetch source deployment information, deployment does not exist for container %s", conn.srcContainerID)
			continue
		}

		for _, lookupResult := range m.clusterEntities.LookupByEndpoint(conn.dest) {
			for _, port := range lookupResult.ContainerPorts {
				indicator := networkConnIndicator{
					srcDeploymentID: container.DeploymentID,
					dstDeploymentID: lookupResult.DeploymentID,
					dstPort:         port,
					protocol:        conn.dest.L4Proto.ToProtobuf(),
				}

				// Multiple connections from a collector can result in a single enriched connection
				// hence update the timestamp only if we have a more recent connection than the one we have already enriched.
				if oldTS, found := enrichedConnections[indicator]; !found || oldTS < ts {
					enrichedConnections[indicator] = ts
				}
			}
		}
	}

	return enrichedConnections
}

func (m *networkFlowManager) getAllConnections() map[connection]timestamp.MicroTS {
	// Phase 1: get a snapshot of all *hostConnections.
	m.connectionsByHostMutex.Lock()
	allHostConns := make([]*hostConnections, 0, len(m.connectionsByHost))
	for _, hostConns := range m.connectionsByHost {
		allHostConns = append(allHostConns, hostConns)
	}
	m.connectionsByHostMutex.Unlock()

	// Phase 2: Merge all connections from all *hostConnections into a single map. This two-phase approach avoids
	// holding two locks simultaneously.
	allConnections := make(map[connection]timestamp.MicroTS)
	for _, hostConns := range allHostConns {
		hostConns.mutex.Lock()
		for conn, ts := range hostConns.connections {
			allConnections[conn] = ts
		}
		hostConns.mutex.Unlock()
	}

	return allConnections
}

func (m *networkFlowManager) RegisterCollector(hostname string) (HostNetworkInfo, int64) {

	m.connectionsByHostMutex.Lock()
	conns := m.connectionsByHost[hostname]

	if conns == nil {
		conns = &hostConnections{
			connections: make(map[connection]timestamp.MicroTS),
		}
		m.connectionsByHost[hostname] = conns
	}

	m.connectionsByHostMutex.Unlock()

	conns.mutex.Lock()
	seqID := conns.sequenceID + 1
	conns.mutex.Unlock()

	return conns, seqID
}

func (h *hostConnections) Process(networkInfo *sensor.NetworkConnectionInfo, nowTimestamp timestamp.MicroTS, sequenceID int64) error {
	updatedConnections := getUpdatedConnections(networkInfo)

	collectorTS := timestamp.FromProtobuf(networkInfo.GetTime())
	tsOffset := nowTimestamp - collectorTS

	h.mutex.Lock()
	defer h.mutex.Unlock()

	if sequenceID < h.sequenceID {
		return errors.New("replaced by newer connection")
	} else if sequenceID > h.sequenceID {
		// This is the first message of the new connection.
		for c := range h.connections {
			// Mark all connections as closed this is the first update
			// after a connection went down and came back up again.
			h.connections[c] = h.lastKnownTimestamp
		}
		h.sequenceID = sequenceID
	}

	for c, t := range updatedConnections {
		// timestamp = zero implies the connection is newly added. Add new connections, update existing ones to mark them closed
		if t != timestamp.InfiniteFuture { // adjust timestamp if not zero.
			t += tsOffset
		}
		h.connections[c] = t
	}

	h.lastKnownTimestamp = nowTimestamp

	return nil
}

func getUpdatedConnections(networkInfo *sensor.NetworkConnectionInfo) map[connection]timestamp.MicroTS {
	updatedConnections := make(map[connection]timestamp.MicroTS)

	for _, conn := range networkInfo.GetUpdatedConnections() {
		// Ignore connection originating from a server
		if conn.Role != v1.ClientServerRole_ROLE_CLIENT {
			continue
		}

		remoteEndpoint := net.MakeNumericEndpoint(net.IPFromBytes(conn.GetRemoteAddress().GetAddressData()), uint16(conn.GetRemoteAddress().GetPort()), net.L4ProtoFromProtobuf(conn.GetProtocol()))
		c := connection{
			srcContainerID: conn.GetContainerId(),
			srcAddr:        net.IPFromBytes(conn.GetLocalAddress().GetAddressData()),
			dest:           remoteEndpoint,
		}

		// timestamp will be set to close timestamp for closed connections, and zero for newly added connection.
		ts := timestamp.FromProtobuf(conn.CloseTimestamp)
		if ts == 0 {
			ts = timestamp.InfiniteFuture
		}
		updatedConnections[c] = ts
	}

	return updatedConnections
}
