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
	"github.com/stackrox/rox/pkg/networkentity"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/metrics"
)

const (
	// Wait at least this long before determining that an unresolvable IP is "outside of the cluster".
	clusterEntityResolutionWaitPeriod = 10 * time.Second
)

var (
	internetUUID = uuid.Nil
)

type hostConnections struct {
	connections        map[connection]*connStatus
	lastKnownTimestamp timestamp.MicroTS
	sequenceID         int64

	mutex sync.Mutex
}

type connStatus struct {
	firstSeen timestamp.MicroTS
	lastSeen  timestamp.MicroTS
	used      bool
}

type networkConnIndicator struct {
	srcEntity networkentity.Entity
	dstEntity networkentity.Entity
	dstPort   uint16
	protocol  v1.L4Protocol
}

func (i networkConnIndicator) toProto(ts timestamp.MicroTS) *v1.NetworkFlow {
	proto := &v1.NetworkFlow{
		Props: &v1.NetworkFlowProperties{
			SrcEntity:  i.srcEntity.ToProto(),
			DstEntity:  i.dstEntity.ToProto(),
			DstPort:    uint32(i.dstPort),
			L4Protocol: i.protocol,
		},
	}

	if ts != timestamp.InfiniteFuture {
		proto.LastSeenTimestamp = ts.GogoProtobuf()
	}
	return proto
}

// connection is an instance of a connection as reported by collector
type connection struct {
	local       net.IPPortPair
	remote      net.NumericEndpoint
	containerID string
	incoming    bool
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
	for conn, status := range conns {
		container, ok := m.clusterEntities.LookupByContainerID(conn.containerID)
		if !ok {
			log.Errorf("Unable to fetch deployment information for container %s: no deployment found", conn.containerID)
			continue
		}

		lookupResults := m.clusterEntities.LookupByEndpoint(conn.remote)
		if len(lookupResults) == 0 {
			if timestamp.Now().ElapsedSince(status.firstSeen) < clusterEntityResolutionWaitPeriod {
				continue
			}
			status.used = true

			var port uint16
			if conn.incoming {
				port = conn.local.Port
			} else {
				port = conn.remote.IPAndPort.Port
			}

			// Fake a lookup result with an empty deployment ID.
			lookupResults = []clusterentities.LookupResult{
				{
					Entity: networkentity.Entity{
						Type: v1.NetworkEntityInfo_INTERNET,
					},
					ContainerPorts: []uint16{port},
				},
			}
		} else {
			status.used = true
			if conn.incoming {
				// Only report incoming connections from outside of the cluster. These are already taken care of by the
				// corresponding outgoing connection from the other end.
				continue
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
					indicator.dstEntity = networkentity.ForDeployment(container.DeploymentID)
				} else {
					indicator.srcEntity = networkentity.ForDeployment(container.DeploymentID)
					indicator.dstEntity = lookupResult.Entity
				}

				// Multiple connections from a collector can result in a single enriched connection
				// hence update the timestamp only if we have a more recent connection than the one we have already enriched.
				if oldTS, found := enrichedConnections[indicator]; !found || oldTS < status.lastSeen {
					enrichedConnections[indicator] = status.lastSeen
				}
			}
		}
	}

	return enrichedConnections
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

func (m *networkFlowManager) getAllConnections() map[connection]*connStatus {
	// Phase 1: get a snapshot of all *hostConnections.
	m.connectionsByHostMutex.Lock()
	allHostConns := make([]*hostConnections, 0, len(m.connectionsByHost))
	for _, hostConns := range m.connectionsByHost {
		allHostConns = append(allHostConns, hostConns)
	}
	m.connectionsByHostMutex.Unlock()

	// Phase 2: Merge all connections from all *hostConnections into a single map. This two-phase approach avoids
	// holding two locks simultaneously.
	allConnections := make(map[connection]*connStatus)
	for _, hostConns := range allHostConns {
		hostConns.mutex.Lock()
		for conn, status := range hostConns.connections {
			allConnections[conn] = status
			if status.lastSeen != timestamp.InfiniteFuture && status.used {
				delete(hostConns.connections, conn) // connection not active, no longer needed
			}
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
			connections: make(map[connection]*connStatus),
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
		for _, status := range h.connections {
			// Mark all connections as closed this is the first update
			// after a connection went down and came back up again.
			status.lastSeen = h.lastKnownTimestamp
		}
		h.sequenceID = sequenceID
	}

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
	}

	h.lastKnownTimestamp = nowTimestamp

	return nil
}

func getIPAndPort(address *sensor.NetworkAddress) net.IPPortPair {
	return net.IPPortPair{
		Address: net.IPFromBytes(address.GetAddressData()),
		Port:    uint16(address.GetPort()),
	}
}

func getUpdatedConnections(networkInfo *sensor.NetworkConnectionInfo) map[connection]timestamp.MicroTS {
	updatedConnections := make(map[connection]timestamp.MicroTS)

	for _, conn := range networkInfo.GetUpdatedConnections() {
		var incoming bool
		switch conn.Role {
		case v1.ClientServerRole_ROLE_SERVER:
			incoming = true
		case v1.ClientServerRole_ROLE_CLIENT:
			incoming = false
		default:
			continue
		}

		remote := net.NumericEndpoint{
			IPAndPort: getIPAndPort(conn.GetRemoteAddress()),
			L4Proto:   net.L4ProtoFromProtobuf(conn.GetProtocol()),
		}
		local := getIPAndPort(conn.GetLocalAddress())
		c := connection{
			local:       local,
			remote:      remote,
			containerID: conn.GetContainerId(),
			incoming:    incoming,
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
