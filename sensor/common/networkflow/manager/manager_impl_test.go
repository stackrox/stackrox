package manager

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	mocksExternalSrc "github.com/stackrox/rox/sensor/common/externalsrcs/mocks"
	mocksManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	openNetworkEndpoint = &sensor.NetworkEndpoint{
		SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
		Protocol:     storage.L4Protocol_L4_PROTOCOL_TCP,
		ContainerId:  "FakeContainerId",
		ListenAddress: &sensor.NetworkAddress{
			Port: 80,
		},
		Originator: &storage.NetworkProcessUniqueKey{
			ProcessName:         "socat",
			ProcessExecFilePath: "/usr/bin/socat",
			ProcessArgs:         "port: 80",
		},
	}
	openNetworkEndpoint81 = &sensor.NetworkEndpoint{
		SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
		Protocol:     storage.L4Protocol_L4_PROTOCOL_TCP,
		ContainerId:  "FakeContainerId",
		ListenAddress: &sensor.NetworkAddress{
			Port: 81,
		},
		Originator: &storage.NetworkProcessUniqueKey{
			ProcessName:         "socat",
			ProcessExecFilePath: "/usr/bin/socat",
			ProcessArgs:         "port: 81",
		},
	}
	openNetworkEndpointNoOriginator = &sensor.NetworkEndpoint{
		SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
		Protocol:     storage.L4Protocol_L4_PROTOCOL_TCP,
		ContainerId:  "FakeContainerId",
		ListenAddress: &sensor.NetworkAddress{
			Port: 80,
		},
	}
	closedNetworkEndpoint = &sensor.NetworkEndpoint{
		SocketFamily:   sensor.SocketFamily_SOCKET_FAMILY_IPV4,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		ContainerId:    "FakeContainerId",
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
		ListenAddress: &sensor.NetworkAddress{
			Port: 80,
		},
		Originator: &storage.NetworkProcessUniqueKey{
			ProcessName:         "socat",
			ProcessExecFilePath: "/usr/bin/socat",
			ProcessArgs:         "port: 80",
		},
	}
)

func TestNetworkFlowManager(t *testing.T) {
	suite.Run(t, new(NetworkFlowManagerTestSuite))
}

type NetworkFlowManagerTestSuite struct {
	suite.Suite
}

//region hostConnection.Process tests

func (s *NetworkFlowManagerTestSuite) TestAddNothing() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfo := &sensor.NetworkConnectionInfo{}
	nowTimestamp := timestamp.Now()
	var sequenceID int64
	err := h.Process(networkInfo, nowTimestamp, sequenceID)
	s.NoError(err)
	s.Len(h.endpoints, 0)
}

func (s *NetworkFlowManagerTestSuite) TestAddOpen() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfo := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64
	h.connectionsSequenceID = sequenceID
	err := h.Process(networkInfo, nowTimestamp, sequenceID)
	s.NoError(err)
	s.Len(h.endpoints, 1)
}

func (s *NetworkFlowManagerTestSuite) TestAddOpenAndClosed() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfoOpen := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint},
	}

	networkInfoClosed := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{closedNetworkEndpoint},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64
	h.connectionsSequenceID = sequenceID

	err := h.Process(networkInfoOpen, nowTimestamp, sequenceID)
	s.NoError(err)

	err = h.Process(networkInfoClosed, nowTimestamp, sequenceID)
	s.NoError(err)

	s.Len(h.endpoints, 1)
}

func (s *NetworkFlowManagerTestSuite) TestAddTwoDifferent() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfoOpen := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint},
	}

	networkInfoOpen81 := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint81},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64
	h.connectionsSequenceID = sequenceID

	err := h.Process(networkInfoOpen, nowTimestamp, sequenceID)
	s.NoError(err)

	err = h.Process(networkInfoOpen81, nowTimestamp, sequenceID)
	s.NoError(err)

	s.Len(h.endpoints, 2)
}

func (s *NetworkFlowManagerTestSuite) TestAddTwoDifferentSameBatch() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfoOpen := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpoint, openNetworkEndpoint81},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64
	h.connectionsSequenceID = sequenceID

	err := h.Process(networkInfoOpen, nowTimestamp, sequenceID)
	s.NoError(err)

	s.Len(h.endpoints, 2)
}

func (s *NetworkFlowManagerTestSuite) TestAddNoOriginator() {
	h := hostConnections{}
	h.endpoints = make(map[containerEndpoint]*connStatus)

	networkInfoOpen := &sensor.NetworkConnectionInfo{
		UpdatedEndpoints: []*sensor.NetworkEndpoint{openNetworkEndpointNoOriginator},
	}

	nowTimestamp := timestamp.Now()
	var sequenceID int64
	h.connectionsSequenceID = sequenceID

	err := h.Process(networkInfoOpen, nowTimestamp, sequenceID)
	s.NoError(err)

	s.Len(h.endpoints, 1)
}

//endregion

//region networkFlowManager tests

func (s *NetworkFlowManagerTestSuite) TestEnrichConnection() {
	mockCtrl := gomock.NewController(s.T())
	m, mockEntityStore, mockExternalSrc, _ := createManager(mockCtrl)
	containerID := "id"
	cases := map[string]struct {
		connPair                    *connectionPair
		enrichConnections           map[networkConnIndicator]timestamp.MicroTS
		expectEntityLookupContainer expectFn
		expectEntityLookupEndpoint  expectFn
		expectExternalLookup        expectFn
		expectedIndicator           *networkConnIndicator
		expectedConnection          *connection
		expectedStatus              *connStatus
	}{
		"Rotten connection": {
			connPair:                    createConnectionPair(true, true, false, timestamp.Now().Add(-maxContainerResolutionWaitPeriod*2)),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{}, false),
			expectedStatus: &connStatus{
				rotten: true,
			},
		},
		"Incoming External connection no external sources hit": {
			connPair:          createConnectionPair(true, true, false, timestamp.Now()),
			enrichConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{
				DeploymentID: containerID,
			}, true),
			expectExternalLookup: expectExternalLookupHelper(mockExternalSrc, 1, nil),
			expectedStatus: &connStatus{
				used: true,
			},
			expectedIndicator: &networkConnIndicator{
				dstPort:   80,
				protocol:  net.TCP.ToProtobuf(),
				srcEntity: networkgraph.InternetEntity(),
				dstEntity: networkgraph.EntityForDeployment(containerID),
			},
		},
		"Outgoing External external sources hit": {
			connPair:          createConnectionPair(false, true, false, timestamp.Now()),
			enrichConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{
				DeploymentID: containerID,
			}, true),
			expectExternalLookup: expectExternalLookupHelper(mockExternalSrc, 1, &storage.NetworkEntityInfo{
				Id: containerID,
			}),
			expectedStatus: &connStatus{
				used: true,
			},
			expectedIndicator: &networkConnIndicator{
				dstPort:  80,
				protocol: net.TCP.ToProtobuf(),
				dstEntity: networkgraph.EntityFromProto(&storage.NetworkEntityInfo{
					Id: containerID,
				}),
				srcEntity: networkgraph.EntityForDeployment(containerID),
			},
		},
		"Incoming": {
			connPair:          createConnectionPair(true, false, false, timestamp.Now()),
			enrichConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{
				DeploymentID: containerID,
			}, true),
			expectEntityLookupEndpoint: expectEntityLookupEndpointHelper(mockEntityStore, 1, []clusterentities.LookupResult{
				{
					Entity: networkgraph.Entity{
						ID: containerID,
					},
				},
			}),
			expectedStatus: &connStatus{
				used: true,
			},
		},
		"Incoming fresh connection with valid address": {
			connPair:          createConnectionPair(true, false, false, timestamp.Now()),
			enrichConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{
				DeploymentID: containerID,
			}, true),
			expectEntityLookupEndpoint: expectEntityLookupEndpointHelper(mockEntityStore, 1, nil),
			expectedStatus:             &connStatus{},
		},
		"Incoming fresh connection with invalid address": {
			connPair:          createConnectionPair(true, false, true, timestamp.Now()),
			enrichConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{
				DeploymentID: containerID,
			}, true),
			expectEntityLookupEndpoint: expectEntityLookupEndpointHelper(mockEntityStore, 1, nil),
			expectExternalLookup:       expectExternalLookupHelper(mockExternalSrc, 1, nil),
			expectedStatus:             &connStatus{},
		},
		"Outgoing": {
			connPair:          createConnectionPair(false, false, false, timestamp.Now()),
			enrichConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{
				DeploymentID: containerID,
			}, true),
			expectEntityLookupEndpoint: expectEntityLookupEndpointHelper(mockEntityStore, 1, []clusterentities.LookupResult{
				{
					Entity: networkgraph.Entity{
						ID: containerID,
					},
					ContainerPorts: []uint16{
						80,
					},
				},
			}),
			expectedStatus: &connStatus{
				used: true,
			},
			expectedIndicator: &networkConnIndicator{
				dstPort:  80,
				protocol: net.TCP.ToProtobuf(),
				dstEntity: networkgraph.EntityFromProto(&storage.NetworkEntityInfo{
					Id: containerID,
				}),
				srcEntity: networkgraph.EntityForDeployment(containerID),
			},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			tCase.expectEntityLookupContainer.runIfSet()
			tCase.expectEntityLookupEndpoint.runIfSet()
			tCase.expectExternalLookup.runIfSet()
			m.enrichConnection(tCase.connPair.conn, tCase.connPair.status, tCase.enrichConnections)
			s.Assert().Equal(tCase.expectedStatus.used, tCase.connPair.status.used)
			s.Assert().Equal(tCase.expectedStatus.rotten, tCase.connPair.status.rotten)
			if tCase.expectedIndicator != nil {
				_, ok := tCase.enrichConnections[*tCase.expectedIndicator]
				s.Assert().True(ok)
			} else {
				s.Assert().Len(tCase.enrichConnections, 0)
			}
		})
	}
}

func (s *NetworkFlowManagerTestSuite) TestEnrichContainerEndpoint() {
	mockCtrl := gomock.NewController(s.T())
	m, mockEntityStore, _, _ := createManager(mockCtrl)
	containerID := "id"
	cases := map[string]struct {
		endpointPair                *endpointPair
		enrichConnections           map[containerEndpointIndicator]timestamp.MicroTS
		expectEntityLookupContainer expectFn
		expectedStatus              *connStatus
		expectedEndpoint            *containerEndpointIndicator
	}{
		"Rotten connection": {
			endpointPair:                createEndpointPair(timestamp.Now().Add(-maxContainerResolutionWaitPeriod * 2)),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{}, false),
			expectedStatus: &connStatus{
				rotten: true,
				used:   true,
			},
		},
		"Endpoint": {
			endpointPair:      createEndpointPair(timestamp.Now()),
			enrichConnections: make(map[containerEndpointIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{
				DeploymentID: containerID,
			}, true),
			expectedStatus: &connStatus{used: true},
			expectedEndpoint: &containerEndpointIndicator{
				entity:   networkgraph.EntityForDeployment(containerID),
				port:     80,
				protocol: net.TCP.ToProtobuf(),
			},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			tCase.expectEntityLookupContainer.runIfSet()
			m.enrichContainerEndpoint(tCase.endpointPair.endpoint, tCase.endpointPair.status, tCase.enrichConnections)
			s.Assert().Equal(tCase.expectedStatus.rotten, tCase.endpointPair.status.rotten)
			s.Assert().Equal(tCase.expectedStatus.used, tCase.endpointPair.status.used)
			if tCase.expectedEndpoint != nil {
				_, ok := tCase.enrichConnections[*tCase.expectedEndpoint]
				s.Assert().True(ok)
			}
		})
	}
}

func (s *NetworkFlowManagerTestSuite) TestEnrichProcessListening() {
	mockCtrl := gomock.NewController(s.T())
	m, mockEntityStore, _, _ := createManager(mockCtrl)
	containerID := "id"
	cases := map[string]struct {
		containerPair               *containerPair
		enrichConnections           map[processListeningIndicator]timestamp.MicroTS
		expectEntityLookupContainer expectFn
		expectedStatus              *connStatus
		expectedListeningIndicator  *processListeningIndicator
	}{
		"Rotten connection": {
			containerPair:               createContainerPair(timestamp.Now().Add(-maxContainerResolutionWaitPeriod * 2)),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{}, false),
			expectedStatus: &connStatus{
				rotten:      true,
				usedProcess: true,
			},
		},
		"Container Endpoint": {
			containerPair:     createContainerPair(timestamp.Now()),
			enrichConnections: make(map[processListeningIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{
				DeploymentID:  containerID,
				ContainerName: "container-name",
				PodID:         containerID,
			}, true),
			expectedStatus: &connStatus{
				usedProcess: true,
			},
			expectedListeningIndicator: &processListeningIndicator{
				key: processUniqueKey{
					podID:         containerID,
					containerName: "container-name",
					deploymentID:  containerID,
					process:       defaultProcessKey(),
				},
				port:     80,
				protocol: net.TCP.ToProtobuf(),
			},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			tCase.expectEntityLookupContainer.runIfSet()
			m.enrichProcessListening(tCase.containerPair.endpoint, tCase.containerPair.status, tCase.enrichConnections)
			s.Assert().Equal(tCase.expectedStatus.rotten, tCase.containerPair.status.rotten)
			s.Assert().Equal(tCase.expectedStatus.usedProcess, tCase.containerPair.status.usedProcess)
			if tCase.expectedListeningIndicator != nil {
				_, ok := tCase.enrichConnections[*tCase.expectedListeningIndicator]
				s.Assert().True(ok)
			}
		})
	}
}

func (s *NetworkFlowManagerTestSuite) TestManagerOfflineMode() {
	s.T().Setenv(env.ProcessesListeningOnPort.EnvVar(), "false")
	containerID := "container-id"
	mockCtrl := gomock.NewController(s.T())
	m, mockEntity, _, mockDetector := createManager(mockCtrl)
	states := []struct {
		notify                      common.SensorComponentEvent
		connections                 []*connectionHostnamePair
		expectEntityLookupContainer expectFn
		expectEntityLookupEndpoint  expectFn
		expectDetector              expectFn
		expectedSensorMessage       []*central.MsgFromSensor
	}{
		{
			notify:      common.SensorComponentEventOfflineMode,
			connections: []*connectionHostnamePair{createConnectionHostnamePair("hostname-1", createConnectionPair(false, false, false, timestamp.Now()))},
		},
		{
			notify: common.SensorComponentEventCentralReachable,
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntity, 1, clusterentities.ContainerMetadata{
				DeploymentID: containerID,
			}, true),
			expectEntityLookupEndpoint: expectEntityLookupEndpointHelper(mockEntity, 1, []clusterentities.LookupResult{
				{
					Entity:         networkgraph.Entity{ID: containerID},
					ContainerPorts: []uint16{80},
				},
			}),
			expectDetector: expectDetectorHelper(mockDetector, 1),
			expectedSensorMessage: []*central.MsgFromSensor{
				{
					Msg: &central.MsgFromSensor_NetworkFlowUpdate{
						NetworkFlowUpdate: &central.NetworkFlowUpdate{
							Updated: []*storage.NetworkFlow{
								{
									Props: &storage.NetworkFlowProperties{
										SrcEntity: networkgraph.EntityForDeployment(containerID).ToProto(),
										DstEntity: networkgraph.EntityFromProto(&storage.NetworkEntityInfo{
											Id: containerID,
										}).ToProto(),
										DstPort:    80,
										L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	fakeTicker := make(chan time.Time, 1)
	defer close(fakeTicker)
	go m.enrichConnections(fakeTicker)
	for i, state := range states {
		for _, cnn := range state.connections {
			addHostConnection(m, cnn.hostname, cnn.conn)
		}
		s.Run(fmt.Sprintf("iteration %d", i), func() {
			state.expectEntityLookupContainer.runIfSet()
			state.expectEntityLookupEndpoint.runIfSet()
			state.expectDetector.runIfSet()
			m.Notify(state.notify)
			fakeTicker <- time.Now()
			if len(state.expectedSensorMessage) > 0 {
				select {
				case <-time.After(10 * time.Second):
					s.Fail("timeout")
				case msg, ok := <-m.sensorUpdates:
					s.Require().True(ok, "channel should not be closed")
					s.Assert().NotNil(msg)
				}
			} else {
				select {
				case _, ok := <-m.sensorUpdates:
					s.Require().True(ok, "channel should not be closed")
					s.Fail("Should not received a message")
				case <-time.After(time.Second):
					break
				}
			}
		})
	}
	m.Stop(nil)
	m.done.Wait()
}

//endregion

//region Helper functions

func createManager(mockCtrl *gomock.Controller) (*networkFlowManager, *mocksManager.MockEntityStore, *mocksExternalSrc.MockStore, *mocksDetector.MockDetector) {
	mockEntityStore := mocksManager.NewMockEntityStore(mockCtrl)
	mockExternalStore := mocksExternalSrc.NewMockStore(mockCtrl)
	mockDetector := mocksDetector.NewMockDetector(mockCtrl)
	mgr := &networkFlowManager{
		clusterEntities:   mockEntityStore,
		externalSrcs:      mockExternalStore,
		policyDetector:    mockDetector,
		done:              concurrency.NewSignal(),
		connectionsByHost: make(map[string]*hostConnections),
		sensorUpdates:     make(chan *central.MsgFromSensor),
		publicIPs:         newPublicIPsManager(),
		centralReady:      concurrency.NewSignal(),
	}
	return mgr, mockEntityStore, mockExternalStore, mockDetector
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

func expectExternalLookupHelper(mockExternalStore *mocksExternalSrc.MockStore, times int, retVal *storage.NetworkEntityInfo) expectFn {
	return func() {
		mockExternalStore.EXPECT().LookupByNetwork(gomock.Any()).Times(times).DoAndReturn(func(_ any) *storage.NetworkEntityInfo {
			return retVal
		})
	}
}

func expectDetectorHelper(mockDetector *mocksDetector.MockDetector, times int) expectFn {
	return func() {
		mockDetector.EXPECT().ProcessNetworkFlow(gomock.Any()).MinTimes(times)
	}
}

type connectionPair struct {
	conn   *connection
	status *connStatus
}

func createConnectionPair(incoming bool, external bool, invalid bool, firstSeen timestamp.MicroTS) *connectionPair {
	address := externalIPv4Addr
	if !external {
		address = net.ParseIP("8.8.8.8")
	}
	if invalid {
		address = net.ParseIP("invalid")
	}
	ret := &connectionPair{
		conn: &connection{
			containerID: "container-id",
			incoming:    incoming,
			remote: net.NumericEndpoint{
				IPAndPort: net.NetworkPeerID{
					Address: address,
					Port:    80,
				},
				L4Proto: net.TCP,
			},
		},
		status: &connStatus{
			firstSeen: firstSeen,
		},
	}
	if incoming {
		ret.conn.local = net.NetworkPeerID{
			Port: 80,
		}
	}
	return ret
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

type connectionHostnamePair struct {
	hostname string
	conn     *connectionPair
}

func createConnectionHostnamePair(hostname string, connPair *connectionPair) *connectionHostnamePair {
	return &connectionHostnamePair{
		hostname: hostname,
		conn:     connPair,
	}
}

func addHostConnection(mgr *networkFlowManager, hostName string, connPair *connectionPair) {
	mgr.connectionsByHostMutex.Lock()
	defer mgr.connectionsByHostMutex.Unlock()
	conn := *connPair.conn
	h, ok := mgr.connectionsByHost[hostName]
	if !ok {
		h = &hostConnections{}
	}
	if h.connections == nil {
		h.connections = make(map[connection]*connStatus)
	}
	h.connections[conn] = connPair.status
	mgr.connectionsByHost[hostName] = h
}

//endregion
