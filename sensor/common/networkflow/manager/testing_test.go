package manager

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	mocksExternalSrc "github.com/stackrox/rox/sensor/common/externalsrcs/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	mocksManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	"go.uber.org/mock/gomock"
)

func createManager(mockCtrl *gomock.Controller, enrichTicker <-chan time.Time) (*networkFlowManager, *mocksManager.MockEntityStore, *mocksExternalSrc.MockStore, *mocksDetector.MockDetector) {
	mockEntityStore := mocksManager.NewMockEntityStore(mockCtrl)
	mockExternalStore := mocksExternalSrc.NewMockStore(mockCtrl)
	mockDetector := mocksDetector.NewMockDetector(mockCtrl)
	mgr := &networkFlowManager{
		clusterEntities:   mockEntityStore,
		externalSrcs:      mockExternalStore,
		policyDetector:    mockDetector,
		connectionsByHost: make(map[string]*hostConnections),
		sensorUpdates:     make(chan *message.ExpiringMessage, 5),
		publicIPs:         newPublicIPsManager(),
		centralReady:      concurrency.NewSignal(),
		enricherTicker:    time.NewTicker(time.Hour),
		enricherTickerC:   enrichTicker,
		activeConnections: make(map[connection]*networkConnIndicatorWithAge),
		activeEndpoints:   make(map[containerEndpoint]*containerEndpointIndicatorWithAge),
		stopper:           concurrency.NewStopper(),
	}
	return mgr, mockEntityStore, mockExternalStore, mockDetector
}

type expectFn func()

func (f expectFn) runIfSet() {
	if f != nil {
		f()
	}
}

func expectationsEndpointPurger(mockEntityStore *mocksManager.MockEntityStore, isKnownEndpoint, containerIDfound, historical bool) {
	mockEntityStore.EXPECT().LookupByContainerID(gomock.Any()).AnyTimes().DoAndReturn(
		func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
			return clusterentities.ContainerMetadata{}, containerIDfound, historical
		})
	mockEntityStore.EXPECT().LookupByEndpoint(gomock.Any()).AnyTimes().DoAndReturn(
		func(_ any) []clusterentities.LookupResult {
			if isKnownEndpoint {
				return []clusterentities.LookupResult{{
					Entity:         networkgraph.Entity{},
					ContainerPorts: []uint16{80},
					PortNames:      []string{"http"},
				}}
			}
			return []clusterentities.LookupResult{}
		})
	mockEntityStore.EXPECT().RegisterPublicIPsListener(gomock.Any()).AnyTimes()

}

func expectEntityLookupContainerHelper(mockEntityStore *mocksManager.MockEntityStore, times int, containerMetadata clusterentities.ContainerMetadata, found, historical bool) expectFn {
	return func() {
		mockEntityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(times).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
			return containerMetadata, found, false
		})
	}
}

func expectEntityLookupEndpointHelper(mockEntityStore *mocksManager.MockEntityStore, times int, retVal []clusterentities.LookupResult) expectFn {
	return func() {
		mockEntityStore.EXPECT().LookupByEndpoint(gomock.Any()).Times(times).DoAndReturn(func(_ any) []clusterentities.LookupResult {
			return retVal
		})
	}
}

func expectDetectorHelper(mockDetector *mocksDetector.MockDetector, times int) expectFn {
	return func() {
		mockDetector.EXPECT().ProcessNetworkFlow(gomock.Any(), gomock.Any()).Times(times)
	}
}

type connectionPair struct {
	conn   *connection
	status *connStatus
}

func createConnectionPair() *connectionPair {
	return &connectionPair{
		conn: &connection{
			containerID: "container-id",
			incoming:    false,
			remote: net.NumericEndpoint{
				IPAndPort: net.NetworkPeerID{
					Address: net.ParseIP("0.0.0.0"),
					Port:    80,
				},
				L4Proto: net.TCP,
			},
		},
		status: &connStatus{
			firstSeen: timestamp.Now(),
		},
	}
}

func (c *connectionPair) tsAdded(tsAdded timestamp.MicroTS) *connectionPair {
	c.status.tsAdded = tsAdded
	return c
}

func (c *connectionPair) lastSeen(lastSeen timestamp.MicroTS) *connectionPair {
	c.status.lastSeen = lastSeen
	return c
}

func (c *connectionPair) containerID(id string) *connectionPair {
	c.conn.containerID = id
	return c
}

func (c *connectionPair) incoming() *connectionPair {
	c.conn.incoming = true
	c.conn.local = net.NetworkPeerID{
		Port: 80,
	}
	return c
}

func (c *connectionPair) external() *connectionPair {
	c.conn.remote.IPAndPort.Address = net.ExternalIPv4Addr
	return c
}

func (c *connectionPair) invalidAddress() *connectionPair {
	c.conn.remote.IPAndPort.Address = net.ParseIP("invalid")
	return c
}

func (c *connectionPair) firstSeen(firstSeen timestamp.MicroTS) *connectionPair {
	c.status.firstSeen = firstSeen
	return c
}

type endpointPair struct {
	endpoint *containerEndpoint
	status   *connStatus
}

func createEndpointPair(firstSeen, tsAdded timestamp.MicroTS) *endpointPair {
	return &endpointPair{
		endpoint: &containerEndpoint{
			endpoint: net.NumericEndpoint{
				IPAndPort: net.NetworkPeerID{
					Address: net.ParseIP("8.8.8.8"),
					Port:    80,
				},
				L4Proto: net.TCP,
			},
			containerID: "container-id",
		},
		status: &connStatus{
			firstSeen: firstSeen,
			tsAdded:   tsAdded,
		},
	}
}

func (ep *endpointPair) containerID(id string) *endpointPair {
	ep.endpoint.containerID = id
	return ep
}

func (ep *endpointPair) lastSeen(lastSeen timestamp.MicroTS) *endpointPair {
	ep.status.lastSeen = lastSeen
	return ep
}

type containerPair struct {
	endpoint *containerEndpoint
	status   *connStatus
}

func defaultProcessKey() processInfo {
	return processInfo{
		processName: "process-name",
		processArgs: "process-args",
		processExec: "process-exec",
	}
}

func createContainerPair(firstSeen timestamp.MicroTS) *containerPair {
	return &containerPair{
		endpoint: &containerEndpoint{
			endpoint: net.NumericEndpoint{
				IPAndPort: net.NetworkPeerID{
					Address: net.ParseIP("8.8.8.8"),
					Port:    80,
				},
			},
			processKey: defaultProcessKey(),
		},
		status: &connStatus{
			firstSeen: firstSeen,
		},
	}
}

type HostnameAndConnections struct {
	hostname     string
	connPair     *connectionPair
	endpointPair *endpointPair
}

func createHostnameConnections(hostname string) *HostnameAndConnections {
	return &HostnameAndConnections{
		hostname: hostname,
	}
}

func (ch *HostnameAndConnections) withConnectionPair(pair *connectionPair) *HostnameAndConnections {
	ch.connPair = pair
	return ch
}

func (ch *HostnameAndConnections) withEndpointPair(pair *endpointPair) *HostnameAndConnections {
	ch.endpointPair = pair
	return ch
}

func addHostConnection(mgr *networkFlowManager, connectionsHostPair *HostnameAndConnections) {
	mgr.connectionsByHostMutex.Lock()
	defer mgr.connectionsByHostMutex.Unlock()
	h, ok := mgr.connectionsByHost[connectionsHostPair.hostname]
	if !ok {
		h = &hostConnections{}
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if connectionsHostPair.connPair != nil {
		if h.connections == nil {
			h.connections = make(map[connection]*connStatus)
		}
		conn := *connectionsHostPair.connPair.conn
		h.connections[conn] = connectionsHostPair.connPair.status
	}
	if connectionsHostPair.endpointPair != nil {
		if h.endpoints == nil {
			h.endpoints = make(map[containerEndpoint]*connStatus)
		}
		ep := *connectionsHostPair.endpointPair.endpoint
		h.endpoints[ep] = connectionsHostPair.endpointPair.status
	}
	mgr.connectionsByHost[connectionsHostPair.hostname] = h
}

type expectedEntitiesPair struct {
	srcID string
	dstID string
}

func createExpectedSensorMessageWithConnections(pairs ...*expectedEntitiesPair) *central.MsgFromSensor {
	var updates []*storage.NetworkFlow
	for _, pair := range pairs {
		updates = append(updates, &storage.NetworkFlow{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  networkgraph.EntityForDeployment(pair.srcID).ToProto(),
				DstEntity:  networkgraph.EntityFromProto(&storage.NetworkEntityInfo{Id: pair.dstID}).ToProto(),
				DstPort:    80,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
		})
	}
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_NetworkFlowUpdate{
			NetworkFlowUpdate: &central.NetworkFlowUpdate{
				Updated: updates,
			},
		},
	}
}

func createExpectedSensorMessageWithEndpoints(ids ...string) *central.MsgFromSensor {
	var updates []*storage.NetworkEndpoint
	for _, id := range ids {
		updates = append(updates, &storage.NetworkEndpoint{
			Props: &storage.NetworkEndpointProperties{
				Entity:     networkgraph.EntityForDeployment(id).ToProto(),
				Port:       80,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
		})
	}
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_NetworkFlowUpdate{
			NetworkFlowUpdate: &central.NetworkFlowUpdate{
				UpdatedEndpoints: updates,
			},
		},
	}
}

func (s *NetworkFlowManagerTestSuite) assertSensorMessageConnectionIDs(expectedUpdates []*storage.NetworkFlow, actualUpdates []*storage.NetworkFlow) {
	for _, exp := range expectedUpdates {
		found := false
		for _, actual := range actualUpdates {
			if exp.GetProps().GetSrcEntity().GetId() == actual.GetProps().GetSrcEntity().GetId() &&
				exp.GetProps().GetDstEntity().GetId() == actual.GetProps().GetDstEntity().GetId() {
				found = true
				break
			}
		}
		s.Assert().True(found, "expected flow with srcID %s and dstID %s not found", exp.Props.SrcEntity.Id, exp.Props.DstEntity.Id)
	}
}

func (s *NetworkFlowManagerTestSuite) assertSensorMessageEndpointIDs(expectedUpdates []*storage.NetworkEndpoint, actualUpdates []*storage.NetworkEndpoint) {
	for _, exp := range expectedUpdates {
		found := false
		for _, actual := range actualUpdates {
			if exp.GetProps().GetEntity().GetId() == actual.GetProps().GetEntity().GetId() {
				found = true
				break
			}
		}
		s.Assert().True(found, "expected endpoint  with ID %s not found", exp.GetProps().GetEntity().GetId())
	}
}
