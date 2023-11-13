package manager

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	mocksExternalSrc "github.com/stackrox/rox/sensor/common/externalsrcs/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	mocksManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	"go.uber.org/mock/gomock"
)

func createManager(mockCtrl *gomock.Controller) (*networkFlowManager, *mocksManager.MockEntityStore, *mocksExternalSrc.MockStore, *mocksDetector.MockDetector) {
	mockEntityStore := mocksManager.NewMockEntityStore(mockCtrl)
	mockExternalStore := mocksExternalSrc.NewMockStore(mockCtrl)
	mockDetector := mocksDetector.NewMockDetector(mockCtrl)
	ticker := time.NewTicker(100 * time.Millisecond)
	mgr := &networkFlowManager{
		clusterEntities:   mockEntityStore,
		externalSrcs:      mockExternalStore,
		policyDetector:    mockDetector,
		done:              concurrency.NewSignal(),
		connectionsByHost: make(map[string]*hostConnections),
		sensorUpdates:     make(chan *message.ExpiringMessage),
		publicIPs:         newPublicIPsManager(),
		centralReady:      concurrency.NewSignal(),
		enricherTicker:    ticker,
		finished:          &sync.WaitGroup{},
	}
	return mgr, mockEntityStore, mockExternalStore, mockDetector
}

func createManagerWithEntityStore(mockCtrl *gomock.Controller, eStore *clusterentities.Store) (*networkFlowManager, *mocksExternalSrc.MockStore, *mocksDetector.MockDetector) {
	mockExternalStore := mocksExternalSrc.NewMockStore(mockCtrl)
	mockDetector := mocksDetector.NewMockDetector(mockCtrl)
	ticker := time.NewTicker(100 * time.Millisecond)
	mgr := &networkFlowManager{
		clusterEntities:   eStore,
		externalSrcs:      mockExternalStore,
		policyDetector:    mockDetector,
		done:              concurrency.NewSignal(),
		connectionsByHost: make(map[string]*hostConnections),
		sensorUpdates:     make(chan *message.ExpiringMessage),
		publicIPs:         newPublicIPsManager(),
		centralReady:      concurrency.NewSignal(),
		enricherTicker:    ticker,
		finished:          &sync.WaitGroup{},
	}
	return mgr, mockExternalStore, mockDetector
}

type expectFn func()

func (f expectFn) runIfSet() {
	if f != nil {
		f()
	}
}

func expectEntityLookupContainerHelper(mockEntityStore *mocksManager.MockEntityStore, times int, containerMetadata clusterentities.ContainerMetadata, found bool) expectFn {
	return func() {
		mockEntityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(times).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool) {
			return containerMetadata, found
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
	c.conn.remote.IPAndPort.Address = externalIPv4Addr
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

func createEndpointPair(firstSeen timestamp.MicroTS) *endpointPair {
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
		},
	}
}

func (ep *endpointPair) containerID(id string) *endpointPair {
	ep.endpoint.containerID = id
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
