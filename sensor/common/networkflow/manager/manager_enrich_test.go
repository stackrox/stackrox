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
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	"github.com/stackrox/rox/sensor/common/networkflow/updatecomputer"
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
	m, mockEntityStore, mockExternalSrc, _ := createManager(mockCtrl, updatecomputer.NewCategorized(), enrichTickerC)
	srcID := "src-id"
	dstID := "dst-id"

	// Create helpers for this test
	mocks := newMockExpectations(mockEntityStore, mockExternalSrc)

	cases := map[string]struct {
		connPair            *connectionPair
		enrichedConnections map[indicator.NetworkConn]timestamp.MicroTS
		setupMocks          func(*mockExpectations)
		expected            struct {
			result    EnrichmentResult
			action    PostEnrichmentAction
			indicator *indicator.NetworkConn
		}
	}{
		"Rotten connection should return rotten status": {
			connPair: createConnectionPair().incoming().external().firstSeen(timestamp.Now().Add(-env.ContainerIDResolutionGracePeriod.DurationSetting() * 2)),
			setupMocks: func(m *mockExpectations) {
				m.expectContainerNotFound()
			},
			expected: struct {
				result    EnrichmentResult
				action    PostEnrichmentAction
				indicator *indicator.NetworkConn
			}{
				result: EnrichmentResultContainerIDMissMarkRotten,
				action: PostEnrichmentActionRemove,
			},
		},
		"Incoming external connection with unsuccessful lookup should return internet entity": {
			connPair:            createConnectionPair().incoming().external(),
			enrichedConnections: make(map[indicator.NetworkConn]timestamp.MicroTS),
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound(dstID).expectExternalNotFound()
			},
			expected: struct {
				result    EnrichmentResult
				action    PostEnrichmentAction
				indicator *indicator.NetworkConn
			}{
				result: EnrichmentResultSuccess,
				action: PostEnrichmentActionCheckRemove,
				indicator: &indicator.NetworkConn{
					DstPort:   80,
					Protocol:  net.TCP.ToProtobuf(),
					SrcEntity: networkgraph.InternetEntity(),
					DstEntity: networkgraph.EntityForDeployment(dstID),
				},
			},
		},
		"Outgoing external connection with successful external lookup should return the correct id": {
			connPair:            createConnectionPair().external(),
			enrichedConnections: make(map[indicator.NetworkConn]timestamp.MicroTS),
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound(srcID).expectExternalFound(dstID)
			},
			expected: struct {
				result    EnrichmentResult
				action    PostEnrichmentAction
				indicator *indicator.NetworkConn
			}{
				result: EnrichmentResultSuccess,
				action: PostEnrichmentActionCheckRemove,
				indicator: &indicator.NetworkConn{
					DstPort:  80,
					Protocol: net.TCP.ToProtobuf(),
					DstEntity: networkgraph.EntityFromProto(&storage.NetworkEntityInfo{
						Id: dstID,
					}),
					SrcEntity: networkgraph.EntityForDeployment(srcID),
				},
			},
		},
		"Incoming local connection with successful lookup should be skipped": {
			connPair:            createConnectionPair().incoming(),
			enrichedConnections: make(map[indicator.NetworkConn]timestamp.MicroTS),
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound(srcID).expectEndpointFound(dstID)
			},
			expected: struct {
				result    EnrichmentResult
				action    PostEnrichmentAction
				indicator *indicator.NetworkConn
			}{
				result: EnrichmentResultSkipped,
				action: PostEnrichmentActionRemove,
			},
		},
		"Incoming fresh connection with valid address should not return anything": {
			connPair:            createConnectionPair().incoming(),
			enrichedConnections: make(map[indicator.NetworkConn]timestamp.MicroTS),
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound(dstID).expectEndpointNotFound()
			},
			expected: struct {
				result    EnrichmentResult
				action    PostEnrichmentAction
				indicator *indicator.NetworkConn
			}{
				result: EnrichmentResultRetryLater,
				action: PostEnrichmentActionRetry,
			},
		},
		"Incoming fresh connection with invalid address should be retried": {
			connPair:            createConnectionPair().incoming().invalidAddress(),
			enrichedConnections: make(map[indicator.NetworkConn]timestamp.MicroTS),
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound(dstID).expectEndpointNotFound().expectExternalNotFound()
			},
			expected: struct {
				result    EnrichmentResult
				action    PostEnrichmentAction
				indicator *indicator.NetworkConn
			}{
				result: EnrichmentResultRetryLater,
				action: PostEnrichmentActionRetry,
			},
		},
		"Outgoing connection with successful internal lookup should return the correct id": {
			connPair:            createConnectionPair(),
			enrichedConnections: make(map[indicator.NetworkConn]timestamp.MicroTS),
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound(srcID).expectEndpointFound(dstID, 80)
			},
			expected: struct {
				result    EnrichmentResult
				action    PostEnrichmentAction
				indicator *indicator.NetworkConn
			}{
				result: EnrichmentResultSuccess,
				action: PostEnrichmentActionCheckRemove,
				indicator: &indicator.NetworkConn{
					DstPort:  80,
					Protocol: net.TCP.ToProtobuf(),
					DstEntity: networkgraph.EntityFromProto(&storage.NetworkEntityInfo{
						Id: dstID,
					}),
					SrcEntity: networkgraph.EntityForDeployment(srcID),
				},
			},
		},
	}

	for name, tCase := range cases {
		s.Run(name, func() {
			// Setup mocks using helper
			tCase.setupMocks(mocks)
			assertions := newEnrichmentAssertion(s.T())

			// Execute test
			result, reason := m.enrichConnection(timestamp.Now(), tCase.connPair.conn, tCase.connPair.status, tCase.enrichedConnections)
			action := m.handleConnectionEnrichmentResult(result, reason, tCase.connPair.conn)

			// Assert using helper
			assertions.assertConnectionEnrichment(result, action, tCase.enrichedConnections, tCase.expected)
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
	containerEndpointIndicator1 := indicator.ContainerEndpoint{
		Entity:   networkgraph.EntityForDeployment(id),
		Port:     80,
		Protocol: net.TCP.ToProtobuf(),
	}
	nonEmptyProcessInfo := indicator.ProcessInfo{
		ProcessName: "grep",
		ProcessArgs: "-i",
		ProcessExec: "abc",
	}

	cases := map[string]struct {
		isPastContainerResolutionDeadline bool
		isFresh                           bool
		shouldFindContainerID             bool
		processKey                        indicator.ProcessInfo
		epInActiveEndpoints               *containerEndpointIndicatorWithAge
		lastSeen                          timestamp.MicroTS
		plopFeatEnabled                   bool
		offlineEnrichmentFeatEnabled      bool
		enrichedEndpoints                 map[indicator.ContainerEndpoint]timestamp.MicroTS
		enrichedProcesses                 map[indicator.ProcessListening]timestamp.MicroTS
		expected                          struct {
			resultNG   EnrichmentResult
			resultPLOP EnrichmentResult
			reasonNG   EnrichmentReasonEp
			reasonPLOP EnrichmentReasonEp
			action     PostEnrichmentAction
			endpoint   *indicator.ContainerEndpoint
		}
	}{
		"Container resolution deadline not passed yet": {
			isPastContainerResolutionDeadline: false, // required to retry
			isFresh:                           false,
			shouldFindContainerID:             false, // required to retry
			processKey:                        indicator.ProcessInfo{},
			epInActiveEndpoints:               nil,
			plopFeatEnabled:                   true,
			offlineEnrichmentFeatEnabled:      true,
			expected: struct {
				resultNG   EnrichmentResult
				resultPLOP EnrichmentResult
				reasonNG   EnrichmentReasonEp
				reasonPLOP EnrichmentReasonEp
				action     PostEnrichmentAction
				endpoint   *indicator.ContainerEndpoint
			}{
				resultNG:   EnrichmentResultRetryLater,
				resultPLOP: EnrichmentResultRetryLater,
				reasonNG:   EnrichmentReasonEpStillInGracePeriod,
				reasonPLOP: EnrichmentReasonEpStillInGracePeriod,
				action:     PostEnrichmentActionRetry,
				endpoint:   nil,
			},
		},
		"Rotten connection should return rotten status": {
			isPastContainerResolutionDeadline: true, // no retries, required for rotten
			isFresh:                           false,
			shouldFindContainerID:             false, // required for rotten
			processKey:                        indicator.ProcessInfo{},
			epInActiveEndpoints:               nil, // required for rotten
			plopFeatEnabled:                   true,
			offlineEnrichmentFeatEnabled:      true,
			expected: struct {
				resultNG   EnrichmentResult
				resultPLOP EnrichmentResult
				reasonNG   EnrichmentReasonEp
				reasonPLOP EnrichmentReasonEp
				action     PostEnrichmentAction
				endpoint   *indicator.ContainerEndpoint
			}{
				resultNG:   EnrichmentResultContainerIDMissMarkRotten,
				resultPLOP: EnrichmentResultContainerIDMissMarkRotten,
				reasonNG:   EnrichmentReasonEpOutsideOfGracePeriod,
				reasonPLOP: EnrichmentReasonEpOutsideOfGracePeriod,
				action:     PostEnrichmentActionRemove,
				endpoint:   nil,
			},
		},
		"Active connection for unknown container": {
			isPastContainerResolutionDeadline: true, // no retries, required
			isFresh:                           false,
			shouldFindContainerID:             false, // required
			processKey:                        indicator.ProcessInfo{},
			// epInActiveEndpoints: false=rotten, true=inactive
			epInActiveEndpoints: &containerEndpointIndicatorWithAge{
				ContainerEndpoint: containerEndpointIndicator1,
				lastUpdate:        now - 1,
			},
			plopFeatEnabled:              true,
			offlineEnrichmentFeatEnabled: true,
			expected: struct {
				resultNG   EnrichmentResult
				resultPLOP EnrichmentResult
				reasonNG   EnrichmentReasonEp
				reasonPLOP EnrichmentReasonEp
				action     PostEnrichmentAction
				endpoint   *indicator.ContainerEndpoint
			}{
				resultNG:   EnrichmentResultContainerIDMissMarkInactive,
				resultPLOP: EnrichmentResultContainerIDMissMarkInactive,
				reasonNG:   EnrichmentReasonEpOutsideOfGracePeriod,
				reasonPLOP: EnrichmentReasonEpOutsideOfGracePeriod,
				action:     PostEnrichmentActionMarkInactive,
				endpoint:   nil,
			},
		},
		"PLOP enrichment should be invalid for missing process info": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey:                        indicator.ProcessInfo{},
			epInActiveEndpoints:               nil,
			plopFeatEnabled:                   true,
			offlineEnrichmentFeatEnabled:      true,
			enrichedEndpoints:                 make(map[indicator.ContainerEndpoint]timestamp.MicroTS),
			enrichedProcesses:                 make(map[indicator.ProcessListening]timestamp.MicroTS),
			expected: struct {
				resultNG   EnrichmentResult
				resultPLOP EnrichmentResult
				reasonNG   EnrichmentReasonEp
				reasonPLOP EnrichmentReasonEp
				action     PostEnrichmentAction
				endpoint   *indicator.ContainerEndpoint
			}{
				resultNG:   EnrichmentResultSuccess,
				resultPLOP: EnrichmentResultInvalidInput,
				reasonNG:   EnrichmentReasonEpSuccessInactive,
				reasonPLOP: EnrichmentReasonEpEmptyProcessInfo,
				action:     PostEnrichmentActionCheckRemove,
			},
		},
		"PLOP enrichment should be skipped if feature is disabled": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey:                        nonEmptyProcessInfo,
			epInActiveEndpoints:               nil,
			plopFeatEnabled:                   false,
			offlineEnrichmentFeatEnabled:      true,
			enrichedEndpoints:                 make(map[indicator.ContainerEndpoint]timestamp.MicroTS),
			enrichedProcesses:                 make(map[indicator.ProcessListening]timestamp.MicroTS),
			expected: struct {
				resultNG   EnrichmentResult
				resultPLOP EnrichmentResult
				reasonNG   EnrichmentReasonEp
				reasonPLOP EnrichmentReasonEp
				action     PostEnrichmentAction
				endpoint   *indicator.ContainerEndpoint
			}{
				resultNG:   EnrichmentResultSuccess,
				resultPLOP: EnrichmentResultSkipped,
				reasonNG:   EnrichmentReasonEpSuccessInactive,
				reasonPLOP: EnrichmentReasonEpFeaturePlopDisabled,
				action:     PostEnrichmentActionCheckRemove,
			},
		},
		"Enrichment should pass for plop and network graph": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey:                        nonEmptyProcessInfo,
			epInActiveEndpoints:               nil,
			plopFeatEnabled:                   true,
			offlineEnrichmentFeatEnabled:      true,
			enrichedEndpoints:                 make(map[indicator.ContainerEndpoint]timestamp.MicroTS),
			enrichedProcesses:                 make(map[indicator.ProcessListening]timestamp.MicroTS),
			expected: struct {
				resultNG   EnrichmentResult
				resultPLOP EnrichmentResult
				reasonNG   EnrichmentReasonEp
				reasonPLOP EnrichmentReasonEp
				action     PostEnrichmentAction
				endpoint   *indicator.ContainerEndpoint
			}{
				resultNG:   EnrichmentResultSuccess,
				resultPLOP: EnrichmentResultSuccess,
				reasonNG:   EnrichmentReasonEpSuccessInactive,
				reasonPLOP: EnrichmentReasonEp(""),
				action:     PostEnrichmentActionCheckRemove,
				endpoint: &indicator.ContainerEndpoint{
					Entity:   networkgraph.EntityForDeployment(id),
					Port:     80,
					Protocol: net.TCP.ToProtobuf(),
				},
			},
		},
		"Enrichment success for unclosed connection should mark endpoint as active": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey:                        nonEmptyProcessInfo,
			epInActiveEndpoints: &containerEndpointIndicatorWithAge{
				ContainerEndpoint: containerEndpointIndicator1,
				lastUpdate:        now - 1,
			},
			plopFeatEnabled:              true,
			offlineEnrichmentFeatEnabled: true,
			lastSeen:                     timestamp.InfiniteFuture, // required for SuccessActive result
			enrichedEndpoints:            make(map[indicator.ContainerEndpoint]timestamp.MicroTS),
			enrichedProcesses:            make(map[indicator.ProcessListening]timestamp.MicroTS),
			expected: struct {
				resultNG   EnrichmentResult
				resultPLOP EnrichmentResult
				reasonNG   EnrichmentReasonEp
				reasonPLOP EnrichmentReasonEp
				action     PostEnrichmentAction
				endpoint   *indicator.ContainerEndpoint
			}{
				resultNG:   EnrichmentResultSuccess,
				resultPLOP: EnrichmentResultSuccess,
				reasonNG:   EnrichmentReasonEpSuccessActive,
				reasonPLOP: EnrichmentReasonEp(""),
				action:     PostEnrichmentActionCheckRemove,
				endpoint: &indicator.ContainerEndpoint{
					Entity:   networkgraph.EntityForDeployment(id),
					Port:     80,
					Protocol: net.TCP.ToProtobuf(),
				},
			},
		},
		"Late enrichment for already enriched closed endpoint should yield EnrichmentReasonEpDuplicate": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey:                        nonEmptyProcessInfo,
			epInActiveEndpoints: &containerEndpointIndicatorWithAge{
				ContainerEndpoint: containerEndpointIndicator1,
				lastUpdate:        now - 1,
			},
			plopFeatEnabled:              true,
			offlineEnrichmentFeatEnabled: true,
			lastSeen:                     now - 10, // message being 10units old should trigger `EnrichmentReasonEpDuplicate`
			enrichedEndpoints: map[indicator.ContainerEndpoint]timestamp.MicroTS{
				containerEndpointIndicator1: now - 1, // existing state in memory must "be younger" than lastSeen
			},
			enrichedProcesses: make(map[indicator.ProcessListening]timestamp.MicroTS),
			expected: struct {
				resultNG   EnrichmentResult
				resultPLOP EnrichmentResult
				reasonNG   EnrichmentReasonEp
				reasonPLOP EnrichmentReasonEp
				action     PostEnrichmentAction
				endpoint   *indicator.ContainerEndpoint
			}{
				resultNG:   EnrichmentResultSuccess,
				resultPLOP: EnrichmentResultSuccess,
				reasonNG:   EnrichmentReasonEpDuplicate,
				reasonPLOP: EnrichmentReasonEp(""),
				action:     PostEnrichmentActionCheckRemove,
				endpoint: &indicator.ContainerEndpoint{
					Entity:   networkgraph.EntityForDeployment(id),
					Port:     80,
					Protocol: net.TCP.ToProtobuf(),
				},
			},
		},
		"Enrichment for disabled SensorCapturesIntermediateEvents feature should yield EnrichmentReasonEpFeatureDisabled": {
			isPastContainerResolutionDeadline: false,
			isFresh:                           false,
			shouldFindContainerID:             true,
			processKey:                        nonEmptyProcessInfo,
			epInActiveEndpoints: &containerEndpointIndicatorWithAge{
				ContainerEndpoint: containerEndpointIndicator1,
				lastUpdate:        now - 1,
			},
			plopFeatEnabled:              true,
			offlineEnrichmentFeatEnabled: false,
			lastSeen:                     timestamp.InfiniteFuture,
			enrichedEndpoints:            make(map[indicator.ContainerEndpoint]timestamp.MicroTS),
			enrichedProcesses:            make(map[indicator.ProcessListening]timestamp.MicroTS),
			expected: struct {
				resultNG   EnrichmentResult
				resultPLOP EnrichmentResult
				reasonNG   EnrichmentReasonEp
				reasonPLOP EnrichmentReasonEp
				action     PostEnrichmentAction
				endpoint   *indicator.ContainerEndpoint
			}{
				resultNG:   EnrichmentResultSuccess,
				resultPLOP: EnrichmentResultSuccess,
				reasonNG:   EnrichmentReasonEpFeatureDisabled,
				reasonPLOP: EnrichmentReasonEp(""),
				action:     PostEnrichmentActionCheckRemove,
				endpoint: &indicator.ContainerEndpoint{
					Entity:   networkgraph.EntityForDeployment(id),
					Port:     80,
					Protocol: net.TCP.ToProtobuf(),
				},
			},
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			m, mockEntityStore, _, _ := createManager(mockCtrl, updatecomputer.NewCategorized(), enrichTickerC)

			// Setup environment variables
			s.T().Setenv(env.ProcessesListeningOnPort.EnvVar(), strconv.FormatBool(tc.plopFeatEnabled))
			s.T().Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), strconv.FormatBool(tc.offlineEnrichmentFeatEnabled))

			// Setup mocks using helper
			mocks := newMockExpectations(mockEntityStore, nil)
			if tc.shouldFindContainerID {
				mocks.expectContainerFound(id)
			} else {
				mocks.expectContainerNotFound()
			}

			// Setup timing based on test configuration
			firstSeen := now // guarantees isFresh=true
			if tc.isPastContainerResolutionDeadline && tc.isFresh {
				s.Fail("Test case invalid: Impossible to make ts older than 2 minutes and younger than 10s")
			}
			if tc.isPastContainerResolutionDeadline { // must be older than 2min
				firstSeen = firstSeen.Add(-env.ContainerIDResolutionGracePeriod.DurationSetting() * 2) // first seen 4 minutes ago
			} else if !tc.isFresh { // must be younger than 2min, but older than 10s
				firstSeen = firstSeen.Add(-env.ClusterEntityResolutionWaitPeriod.DurationSetting() * 2) // first seen 20s ago
			}
			ep := createEndpointPairWithProcess(firstSeen, now, tc.lastSeen, tc.processKey)

			if tc.epInActiveEndpoints != nil {
				m.activeEndpoints[*ep.endpoint] = tc.epInActiveEndpoints
			}

			// Execute test
			resultNG, resultPLOP, reasonNG, reasonPLOP := m.enrichContainerEndpoint(now, ep.endpoint, ep.status, tc.enrichedEndpoints, tc.enrichedProcesses, now)
			action := m.handleEndpointEnrichmentResult(resultNG, resultPLOP, reasonNG, reasonPLOP, ep.endpoint)

			// Assert using helper
			assertions := newEnrichmentAssertion(s.T())
			assertions.assertEndpointEnrichment(resultNG, resultPLOP, reasonNG, reasonPLOP, action, tc.enrichedEndpoints, tc.expected)
		})
	}
}
