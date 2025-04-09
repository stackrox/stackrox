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
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	mocksManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
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

const (
	waitTimeout = 20 * time.Millisecond
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

// endregion

// region networkFlowManager tests

func (s *NetworkFlowManagerTestSuite) TestEnrichConnection() {
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
	defer mockCtrl.Finish()
	m, mockEntityStore, mockExternalSrc, _ := createManager(mockCtrl, enrichTickerC, purgerTickerC)
	srcID := "src-id"
	dstID := "dst-id"
	cases := map[string]struct {
		connPair                    *connectionPair
		enrichedConnections         map[networkConnIndicator]timestamp.MicroTS
		expectEntityLookupContainer expectFn
		expectEntityLookupEndpoint  expectFn
		expectExternalLookup        expectFn
		expectedIndicator           *networkConnIndicator
		expectedConnection          *connection
		expectedStatus              *connStatus
	}{
		"Rotten connection should return rotten status": {
			connPair: createConnectionPair().incoming().external().firstSeen(timestamp.Now().Add(-maxContainerResolutionWaitPeriod * 2)),
			expectEntityLookupContainer: func() {
				mockEntityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
					return clusterentities.ContainerMetadata{}, false, false
				})
			},
			expectedStatus: &connStatus{
				rotten: true,
			},
		},
		"Incoming external connection with unsuccessful lookup should return internet entity": {
			connPair:            createConnectionPair().incoming().external(),
			enrichedConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: func() {
				mockEntityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
					return clusterentities.ContainerMetadata{
						DeploymentID: dstID,
					}, true, false
				})
			},
			expectExternalLookup: func() {
				mockExternalSrc.EXPECT().LookupByNetwork(gomock.Any()).Times(1).DoAndReturn(func(_ any) *storage.NetworkEntityInfo {
					return nil
				})
			},
			expectedStatus: &connStatus{
				used: true,
			},
			expectedIndicator: &networkConnIndicator{
				dstPort:   80,
				protocol:  net.TCP.ToProtobuf(),
				srcEntity: networkgraph.InternetEntity(),
				dstEntity: networkgraph.EntityForDeployment(dstID),
			},
		},
		"Outgoing external connection with successful external lookup should return the correct id": {
			connPair:            createConnectionPair().external(),
			enrichedConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: func() {
				mockEntityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
					return clusterentities.ContainerMetadata{
						DeploymentID: srcID,
					}, true, false
				})
			},
			expectExternalLookup: func() {
				mockExternalSrc.EXPECT().LookupByNetwork(gomock.Any()).Times(1).DoAndReturn(func(_ any) *storage.NetworkEntityInfo {
					return &storage.NetworkEntityInfo{
						Id: dstID,
					}
				})

			},
			expectedStatus: &connStatus{
				used: true,
			},
			expectedIndicator: &networkConnIndicator{
				dstPort:  80,
				protocol: net.TCP.ToProtobuf(),
				dstEntity: networkgraph.EntityFromProto(&storage.NetworkEntityInfo{
					Id: dstID,
				}),
				srcEntity: networkgraph.EntityForDeployment(srcID),
			},
		},
		"Incoming connection with successful lookup should not return a networkConnIndicator": {
			connPair:            createConnectionPair().incoming(),
			enrichedConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: func() {
				mockEntityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
					return clusterentities.ContainerMetadata{
						DeploymentID: srcID,
					}, true, false
				})
			},
			expectEntityLookupEndpoint: func() {
				mockEntityStore.EXPECT().LookupByEndpoint(gomock.Any()).Times(1).DoAndReturn(func(_ any) []clusterentities.LookupResult {
					return []clusterentities.LookupResult{
						{
							Entity: networkgraph.Entity{
								ID: dstID,
							},
						},
					}
				})
			},
			expectedStatus: &connStatus{
				used: true,
			},
		},
		"Incoming fresh connection with valid address should not return anything": {
			connPair:            createConnectionPair().incoming(),
			enrichedConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: func() {
				mockEntityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
					return clusterentities.ContainerMetadata{
						DeploymentID: dstID,
					}, true, false
				})
			},
			expectEntityLookupEndpoint: func() {
				mockEntityStore.EXPECT().LookupByEndpoint(gomock.Any()).Times(1).DoAndReturn(func(_ any) []clusterentities.LookupResult {
					return nil
				})
			},
			expectedStatus: &connStatus{},
		},
		"Incoming fresh connection with invalid address should not return anything": {
			connPair:            createConnectionPair().incoming().invalidAddress(),
			enrichedConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: func() {
				mockEntityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
					return clusterentities.ContainerMetadata{
						DeploymentID: dstID,
					}, true, false
				})
			},
			expectEntityLookupEndpoint: func() {
				mockEntityStore.EXPECT().LookupByEndpoint(gomock.Any()).Times(1).DoAndReturn(func(_ any) []clusterentities.LookupResult {
					return nil
				})
			},
			expectExternalLookup: func() {
				mockExternalSrc.EXPECT().LookupByNetwork(gomock.Any()).Times(1).DoAndReturn(func(_ any) *storage.NetworkEntityInfo {
					return nil
				})
			},
			expectedStatus: &connStatus{},
		},
		"Outgoing connection with successful internal lookup should return the correct id": {
			connPair:            createConnectionPair(),
			enrichedConnections: make(map[networkConnIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: func() {
				mockEntityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
					return clusterentities.ContainerMetadata{
						DeploymentID: srcID,
					}, true, false
				})
			},
			expectEntityLookupEndpoint: func() {
				mockEntityStore.EXPECT().LookupByEndpoint(gomock.Any()).Times(1).DoAndReturn(func(_ any) []clusterentities.LookupResult {
					return []clusterentities.LookupResult{
						{
							Entity: networkgraph.Entity{
								ID: dstID,
							},
							ContainerPorts: []uint16{
								80,
							},
						},
					}
				})
			},
			expectedStatus: &connStatus{
				used: true,
			},
			expectedIndicator: &networkConnIndicator{
				dstPort:  80,
				protocol: net.TCP.ToProtobuf(),
				dstEntity: networkgraph.EntityFromProto(&storage.NetworkEntityInfo{
					Id: dstID,
				}),
				srcEntity: networkgraph.EntityForDeployment(srcID),
			},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			tCase.expectEntityLookupContainer.runIfSet()
			tCase.expectEntityLookupEndpoint.runIfSet()
			tCase.expectExternalLookup.runIfSet()
			m.enrichConnection(tCase.connPair.conn, tCase.connPair.status, tCase.enrichedConnections)
			s.Assert().Equal(tCase.expectedStatus.used, tCase.connPair.status.used)
			s.Assert().Equal(tCase.expectedStatus.rotten, tCase.connPair.status.rotten)
			if tCase.expectedIndicator != nil {
				_, ok := tCase.enrichedConnections[*tCase.expectedIndicator]
				s.Assert().True(ok)
			} else {
				s.Assert().Len(tCase.enrichedConnections, 0)
			}
		})
	}
}

func (s *NetworkFlowManagerTestSuite) TestEnrichContainerEndpoint() {
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
	defer mockCtrl.Finish()
	m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC, purgerTickerC)
	id := "id"
	_ = id
	cases := map[string]struct {
		endpointPair                *endpointPair
		enrichedConnections         map[containerEndpointIndicator]timestamp.MicroTS
		expectEntityLookupContainer expectFn
		expectedStatus              *connStatus
		expectedEndpoint            *containerEndpointIndicator
	}{
		"Rotten connection should return rotten status": {
			endpointPair:                createEndpointPair(timestamp.Now().Add(-maxContainerResolutionWaitPeriod*2), timestamp.Now()),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{}, false, false),
			expectedStatus: &connStatus{
				rotten: true,
				used:   true,
			},
		},
		"Container endpoint should return an containerEndpointIndicator with the correct id": {
			endpointPair:        createEndpointPair(timestamp.Now(), timestamp.Now()),
			enrichedConnections: make(map[containerEndpointIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{
				DeploymentID: id,
			}, true, false),
			expectedStatus: &connStatus{used: true},
			expectedEndpoint: &containerEndpointIndicator{
				entity:   networkgraph.EntityForDeployment(id),
				port:     80,
				protocol: net.TCP.ToProtobuf(),
			},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			tCase.expectEntityLookupContainer.runIfSet()
			m.enrichContainerEndpoint(tCase.endpointPair.endpoint, tCase.endpointPair.status, tCase.enrichedConnections, timestamp.Now())
			s.Assert().Equal(tCase.expectedStatus.rotten, tCase.endpointPair.status.rotten)
			s.Assert().Equal(tCase.expectedStatus.used, tCase.endpointPair.status.used)
			if tCase.expectedEndpoint != nil {
				_, ok := tCase.enrichedConnections[*tCase.expectedEndpoint]
				s.Assert().True(ok)
			}
		})
	}
}

func (s *NetworkFlowManagerTestSuite) TestEndpointPurger() {
	const hostname = "host"
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
	defer mockCtrl.Finish()
	m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC, purgerTickerC)
	id := "id"
	_ = id
	cases := map[string]struct {
		firstSeen            time.Duration
		lastUpdateTime       time.Duration
		purgerMaxAge         time.Duration
		isKnownEndpoint      bool
		expectedStatus       *connStatus
		expectedEndpoint     *containerEndpointIndicator
		expectedHostConnSize int
	}{
		"Purger maxAge: should purge old endpoints": {
			firstSeen:            2 * time.Hour,
			lastUpdateTime:       2 * time.Hour,
			purgerMaxAge:         time.Hour,
			isKnownEndpoint:      true,
			expectedHostConnSize: 0,
		},
		"Purger maxAge: should keep endpoints with young lastUpdateTime": {
			firstSeen:            time.Minute,
			lastUpdateTime:       time.Minute,
			purgerMaxAge:         time.Hour,
			isKnownEndpoint:      true,
			expectedHostConnSize: 1,
		},
		"Purger endpoint-gone: should remove unknown endpoints": {
			firstSeen:            time.Minute,
			lastUpdateTime:       time.Minute,
			purgerMaxAge:         time.Hour,
			isKnownEndpoint:      false,
			expectedHostConnSize: 1,
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			now := time.Now()
			lastUpdateTS := timestamp.FromGoTime(now.Add(-tc.lastUpdateTime))
			expectationsEndpointPurger(mockEntityStore, tc.isKnownEndpoint, true, false)
			ep := createEndpointPair(timestamp.FromGoTime(now.Add(-tc.firstSeen)), lastUpdateTS)
			concurrency.WithLock(&m.connectionsByHostMutex, func() {
				m.connectionsByHost[hostname] = &hostConnections{
					hostname:    hostname,
					connections: nil,
					endpoints: map[containerEndpoint]*connStatus{
						*ep.endpoint: ep.status,
					},
				}
			})
			// Purger checks activeEndpoints only if not empty, so let's make sure that
			// the mock is called correct number of times by always having one active endpoint.
			m.activeEndpoints[*ep.endpoint] = &containerEndpointIndicatorWithAge{
				containerEndpointIndicator: containerEndpointIndicator{
					entity:   networkgraph.Entity{},
					port:     80,
					protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				},
				lastUpdate: lastUpdateTS,
			}
			m.runAllPurgerRules(tc.purgerMaxAge)

			concurrency.WithLock(&m.connectionsByHostMutex, func() {
				s.Len(m.connectionsByHost[hostname].endpoints, tc.expectedHostConnSize)
			})
		})
	}
}

func (s *NetworkFlowManagerTestSuite) TestEnrichProcessListening() {
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
	defer mockCtrl.Finish()
	m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC, purgerTickerC)
	deploymentID := "deployment-id"
	podID := "pod-id"
	cases := map[string]struct {
		containerPair               *containerPair
		enrichedConnections         map[processListeningIndicator]timestamp.MicroTS
		expectEntityLookupContainer expectFn
		expectedStatus              *connStatus
		expectedListeningIndicator  *processListeningIndicator
	}{
		"Rotten connection should return rotten status": {
			containerPair:               createContainerPair(timestamp.Now().Add(-maxContainerResolutionWaitPeriod * 2)),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{}, false, false),
			expectedStatus: &connStatus{
				rotten:      true,
				usedProcess: true,
			},
		},
		"Container endpoint should return a processListeningIndicator with the correct id": {
			containerPair:       createContainerPair(timestamp.Now()),
			enrichedConnections: make(map[processListeningIndicator]timestamp.MicroTS),
			expectEntityLookupContainer: expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{
				DeploymentID:  deploymentID,
				ContainerName: "container-name",
				PodID:         podID,
			}, true, false),
			expectedStatus: &connStatus{
				usedProcess: true,
			},
			expectedListeningIndicator: &processListeningIndicator{
				key: processUniqueKey{
					podID:         podID,
					containerName: "container-name",
					deploymentID:  deploymentID,
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
			m.enrichProcessListening(tCase.containerPair.endpoint, tCase.containerPair.status, tCase.enrichedConnections)
			s.Assert().Equal(tCase.expectedStatus.rotten, tCase.containerPair.status.rotten)
			s.Assert().Equal(tCase.expectedStatus.usedProcess, tCase.containerPair.status.usedProcess)
			if tCase.expectedListeningIndicator != nil {
				_, ok := tCase.enrichedConnections[*tCase.expectedListeningIndicator]
				s.Assert().True(ok)
			}
		})
	}
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
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
	defer mockCtrl.Finish()
	m, mockEntity, _, mockDetector := createManager(mockCtrl, enrichTickerC, purgerTickerC)
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
	m.Stop(nil)
}

func (s *NetworkFlowManagerTestSuite) TestExpireMessage() {
	// This test is for v1/v2 behavior
	s.T().Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), "false")
	s.T().Setenv(env.ProcessesListeningOnPort.EnvVar(), "false")
	hostname := "hostname"
	containerID := "container-id"

	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
	defer mockCtrl.Finish()
	m, mockEntity, _, mockDetector := createManager(mockCtrl, enrichTickerC, purgerTickerC)
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
	enrichTickerC <- time.Now()
	select {
	case <-time.After(10 * time.Second):
		s.Fail("timeout waiting for sensor message")
	case msg, ok := <-m.sensorUpdates:
		s.Require().True(ok, "the sensorUpdates channel should not be closed")
		m.Notify(common.SensorComponentEventOfflineMode)
		m.Notify(common.SensorComponentEventResourceSyncFinished)
		s.Assert().True(msg.IsExpired(), "the message should be expired")
	}
	m.Stop(nil)
}

func TestSendNetworkFlows(t *testing.T) {
	t.Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), "true")
	suite.Run(t, new(sendNetflowsSuite))
}

type sendNetflowsSuite struct {
	suite.Suite
	mockCtrl     *gomock.Controller
	mockEntity   *mocksManager.MockEntityStore
	m            *networkFlowManager
	mockDetector *mocksDetector.MockDetector
	fakeTicker   chan time.Time
}

const (
	srcID = "src-id"
	dstID = "dst-id"
)

func (b *sendNetflowsSuite) SetupTest() {
	b.mockCtrl = gomock.NewController(b.T())
	enrichTickerC := make(chan time.Time)
	purgerTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer close(purgerTickerC)
	b.m, b.mockEntity, _, b.mockDetector = createManager(b.mockCtrl, enrichTickerC, purgerTickerC)

	b.fakeTicker = make(chan time.Time)
	go b.m.enrichConnections(b.fakeTicker)
}

func (b *sendNetflowsSuite) TeardownTest() {
	b.m.stopper.Client().Stop()
}

func (b *sendNetflowsSuite) updateConn(pair *connectionPair) {
	addHostConnection(b.m, createHostnameConnections("hostname").withConnectionPair(pair))
}

func (b *sendNetflowsSuite) updateEp(pair *endpointPair) {
	addHostConnection(b.m, createHostnameConnections("hostname").withEndpointPair(pair))
}

func (b *sendNetflowsSuite) expectContainerLookups(n int) {
	b.mockEntity.EXPECT().RecordTick().AnyTimes()
	expectEntityLookupContainerHelper(b.mockEntity, n, clusterentities.ContainerMetadata{
		DeploymentID: srcID,
	}, true, false)()
}

func (b *sendNetflowsSuite) expectLookups(n int) {
	b.mockEntity.EXPECT().RecordTick().AnyTimes()
	expectEntityLookupContainerHelper(b.mockEntity, n, clusterentities.ContainerMetadata{
		DeploymentID: srcID,
	}, true, false)()
	expectEntityLookupEndpointHelper(b.mockEntity, n, []clusterentities.LookupResult{
		{
			Entity:         networkgraph.Entity{ID: dstID},
			ContainerPorts: []uint16{80},
		},
	})()
}

func (b *sendNetflowsSuite) expectFailedLookup(n int) {
	b.mockEntity.EXPECT().RecordTick().AnyTimes()
	expectEntityLookupContainerHelper(b.mockEntity, n, clusterentities.ContainerMetadata{}, false, false)()
}

func (b *sendNetflowsSuite) expectDetections(n int) {
	expectDetectorHelper(b.mockDetector, n)()
}

func (b *sendNetflowsSuite) TestUpdateConnectionGeneratesNetflow() {
	b.expectLookups(1)
	b.expectDetections(1)

	b.updateConn(createConnectionPair())
	b.thenTickerTicks()
	b.assertOneUpdatedOpenConnection()
}

func (b *sendNetflowsSuite) TestCloseConnection() {
	b.expectLookups(1)
	b.expectDetections(1)

	b.updateConn(createConnectionPair().lastSeen(timestamp.Now()))
	b.thenTickerTicks()
	b.assertOneUpdatedCloseConnection()
}

func (b *sendNetflowsSuite) TestCloseConnectionFailedLookup() {
	b.expectFailedLookup(1)

	b.updateConn(createConnectionPair().lastSeen(timestamp.Now()))
	b.thenTickerTicks()
	mustNotRead(b.T(), b.m.sensorUpdates)
}

func (b *sendNetflowsSuite) TestCloseOldConnectionFailedLookup() {
	b.expectFailedLookup(1)
	b.expectDetections(1)

	pair := createConnectionPair().
		firstSeen(timestamp.Now().Add(-maxContainerResolutionWaitPeriod * 2)).
		lastSeen(timestamp.Now())
	b.m.activeConnections[*pair.conn] = &networkConnIndicatorWithAge{}
	b.updateConn(pair)
	b.thenTickerTicks()
	b.assertOneUpdatedCloseConnection()
}

func (b *sendNetflowsSuite) TestCloseEndpoint() {
	b.expectContainerLookups(1)

	b.updateEp(createEndpointPair(timestamp.Now().Add(-time.Hour), timestamp.Now()).lastSeen(timestamp.Now()))
	b.thenTickerTicks()
	b.assertOneUpdatedCloseEndpoint()
}

func (b *sendNetflowsSuite) TestCloseEndpointFailedLookup() {
	b.expectFailedLookup(1)

	b.updateEp(createEndpointPair(timestamp.Now().Add(-time.Hour), timestamp.Now()).lastSeen(timestamp.Now()))
	b.thenTickerTicks()
	mustNotRead(b.T(), b.m.sensorUpdates)
}

func (b *sendNetflowsSuite) TestCloseOldEndpointFailedLookup() {
	b.expectFailedLookup(1)

	pair := createEndpointPair(
		timestamp.Now().Add(-maxContainerResolutionWaitPeriod*2), timestamp.Now()).
		lastSeen(timestamp.Now())
	b.m.activeEndpoints[*pair.endpoint] = &containerEndpointIndicatorWithAge{}
	b.updateEp(pair)
	b.thenTickerTicks()
	b.assertOneUpdatedCloseEndpoint()
}

func (b *sendNetflowsSuite) TestUnchangedConnection() {
	b.expectLookups(2)
	b.expectDetections(1)

	b.updateConn(createConnectionPair().lastSeen(timestamp.InfiniteFuture))
	b.thenTickerTicks()
	b.assertOneUpdatedOpenConnection()

	// There should be no second update, the connection did not change
	b.thenTickerTicks()
	mustNotRead(b.T(), b.m.sensorUpdates)
}

func (b *sendNetflowsSuite) TestSendTwoUpdatesOnConnectionChanged() {
	b.expectLookups(2)
	b.expectDetections(2)

	pair := createConnectionPair()
	b.updateConn(pair.lastSeen(timestamp.FromProtobuf(protoconv.NowMinus(time.Hour))))
	b.thenTickerTicks()
	b.assertOneUpdatedCloseConnection()

	pair.lastSeen(timestamp.Now())
	b.updateConn(pair)
	b.thenTickerTicks()
	b.assertOneUpdatedCloseConnection()
}

func (b *sendNetflowsSuite) TestUpdatesGetBufferedWhenUnread() {
	b.expectLookups(4)
	b.expectDetections(4)

	// four times without reading
	for i := 4; i > 0; i-- {
		ts := protoconv.NowMinus(time.Duration(i) * time.Hour)
		b.updateConn(createConnectionPair().lastSeen(timestamp.FromProtobuf(ts)))
		b.thenTickerTicks()
		time.Sleep(100 * time.Millisecond) // Immediately ticking without waiting causes unexpected behavior
	}

	// should be able to read four buffered updates in sequence
	for i := 0; i < 4; i++ {
		b.assertOneUpdatedCloseConnection()
	}
}

func (b *sendNetflowsSuite) TestCallsDetectionEvenOnFullBuffer() {
	b.expectLookups(6)
	b.expectDetections(6)

	for i := 6; i > 0; i-- {
		ts := protoconv.NowMinus(time.Duration(i) * time.Hour)
		b.updateConn(createConnectionPair().lastSeen(timestamp.FromProtobuf(ts)))
		b.thenTickerTicks()
		time.Sleep(100 * time.Millisecond)
	}

	// Will only store 5 network flow updates, as it's the maximum buffer size in the test
	for i := 0; i < 5; i++ {
		b.assertOneUpdatedCloseConnection()
	}

	mustNotRead(b.T(), b.m.sensorUpdates)
}

func (b *sendNetflowsSuite) thenTickerTicks() {
	mustSendWithoutBlock(b.T(), b.fakeTicker, time.Now())
}

func (b *sendNetflowsSuite) assertOneUpdatedOpenConnection() {
	msg := mustReadTimeout(b.T(), b.m.sensorUpdates)
	netflowUpdate, ok := msg.Msg.(*central.MsgFromSensor_NetworkFlowUpdate)
	b.Require().True(ok, "message is NetworkFlowUpdate")
	b.Require().Len(netflowUpdate.NetworkFlowUpdate.GetUpdated(), 1, "one updated connection")
	b.Assert().Equal(int32(0), netflowUpdate.NetworkFlowUpdate.GetUpdated()[0].GetLastSeenTimestamp().GetNanos(), "the connection should be open")
}

func (b *sendNetflowsSuite) assertOneUpdatedCloseConnection() {
	msg := mustReadTimeout(b.T(), b.m.sensorUpdates)
	netflowUpdate, ok := msg.Msg.(*central.MsgFromSensor_NetworkFlowUpdate)
	b.Require().True(ok, "message is NetworkFlowUpdate")
	b.Require().Len(netflowUpdate.NetworkFlowUpdate.GetUpdated(), 1, "one updated connection")
	b.Assert().NotEqual(int32(0), netflowUpdate.NetworkFlowUpdate.GetUpdated()[0].GetLastSeenTimestamp().GetNanos(), "the connection should not be open")
}

func (b *sendNetflowsSuite) assertOneUpdatedCloseEndpoint() {
	msg := mustReadTimeout(b.T(), b.m.sensorUpdates)
	netflowUpdate, ok := msg.Msg.(*central.MsgFromSensor_NetworkFlowUpdate)
	b.Require().True(ok, "message is NetworkFlowUpdate")
	b.Require().Len(netflowUpdate.NetworkFlowUpdate.GetUpdatedEndpoints(), 1, "one updated endpint")
	b.Assert().NotEqual(int32(0), netflowUpdate.NetworkFlowUpdate.GetUpdatedEndpoints()[0].GetLastActiveTimestamp().GetNanos(), "the endpoint should not be open")
}

func mustNotRead[T any](t *testing.T, ch chan T) {
	select {
	case <-ch:
		t.Fatal("should not receive in channel")
	case <-time.After(waitTimeout):
	}
}

func mustReadTimeout[T any](t *testing.T, ch chan T) T {
	var result T
	select {
	case v, more := <-ch:
		if !more {
			require.True(t, more, "channel should never close")
		}
		result = v
	case <-time.After(waitTimeout):
		t.Fatal("blocked on reading from channel")
	}
	return result
}

func mustSendWithoutBlock[T any](t *testing.T, ch chan T, v T) {
	select {
	case ch <- v:
		return
	case <-time.After(waitTimeout):
		t.Fatal("blocked on sending to channel")
	}
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
