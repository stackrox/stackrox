package manager

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// region hostConnection.Process tests

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

func (s *NetworkFlowManagerTestSuite) TestManagerOfflineMode() {
	// This test is for v1/v2 behavior
	s.T().Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), "false")
	s.T().Setenv(env.ProcessesListeningOnPort.EnvVar(), "false")
	const (
		srcID       = "src-id"
		dstID       = "dst-id"
		hostname    = "hostname"
		containerID = "container-id"
	)
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer mockCtrl.Finish()
	m, mockEntity, _, mockDetector := createManager(mockCtrl, enrichTickerC)
	states := []struct {
		testName                    string
		notify                      common.SensorComponentEvent
		connections                 []*HostnameAndConnections
		expectEntityLookupContainer expectFn
		expectEntityLookupEndpoint  expectFn
		expectDetector              expectFn
		expectedSensorMessage       *central.MsgFromSensor
	}{
		// The test cases are supposed to be run in order!
		{
			testName:    "In offline mode we should not send any messages upon receiving a connection",
			notify:      common.SensorComponentEventOfflineMode,
			connections: []*HostnameAndConnections{createHostnameConnections(hostname).withConnectionPair(createConnectionPair())},
		},
		{
			testName: "In online mode we should enrich and send the previously received connection",
			notify:   common.SensorComponentEventResourceSyncFinished,
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntity, 1, clusterentities.ContainerMetadata{
				DeploymentID: srcID,
			}, true, false),
			expectEntityLookupEndpoint: expectEntityLookupEndpointHelper(mockEntity, 1, []clusterentities.LookupResult{
				{
					Entity:         networkgraph.Entity{ID: dstID},
					ContainerPorts: []uint16{80},
				},
			}),
			expectDetector:        expectDetectorHelper(mockDetector, 1),
			expectedSensorMessage: createExpectedSensorMessageWithConnections(&expectedEntitiesPair{srcID: srcID, dstID: dstID}),
		},
		{
			testName: "In offline mode we should not send any messages upon receiving multiple connections",
			notify:   common.SensorComponentEventOfflineMode,
			connections: []*HostnameAndConnections{
				createHostnameConnections(hostname).withConnectionPair(createConnectionPair().containerID(fmt.Sprintf("%s-1", containerID))),
				createHostnameConnections(hostname).withConnectionPair(createConnectionPair().containerID(fmt.Sprintf("%s-2", containerID))),
			},
		},
		{
			testName: "In online mode we should enrich and send the previously received connections",
			notify:   common.SensorComponentEventResourceSyncFinished,
			expectEntityLookupContainer: func() {
				gomock.InOrder(
					mockEntity.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
						return clusterentities.ContainerMetadata{DeploymentID: fmt.Sprintf("%s-1", srcID)}, true, false
					}),
					mockEntity.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
						return clusterentities.ContainerMetadata{DeploymentID: fmt.Sprintf("%s-2", srcID)}, true, false

					}),
				)
			},
			expectEntityLookupEndpoint: func() {
				gomock.InOrder(
					mockEntity.EXPECT().LookupByEndpoint(gomock.Any()).Times(1).DoAndReturn(func(_ any) []clusterentities.LookupResult {
						return []clusterentities.LookupResult{
							{
								Entity:         networkgraph.Entity{ID: fmt.Sprintf("%s-1", dstID)},
								ContainerPorts: []uint16{80},
							},
						}
					}),
					mockEntity.EXPECT().LookupByEndpoint(gomock.Any()).Times(1).DoAndReturn(func(_ any) []clusterentities.LookupResult {
						return []clusterentities.LookupResult{
							{
								Entity:         networkgraph.Entity{ID: fmt.Sprintf("%s-2", dstID)},
								ContainerPorts: []uint16{80},
							},
						}
					}),
				)
			},
			expectDetector: expectDetectorHelper(mockDetector, 2),
			expectedSensorMessage: createExpectedSensorMessageWithConnections(
				&expectedEntitiesPair{srcID: fmt.Sprintf("%s-1", srcID), dstID: fmt.Sprintf("%s-1", dstID)},
				&expectedEntitiesPair{srcID: fmt.Sprintf("%s-2", srcID), dstID: fmt.Sprintf("%s-2", dstID)},
			),
		},
		{
			testName: "In offline mode we should not send any messages upon receiving multiple endpoints",
			notify:   common.SensorComponentEventOfflineMode,
			connections: []*HostnameAndConnections{
				createHostnameConnections(hostname).withEndpointPair(createEndpointPair(timestamp.Now(), timestamp.Now()).containerID(fmt.Sprintf("%s-1", containerID))),
				createHostnameConnections(hostname).withEndpointPair(createEndpointPair(timestamp.Now(), timestamp.Now()).containerID(fmt.Sprintf("%s-2", containerID))),
			},
		},
		{
			testName: "In online mode we should enrich and send the previously received endpoints",
			notify:   common.SensorComponentEventResourceSyncFinished,
			expectEntityLookupContainer: func() {
				gomock.InOrder(
					mockEntity.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
						return clusterentities.ContainerMetadata{DeploymentID: fmt.Sprintf("%s-1", srcID)}, true, false
					}),
					mockEntity.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
						return clusterentities.ContainerMetadata{DeploymentID: fmt.Sprintf("%s-2", srcID)}, true, false
					}),
				)
			},
			expectedSensorMessage: createExpectedSensorMessageWithEndpoints(
				fmt.Sprintf("%s-1", srcID),
				fmt.Sprintf("%s-2", srcID),
			),
		},
	}
	fakeTicker := make(chan time.Time)
	defer close(fakeTicker)
	go m.enrichConnections(fakeTicker)
	// The test cases are supposed to be run in order!
	for _, state := range states {
		s.Run(state.testName, func() {
			for _, cnn := range state.connections {
				addHostConnection(m, cnn)
			}
			state.expectEntityLookupContainer.runIfSet()
			state.expectEntityLookupEndpoint.runIfSet()
			state.expectDetector.runIfSet()
			// We do not test ticking here, but without this line, the test would deadlock.
			mockEntity.EXPECT().RecordTick().AnyTimes()
			m.Notify(state.notify)
			fakeTicker <- time.Now()
			if state.expectedSensorMessage != nil {
				select {
				case <-time.After(10 * time.Second):
					s.Fail("timeout waiting for sensor message")
				case msg, ok := <-m.sensorUpdates:
					s.Require().True(ok, "the sensorUpdates channel should not be closed")
					s.Assert().NotNil(msg)
					msgFromSensor, ok := msg.Msg.(*central.MsgFromSensor_NetworkFlowUpdate)
					s.Require().True(ok, "the message received is not a NetworkFlowUpdate message")
					expectedMsg, ok := state.expectedSensorMessage.Msg.(*central.MsgFromSensor_NetworkFlowUpdate)
					s.Require().True(ok, "the message expected is not a NetworkFlowUpdate message")
					s.Assert().Len(msgFromSensor.NetworkFlowUpdate.GetUpdated(), len(expectedMsg.NetworkFlowUpdate.GetUpdated()))
					s.assertSensorMessageConnectionIDs(expectedMsg.NetworkFlowUpdate.GetUpdated(), msgFromSensor.NetworkFlowUpdate.GetUpdated())
					s.Assert().Len(msgFromSensor.NetworkFlowUpdate.GetUpdatedEndpoints(), len(expectedMsg.NetworkFlowUpdate.GetUpdatedEndpoints()))
					s.assertSensorMessageEndpointIDs(expectedMsg.NetworkFlowUpdate.GetUpdatedEndpoints(), msgFromSensor.NetworkFlowUpdate.GetUpdatedEndpoints())
				}
			} else {
				select {
				case _, ok := <-m.sensorUpdates:
					s.Require().True(ok, "the sensorUpdates channel should not be closed")
					s.Fail("should not received message in sensorUpdates channel")
				case <-time.After(time.Second):
					break
				}
			}
		})
	}
	m.Stop()
}

func (s *NetworkFlowManagerTestSuite) TestExpireMessage() {
	// This test is for v1/v2 behavior
	s.T().Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), "false")
	s.T().Setenv(env.ProcessesListeningOnPort.EnvVar(), "false")
	hostname := "hostname"
	containerID := "container-id"

	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer mockCtrl.Finish()
	m, mockEntity, _, mockDetector := createManager(mockCtrl, enrichTickerC)
	go m.enrichConnections(enrichTickerC)
	mockEntity.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
		return clusterentities.ContainerMetadata{
			DeploymentID: containerID,
		}, true, false
	})
	mockEntity.EXPECT().LookupByEndpoint(gomock.Any()).Times(1).DoAndReturn(func(_ any) []clusterentities.LookupResult {
		return []clusterentities.LookupResult{
			{
				Entity:         networkgraph.Entity{ID: containerID},
				ContainerPorts: []uint16{80},
			},
		}
	})
	mockDetector.EXPECT().ProcessNetworkFlow(gomock.Any(), gomock.Any()).Times(1)
	mockEntity.EXPECT().RecordTick().AnyTimes()
	addHostConnection(m, createHostnameConnections(hostname).withConnectionPair(createConnectionPair()))
	m.Notify(common.SensorComponentEventResourceSyncFinished)

	select {
	case <-time.After(10 * time.Second):
		s.Fail("enrichTickerC blocks!")
	case enrichTickerC <- time.Now():
	}
	select {
	case <-time.After(10 * time.Second):
		s.Fail("timeout waiting for sensor message")
	case msg, ok := <-m.sensorUpdates:
		s.Require().True(ok, "the sensorUpdates channel should not be closed")
		m.Notify(common.SensorComponentEventOfflineMode)
		m.Notify(common.SensorComponentEventResourceSyncFinished)
		s.Assert().True(msg.IsExpired(), "the message should be expired")
	}
	m.Stop()
}

// endregion

func Test_connection_IsExternal(t *testing.T) {
	tests := map[string]struct {
		remoteIP         string
		remoteCIDR       string
		expectedExternal bool
		wantErr          bool
	}{
		"10.0.0.1 IP address should be internal": {
			remoteIP:         "10.0.0.1",
			remoteCIDR:       "",
			expectedExternal: false,
			wantErr:          false,
		},
		"169.254.1.1 IP address should be internal": {
			remoteIP:         "169.254.1.1",
			remoteCIDR:       "",
			expectedExternal: false,
			wantErr:          false,
		},
		"127.0.0.1 localhost should return an error (and not be shown on the graph)": {
			remoteIP:         "127.0.0.1",
			remoteCIDR:       "",
			expectedExternal: false,
			wantErr:          true,
		},
		"192.168.1.1 IP address should be internal": {
			remoteIP:         "192.168.1.1",
			remoteCIDR:       "",
			expectedExternal: false,
			wantErr:          false,
		},
		"11.12.13.14 IP address should be external": {
			remoteIP:         "11.12.13.14",
			remoteCIDR:       "",
			expectedExternal: true,
			wantErr:          false,
		},
		"8.8.8.8 IP address should be external": {
			remoteIP:         "8.8.8.8",
			remoteCIDR:       "",
			expectedExternal: true,
			wantErr:          false,
		},
		"10.0.0.0/8 Network should be internal": {
			remoteIP:         "",
			remoteCIDR:       "10.0.0.0/8",
			expectedExternal: false,
			wantErr:          false,
		},
		// 10.0.0.0/6 contains entire 10.0.0.0/8 (which is internal) and additional address range that is external
		"10.0.0.0/6 Network should be external": {
			remoteIP:         "",
			remoteCIDR:       "10.0.0.0/6",
			expectedExternal: true,
			wantErr:          false,
		},
		"169.254.1.0/24 Network should be internal": {
			remoteIP:         "",
			remoteCIDR:       "169.254.1.0/24",
			expectedExternal: false,
			wantErr:          false,
		},
		"192.168.1.0/24 Network should be internal": {
			remoteIP:         "",
			remoteCIDR:       "192.168.1.0/24",
			expectedExternal: false,
			wantErr:          false,
		},
		"11.12.13.2/30 Network should be external": {
			remoteIP:         "",
			remoteCIDR:       "11.12.13.2/30",
			expectedExternal: true,
			wantErr:          false,
		},
		"8.8.8.8/32 Network should be external": {
			remoteIP:         "",
			remoteCIDR:       "8.8.8.8/32",
			expectedExternal: true,
			wantErr:          false,
		},
		"IP address should have precedence over Network CIDR": {
			remoteIP:         "192.168.1.1", // internal
			remoteCIDR:       "8.8.8.8/32",  // external
			expectedExternal: false,
			wantErr:          false,
		},
		"Both empty should yield external and an error": {
			remoteIP:         "",
			remoteCIDR:       "",
			expectedExternal: true,
			wantErr:          true,
		},
		"fd00::/8 Network should be internal": {
			remoteIP:         "",
			remoteCIDR:       "fd00::/8",
			expectedExternal: false,
			wantErr:          false,
		},
		"fe80::/10 Network should be internal": {
			remoteIP:         "",
			remoteCIDR:       "fe80::/10",
			expectedExternal: false,
			wantErr:          false,
		},
		"fd12:3456:789a:1::1 (Unique Local Addresses) should be internal": {
			remoteIP:         "fd12:3456:789a:1::1",
			remoteCIDR:       "",
			expectedExternal: false,
			wantErr:          false,
		},
		"::1 IP address (localhost) should return an error (and not be shown on the graph)": {
			remoteIP:         "::1",
			remoteCIDR:       "",
			expectedExternal: false,
			wantErr:          true,
		},
		"::1/128 (localhost) should return an error (and not be shown on the graph)": {
			remoteIP:         "",
			remoteCIDR:       "::1/128",
			expectedExternal: false,
			wantErr:          true,
		},
		"255.255.255.255 is a special value returned from collector and should be treated as external": {
			remoteIP:         "255.255.255.255",
			remoteCIDR:       "",
			expectedExternal: true,
			wantErr:          false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &connection{
				remote: net.NumericEndpoint{
					IPAndPort: net.NetworkPeerID{
						Address:   net.ParseIP(tt.remoteIP),
						Port:      80,
						IPNetwork: net.IPNetworkFromCIDR(tt.remoteCIDR),
					},
				},
			}
			got, err := c.IsExternal()
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			assert.Equalf(t, tt.expectedExternal, got, "expected %q/%q to be external=%t, but got=%t",
				tt.remoteIP, tt.remoteCIDR, tt.expectedExternal, got)
		})
	}
}

func Test_getConnection_direction(t *testing.T) {
	tests := []struct {
		localPort  uint32
		remotePort uint32
		protocol   storage.L4Protocol
		role       sensor.ClientServerRole
		expected   bool
	}{
		{
			localPort:  0,
			remotePort: 53,
			protocol:   storage.L4Protocol_L4_PROTOCOL_UDP,
			role:       sensor.ClientServerRole_ROLE_CLIENT,
			expected:   false,
		}, {
			localPort:  0,
			remotePort: 53,
			protocol:   storage.L4Protocol_L4_PROTOCOL_UDP,
			role:       sensor.ClientServerRole_ROLE_SERVER,
			expected:   false,
		}, {
			localPort:  53,
			remotePort: 0,
			protocol:   storage.L4Protocol_L4_PROTOCOL_UDP,
			role:       sensor.ClientServerRole_ROLE_CLIENT,
			expected:   true,
		}, {
			localPort:  53,
			remotePort: 0,
			protocol:   storage.L4Protocol_L4_PROTOCOL_UDP,
			role:       sensor.ClientServerRole_ROLE_SERVER,
			expected:   true,
		}, {
			// Having both ports set to 0 should be impossible, it means we've
			// failed to get lots of information about the connection and is
			// more likely the event will not leave collector.
			localPort:  0,
			remotePort: 0,
			protocol:   storage.L4Protocol_L4_PROTOCOL_UDP,
			role:       sensor.ClientServerRole_ROLE_CLIENT,
			expected:   false,
		}, {
			localPort:  0,
			remotePort: 0,
			protocol:   storage.L4Protocol_L4_PROTOCOL_UDP,
			role:       sensor.ClientServerRole_ROLE_SERVER,
			expected:   false,
		}, {
			localPort:  50000,
			remotePort: 53,
			protocol:   storage.L4Protocol_L4_PROTOCOL_UDP,
			role:       sensor.ClientServerRole_ROLE_CLIENT,
			expected:   false,
		}, {
			localPort:  50000,
			remotePort: 53,
			protocol:   storage.L4Protocol_L4_PROTOCOL_UDP,
			role:       sensor.ClientServerRole_ROLE_SERVER,
			expected:   false,
		}, {
			localPort:  53,
			remotePort: 50000,
			protocol:   storage.L4Protocol_L4_PROTOCOL_UDP,
			role:       sensor.ClientServerRole_ROLE_CLIENT,
			expected:   true,
		}, {
			localPort:  53,
			remotePort: 50000,
			protocol:   storage.L4Protocol_L4_PROTOCOL_UDP,
			role:       sensor.ClientServerRole_ROLE_SERVER,
			expected:   true,
		}, {
			localPort:  50000,
			remotePort: 53,
			protocol:   storage.L4Protocol_L4_PROTOCOL_TCP,
			role:       sensor.ClientServerRole_ROLE_CLIENT,
			expected:   false,
		}, {
			localPort:  50000,
			remotePort: 53,
			protocol:   storage.L4Protocol_L4_PROTOCOL_TCP,
			role:       sensor.ClientServerRole_ROLE_SERVER,
			expected:   true,
		}, {
			localPort:  53,
			remotePort: 50000,
			protocol:   storage.L4Protocol_L4_PROTOCOL_TCP,
			role:       sensor.ClientServerRole_ROLE_CLIENT,
			expected:   false,
		}, {
			localPort:  53,
			remotePort: 50000,
			protocol:   storage.L4Protocol_L4_PROTOCOL_TCP,
			role:       sensor.ClientServerRole_ROLE_SERVER,
			expected:   true,
		},
	}

	for _, tt := range tests {
		networkConn := sensor.NetworkConnection{
			SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
			LocalAddress: &sensor.NetworkAddress{
				Port: tt.localPort,
			},
			RemoteAddress: &sensor.NetworkAddress{
				Port: tt.remotePort,
			},
			Protocol: tt.protocol,
			Role:     tt.role,
		}
		conn, err := processConnection(&networkConn)
		assert.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Equal(t, tt.expected, conn.incoming, "local: %d, remote: %d, protocol: %s, role: %s", tt.localPort, tt.remotePort, storage.L4Protocol_name[int32(tt.protocol)], sensor.ClientServerRole_name[int32(tt.role)])
	}
}
