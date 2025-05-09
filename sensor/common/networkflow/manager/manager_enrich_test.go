package manager

import (
	"strconv"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestNetworkFlowManagerEnrichment(t *testing.T) {
	suite.Run(t, new(TestNetworkFlowManagerEnrichmentTestSuite))
}

// TestNetworkFlowManagerEnrichmentTestSuite focuses on the enrichment behavior of the manager
type TestNetworkFlowManagerEnrichmentTestSuite struct {
	suite.Suite
}

func (s *TestNetworkFlowManagerEnrichmentTestSuite) TestEnrichConnection() {
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer mockCtrl.Finish()
	m, mockEntityStore, mockExternalSrc, _ := createManager(mockCtrl, enrichTickerC)
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
		expectedResult              EnrichmentResult
		expectedAction              PostEnrichmentAction
	}{
		"Rotten connection should return rotten status": {
			connPair: createConnectionPair().incoming().external().firstSeen(timestamp.Now().Add(-env.ContainerIDResolutionGracePeriod.DurationSetting() * 2)),
			expectEntityLookupContainer: func() {
				mockEntityStore.EXPECT().LookupByContainerID(gomock.Any()).Times(1).DoAndReturn(func(_ any) (clusterentities.ContainerMetadata, bool, bool) {
					return clusterentities.ContainerMetadata{}, false, false
				})
			},
			expectedResult: EnrichmentResultContainerIDMissMarkRotten,
			expectedAction: PostEnrichmentActionRemove,
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
			expectedResult: EnrichmentResultSuccess,
			expectedAction: PostEnrichmentActionCheckRemove,
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
			expectedResult: EnrichmentResultSuccess,
			expectedAction: PostEnrichmentActionCheckRemove,
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
			expectedResult: EnrichmentResultInvalidInput,
			expectedAction: PostEnrichmentActionRetry, //FIXME: should this really be retried?
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
			expectedResult: EnrichmentResultRetryLater,
			expectedAction: PostEnrichmentActionRetry,
		},
		"Incoming fresh connection with invalid address should be retried": {
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
			expectedResult: EnrichmentResultRetryLater,
			expectedAction: PostEnrichmentActionRetry,
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
			expectedResult: EnrichmentResultSuccess,
			expectedAction: PostEnrichmentActionCheckRemove,
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
			result, reason := m.enrichConnection(timestamp.Now(), tCase.connPair.conn, tCase.connPair.status, tCase.enrichedConnections)
			action := m.handleConnectionEnrichmentResult(result, reason, *tCase.connPair.conn)
			s.Assert().Equal(tCase.expectedResult, result)
			s.Assert().Equal(tCase.expectedAction, action)
			if tCase.expectedIndicator != nil {
				_, ok := tCase.enrichedConnections[*tCase.expectedIndicator]
				s.Assert().True(ok)
			} else {
				s.Assert().Len(tCase.enrichedConnections, 0)
			}
		})
	}
}

func (s *TestNetworkFlowManagerEnrichmentTestSuite) TestEnrichContainerEndpoint() {
	mockCtrl := gomock.NewController(s.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	defer mockCtrl.Finish()
	id := "id"
	now := timestamp.Now()
	containerEndpointIndicator1 := containerEndpointIndicator{
		entity:   networkgraph.EntityForDeployment(id),
		port:     80,
		protocol: net.TCP.ToProtobuf(),
	}

	cases := map[string]struct {
		isPastContainerResolutionDeadline bool
		isFresh                           bool
		shouldFindContainerID             bool
		processKey                        processInfo
		epInActiveEndpoints               *containerEndpointIndicatorWithAge
		lastSeen                          timestamp.MicroTS
		plopFeatEnabled                   bool
		offlineEnrichmentFeatEnabled      bool
		expectedResultNG                  EnrichmentResult
		expectedReasonNG                  EnrichmentReasonEp
		expectedResultPLOP                EnrichmentResult
		expectedReasonPLOP                EnrichmentReasonEp
		expectedAction                    PostEnrichmentAction

		enrichedEndpoints map[containerEndpointIndicator]timestamp.MicroTS
		enrichedProcesses map[processListeningIndicator]timestamp.MicroTS
		expectedEndpoint  *containerEndpointIndicator
	}{
		"Container resolution deadline not passed yet": {
			isPastContainerResolutionDeadline: false, // required to retry
			isFresh:                           false,
			shouldFindContainerID:             false, // required to retry
			processKey:                        processInfo{},
			epInActiveEndpoints:               nil,
			plopFeatEnabled:                   true,
			offlineEnrichmentFeatEnabled:      true,
			expectedResultNG:                  EnrichmentResultRetryLater,
			expectedResultPLOP:                EnrichmentResultRetryLater,
			expectedReasonNG:                  EnrichmentReasonEpStillInGracePeriod,
			expectedReasonPLOP:                EnrichmentReasonEpStillInGracePeriod,
			expectedAction:                    PostEnrichmentActionRetry,
			expectedEndpoint:                  nil,
		},
		"Rotten connection should return rotten status": {
			isPastContainerResolutionDeadline: true, // no retries, required for rotten
			isFresh:                           false,
			shouldFindContainerID:             false, // required for rotten
			processKey:                        processInfo{},
			epInActiveEndpoints:               nil, // required for rotten
			plopFeatEnabled:                   true,
			offlineEnrichmentFeatEnabled:      true,
			expectedResultNG:                  EnrichmentResultContainerIDMissMarkRotten,
			expectedResultPLOP:                EnrichmentResultContainerIDMissMarkRotten,
			expectedReasonNG:                  EnrichmentReasonEpOutsideOfGracePeriod,
			expectedReasonPLOP:                EnrichmentReasonEpOutsideOfGracePeriod,
			expectedAction:                    PostEnrichmentActionRemove,
			expectedEndpoint:                  nil,
		},
		"Active connection for unknown container": {
			isPastContainerResolutionDeadline: true, // no retries, required
			isFresh:                           false,
			shouldFindContainerID:             false, // required
			processKey:                        processInfo{},
			// epInActiveEndpoints: false=rotten, true=inactive
			epInActiveEndpoints: &containerEndpointIndicatorWithAge{
				containerEndpointIndicator: containerEndpointIndicator1,
				lastUpdate:                 now - 1,
			},
			plopFeatEnabled:              true,
			offlineEnrichmentFeatEnabled: true,
			expectedResultNG:             EnrichmentResultContainerIDMissMarkInactive,
			expectedResultPLOP:           EnrichmentResultContainerIDMissMarkInactive,
			expectedReasonNG:             EnrichmentReasonEpOutsideOfGracePeriod,
			expectedReasonPLOP:           EnrichmentReasonEpOutsideOfGracePeriod,
			expectedAction:               PostEnrichmentActionMarkInactive, // TODO: check if this is correct action. IF containerID is not found, maybe we should ADDITIONALLY remove it (it would have been anyway removed in the next cycle)?
			expectedEndpoint:             nil,
		},
		"PLOP enrichment should be invalid for missing process info": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey:                        processInfo{},
			epInActiveEndpoints:               nil,
			plopFeatEnabled:                   true,
			offlineEnrichmentFeatEnabled:      true,
			expectedResultNG:                  EnrichmentResultSuccess,
			expectedResultPLOP:                EnrichmentResultInvalidInput,
			expectedReasonNG:                  EnrichmentReasonEpSuccessInactive,
			expectedReasonPLOP:                EnrichmentReasonEpEmptyProcessInfo,
			expectedAction:                    PostEnrichmentActionCheckRemove,

			enrichedEndpoints: make(map[containerEndpointIndicator]timestamp.MicroTS),
			enrichedProcesses: make(map[processListeningIndicator]timestamp.MicroTS),
		},
		"PLOP enrichment should be skipped if feature is disabled": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey: processInfo{
				processName: "grep",
				processArgs: "-i",
				processExec: "abc",
			},
			epInActiveEndpoints:          nil,
			plopFeatEnabled:              false,
			offlineEnrichmentFeatEnabled: true,
			expectedResultNG:             EnrichmentResultSuccess,
			expectedResultPLOP:           EnrichmentResultSkipped,
			expectedReasonNG:             EnrichmentReasonEpSuccessInactive,
			expectedReasonPLOP:           EnrichmentReasonEpFeaturePlopDisabled,
			expectedAction:               PostEnrichmentActionCheckRemove,

			enrichedEndpoints: make(map[containerEndpointIndicator]timestamp.MicroTS),
			enrichedProcesses: make(map[processListeningIndicator]timestamp.MicroTS),
		},
		"Enrichment should pass for plop and network graph": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey: processInfo{
				processName: "grep",
				processArgs: "-i",
				processExec: "abc",
			},
			epInActiveEndpoints:          nil,
			plopFeatEnabled:              true,
			offlineEnrichmentFeatEnabled: true,
			expectedResultNG:             EnrichmentResultSuccess,
			expectedResultPLOP:           EnrichmentResultSuccess,
			expectedReasonNG:             EnrichmentReasonEpSuccessInactive,
			expectedReasonPLOP:           EnrichmentReasonEp(""), // EnrichmentResult is Success, so no need to give more verbose reason
			expectedAction:               PostEnrichmentActionCheckRemove,

			enrichedEndpoints: make(map[containerEndpointIndicator]timestamp.MicroTS),
			enrichedProcesses: make(map[processListeningIndicator]timestamp.MicroTS),
			expectedEndpoint: &containerEndpointIndicator{
				entity:   networkgraph.EntityForDeployment(id),
				port:     80,
				protocol: net.TCP.ToProtobuf(),
			},
		},
		"Enrichment success for unclosed connection should mark endpoint as active": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey:                        defaultProcessKey(),
			epInActiveEndpoints: &containerEndpointIndicatorWithAge{
				containerEndpointIndicator: containerEndpointIndicator1,
				lastUpdate:                 now - 1,
			},
			plopFeatEnabled:              true,
			offlineEnrichmentFeatEnabled: true,
			lastSeen:                     timestamp.InfiniteFuture, // required for SuccessActive result
			expectedResultNG:             EnrichmentResultSuccess,
			expectedResultPLOP:           EnrichmentResultSuccess,
			expectedReasonNG:             EnrichmentReasonEpSuccessActive,
			expectedReasonPLOP:           EnrichmentReasonEp(""),
			expectedAction:               PostEnrichmentActionCheckRemove,

			enrichedEndpoints: make(map[containerEndpointIndicator]timestamp.MicroTS),
			enrichedProcesses: make(map[processListeningIndicator]timestamp.MicroTS),
			expectedEndpoint: &containerEndpointIndicator{
				entity:   networkgraph.EntityForDeployment(id),
				port:     80,
				protocol: net.TCP.ToProtobuf(),
			},
		},
		"Late enrichment for already enriched closed endpoint should yield EnrichmentReasonEpDuplicate": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey:                        defaultProcessKey(),
			epInActiveEndpoints: &containerEndpointIndicatorWithAge{
				containerEndpointIndicator: containerEndpointIndicator1,
				lastUpdate:                 now - 1,
			},
			plopFeatEnabled:              true,
			offlineEnrichmentFeatEnabled: true,
			lastSeen:                     now - 10, // message being 10units old should trigger `EnrichmentReasonEpDuplicate`
			expectedResultNG:             EnrichmentResultSuccess,
			expectedResultPLOP:           EnrichmentResultSuccess,
			expectedReasonNG:             EnrichmentReasonEpDuplicate,
			expectedReasonPLOP:           EnrichmentReasonEp(""),
			expectedAction:               PostEnrichmentActionCheckRemove,

			enrichedEndpoints: map[containerEndpointIndicator]timestamp.MicroTS{
				containerEndpointIndicator1: now - 1, // existing state in memory must "be younger" than lastSeen
			},
			enrichedProcesses: make(map[processListeningIndicator]timestamp.MicroTS),
			expectedEndpoint: &containerEndpointIndicator{
				entity:   networkgraph.EntityForDeployment(id),
				port:     80,
				protocol: net.TCP.ToProtobuf(),
			},
		},
		"Enrichment for disabled SensorCapturesIntermediateEvents feature should yield EnrichmentReasonEpFeatureDisabled": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey:                        defaultProcessKey(),
			epInActiveEndpoints: &containerEndpointIndicatorWithAge{
				containerEndpointIndicator: containerEndpointIndicator1,
				lastUpdate:                 now - 1,
			},
			plopFeatEnabled:              true,
			offlineEnrichmentFeatEnabled: false,
			lastSeen:                     timestamp.InfiniteFuture,
			expectedResultNG:             EnrichmentResultSuccess,
			expectedResultPLOP:           EnrichmentResultSuccess,
			expectedReasonNG:             EnrichmentReasonEpFeatureDisabled,
			expectedReasonPLOP:           EnrichmentReasonEp(""),
			expectedAction:               PostEnrichmentActionCheckRemove,

			enrichedEndpoints: make(map[containerEndpointIndicator]timestamp.MicroTS),
			enrichedProcesses: make(map[processListeningIndicator]timestamp.MicroTS),
			expectedEndpoint: &containerEndpointIndicator{
				entity:   networkgraph.EntityForDeployment(id),
				port:     80,
				protocol: net.TCP.ToProtobuf(),
			},
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)

			s.T().Setenv(env.ProcessesListeningOnPort.EnvVar(), strconv.FormatBool(tc.plopFeatEnabled))
			s.T().Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), strconv.FormatBool(tc.offlineEnrichmentFeatEnabled))
			if tc.shouldFindContainerID {
				expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{DeploymentID: id}, true, false)()
			} else {
				expectEntityLookupContainerHelper(mockEntityStore, 1, clusterentities.ContainerMetadata{}, false, false)()
			}
			firstSeen := now // guarantees isFresh=true

			if tc.isPastContainerResolutionDeadline && tc.isFresh {
				s.Fail("Test case invalid: Impossible to make ts older than 2 minutes and younger than 10s")
			}
			if tc.isPastContainerResolutionDeadline { // must be older than 2min
				firstSeen = firstSeen.Add(-env.ContainerIDResolutionGracePeriod.DurationSetting() * 2) // first seen 4 minutes ago
			} else if !tc.isFresh { // must be younger than 2min, but older than 10s
				firstSeen = firstSeen.Add(-clusterEntityResolutionWaitPeriod * 2) // first seen 20s ago
			}
			ep := createEndpointPairWithProcess(firstSeen, now, tc.lastSeen, tc.processKey)

			if tc.epInActiveEndpoints != nil {
				m.activeEndpoints[*ep.endpoint] = tc.epInActiveEndpoints
			}

			resultNG, resultPLOP, reasonNG, reasonPLOP := m.enrichContainerEndpoint(now, ep.endpoint, ep.status, tc.enrichedEndpoints, tc.enrichedProcesses, now)
			action := m.handleEndpointEnrichmentResult(resultNG, resultPLOP, reasonNG, reasonPLOP, ep.endpoint)

			s.Equal(tc.expectedResultNG, resultNG, "Incorrect NetGraph result. Reason: %s", reasonNG)
			s.Equal(tc.expectedResultPLOP, resultPLOP, "Incorrect PLOP result. Reason: %s", reasonPLOP)

			s.Equal(tc.expectedReasonNG, reasonNG, "Expected NetGraph reason %s, got %s", tc.expectedReasonNG, reasonNG)
			s.Equal(tc.expectedReasonPLOP, reasonPLOP, "Incorrect PLOP reason %s, got %s", tc.expectedReasonPLOP, reasonPLOP)

			s.Equal(tc.expectedAction, action, "Incorrect action. Reasons (NG, PLOP): %s, %s", reasonNG, reasonPLOP)

			if tc.expectedEndpoint != nil {
				_, ok := tc.enrichedEndpoints[*tc.expectedEndpoint]
				s.Assert().True(ok)
			}
		})
	}
}
