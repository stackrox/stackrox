package manager

import (
	"sync"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/data/common"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/cache"
)

type hostConnections struct {
	connections        map[connection]time.Time
	lastKnownTimestamp time.Time

	mutex sync.Mutex
}

type networkConnIndicator struct {
	srcDeploymentID string
	dstDeploymentID string
	dstPort         uint16
	protocol        data.L4Protocol
}

// connection is an instance of a connection as reported by collector
type connection struct {
	srcAddr     string
	dstAddr     string
	dstPort     uint16
	containerID string
	protocol    data.L4Protocol
}

type networkFlowManager struct {
	connectionsByHost      map[string]*hostConnections
	connectionsByHostMutex sync.Mutex

	pendingCache *cache.PendingEvents

	enrichedConnections      map[networkConnIndicator]time.Time
	enrichedConnectionsMutex sync.Mutex

	done concurrency.Signal
}

func (m *networkFlowManager) Start() {
	go m.enrichConnections()
}

func (m *networkFlowManager) Stop() {
	m.done.Signal()
}

func (m *networkFlowManager) enrichConnections() {
	ticker := time.NewTicker(time.Second * 30)

	for {
		select {
		case <-m.done.WaitC():
			return
		case <-ticker.C:
			m.enrich()
		}
	}
}

func (m *networkFlowManager) enrich() {
	conns := m.getAllConnections()

	enrichedConnections := make(map[networkConnIndicator]time.Time)
	for conn, ts := range conns {

		srcDeploymentID, exists := m.pendingCache.FetchDeploymentByContainer(conn.containerID)
		if !exists {
			log.Errorf("Unable to fetch source deployment information, deployment does not exist for container %s", conn.containerID)
			continue
		}

		indicator := networkConnIndicator{
			srcDeploymentID: srcDeploymentID,
			dstDeploymentID: "",
			dstPort:         conn.dstPort,
			protocol:        conn.protocol,
		}
		/*
		 * Multiple connections from a collector can result in a single enriched connection
		 * hence update the timestamp only if we have a more recent connection than the one we have already enriched.
		 */

		if oldTS, found := enrichedConnections[indicator]; !found || oldTS.Before(ts) {
			enrichedConnections[indicator] = ts
		}
	}

	m.enrichedConnectionsMutex.Lock()
	m.enrichedConnections = enrichedConnections
	m.enrichedConnectionsMutex.Unlock()

	// @todo(boo): Send enriched network connections to Central
}

func (m *networkFlowManager) getAllConnections() map[connection]time.Time {
	m.connectionsByHostMutex.Lock()
	defer m.connectionsByHostMutex.Unlock()

	allConnections := make(map[connection]time.Time)
	for _, c := range m.connectionsByHost {
		for conn, ts := range c.connections {
			allConnections[conn] = ts
		}
	}

	return allConnections
}

func (m *networkFlowManager) RegisterCollector(hostname string) HostNetworkInfo {

	m.connectionsByHostMutex.Lock()
	conns := m.connectionsByHost[hostname]

	if conns == nil {
		conns = &hostConnections{
			connections: make(map[connection]time.Time),
		}
		m.connectionsByHost[hostname] = conns
	}

	m.connectionsByHostMutex.Unlock()

	conns.mutex.Lock()
	conns.lastKnownTimestamp = time.Now()
	conns.mutex.Unlock()

	return conns
}

func (h *hostConnections) Process(networkInfo *sensor.NetworkConnectionInfo, currTimestamp time.Time, isFirst bool) {
	updatedConnections := getUpdatedConnections(networkInfo)

	h.mutex.Lock()
	defer h.mutex.Unlock()

	if isFirst {
		for c := range h.connections {
			// Mark all connections as closed this is the first update
			// after a connection went down and came back up again.
			h.connections[c] = h.lastKnownTimestamp
		}
	}

	for c, t := range updatedConnections {
		// timestamp = zero implies the connection is newly added. Add new connections, update existing ones to mark them closed
		h.connections[c] = t
	}

	h.lastKnownTimestamp = currTimestamp
}

func getUpdatedConnections(networkInfo *sensor.NetworkConnectionInfo) map[connection]time.Time {
	updatedConnections := make(map[connection]time.Time)

	for _, conn := range networkInfo.GetUpdatedConnections() {
		// Ignore connection originating from a server
		if conn.Role != data.Role_ROLE_CLIENT {
			continue
		}
		c := connection{
			srcAddr:     string(conn.GetLocalAddress().GetAddressData()),
			dstAddr:     string(conn.GetRemoteAddress().GetAddressData()),
			dstPort:     uint16(conn.GetRemoteAddress().GetPort()),
			containerID: conn.GetContainerId(),
			protocol:    conn.GetProtocol(),
		}

		// timestamp will be set to close timestamp for closed connections, and zero for newly added connection.
		if conn.CloseTimestamp != nil {
			timestamp, err := types.TimestampFromProto(conn.CloseTimestamp)
			if err != nil {
				log.Errorf("Unable to convert close timestamp in proto: %s", conn.CloseTimestamp)
				continue
			}
			updatedConnections[c] = timestamp
		} else {
			updatedConnections[c] = time.Unix(0, 0).UTC()
		}

	}

	return updatedConnections
}
