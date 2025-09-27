package manager

import (
	"strconv"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestEnrichmentResult_IsConsumed(t *testing.T) {
	tests := map[string]struct {
		plopEnabled          bool
		consumedNetworkGraph bool
		consumedPLOP         bool
		expectedIsConsumed   bool
	}{
		"Both consumed with PLOP enabled": {
			plopEnabled:          true,
			consumedNetworkGraph: true,
			consumedPLOP:         true,
			expectedIsConsumed:   true,
		},
		"Only network graph consumed with PLOP enabled": {
			plopEnabled:          true,
			consumedNetworkGraph: true,
			consumedPLOP:         false,
			expectedIsConsumed:   false,
		},
		"Only PLOP consumed with PLOP enabled": {
			plopEnabled:          true,
			consumedNetworkGraph: false,
			consumedPLOP:         true,
			expectedIsConsumed:   false,
		},
		"Neither consumed with PLOP enabled": {
			plopEnabled:          true,
			consumedNetworkGraph: false,
			consumedPLOP:         false,
			expectedIsConsumed:   false,
		},
		"Network graph consumed with PLOP disabled": {
			plopEnabled:          false,
			consumedNetworkGraph: true,
			consumedPLOP:         false, // should be ignored when PLOP disabled
			expectedIsConsumed:   true,
		},
		"Network graph not consumed with PLOP disabled": {
			plopEnabled:          false,
			consumedNetworkGraph: false,
			consumedPLOP:         true, // should be ignored when PLOP disabled
			expectedIsConsumed:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup environment for PLOP feature
			t.Setenv(env.ProcessesListeningOnPort.EnvVar(), strconv.FormatBool(tt.plopEnabled))

			// Create enrichment result with test values
			consumption := &enrichmentConsumption{
				consumedNetworkGraph: tt.consumedNetworkGraph,
				consumedPLOP:         tt.consumedPLOP,
			}

			// Test IsConsumed method
			isConsumed := consumption.IsConsumed()

			assert.Equal(t, tt.expectedIsConsumed, isConsumed,
				"IsConsumed() should return %v when PLOP=%v, networkGraph=%v, PLOP=%v",
				tt.expectedIsConsumed, tt.plopEnabled, tt.consumedNetworkGraph, tt.consumedPLOP)
		})
	}
}

// TestEnrichConnection_BusinessLogicPaths tests the real business logic in enrichConnection
// focusing on uncovered paths identified by coverage analysis
func TestEnrichConnection_BusinessLogicPaths(t *testing.T) {
	// notFreshConnStatus represents a connection that was first seen beyond the cluster entity resolution wait period (not fresh)
	notFreshConnStatus := &connStatus{
		firstSeen:             timestamp.Now().Add(-env.ClusterEntityResolutionWaitPeriod.DurationSetting() - time.Second), // not fresh
		lastSeen:              timestamp.Now(),
		enrichmentConsumption: enrichmentConsumption{},
	}
	tests := map[string]struct {
		setupConnection    func() (*connection, *connStatus)
		setupMocks         func(*mockExpectations)
		setupFeatureFlags  func(*testing.T)
		expectedResult     EnrichmentResult
		expectedReason     EnrichmentReasonConn
		validateEnrichment func(*testing.T, map[indicator.NetworkConn]timestamp.MicroTS)
	}{
		"IP parsing error caused by malformed address should yield result EnrichmentResultInvalidInput with reason EnrichmentReasonConnParsingIPFailed": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    false,
					remote:      createEndpoint("invalid-ip", 80), // This will cause IsExternal() to fail
				}
				return conn, notFreshConnStatus
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment").expectEndpointNotFound().expectExternalNotFound()
			},
			expectedResult: EnrichmentResultInvalidInput,
			expectedReason: EnrichmentReasonConnParsingIPFailed,
		},
		"External source network found should yield result EnrichmentResultSuccess with reason EnrichmentReasonConnSuccess": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    false,
					remote:      createEndpoint("192.168.1.100", 80), // internal but with external source
				}
				return conn, notFreshConnStatus
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment").expectEndpointNotFound().expectExternalFound("external-network-id")
			},
			expectedResult: EnrichmentResultSuccess,
			expectedReason: EnrichmentReasonConnSuccess,
			validateEnrichment: func(t *testing.T, enriched map[indicator.NetworkConn]timestamp.MicroTS) {
				assert.Len(t, enriched, 1, "Should have one enriched connection")
				for indicator := range enriched {
					assert.Equal(t, "external-network-id", indicator.DstEntity.ID, "Should use external source entity")
				}
			},
		},
		"Incoming local connection should be skipped with result EnrichmentResultSkipped and reason EnrichmentReasonConnIncomingInternalConnection": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    true,                           // incoming connection
					remote:      createEndpoint("10.0.0.1", 80), // internal IP
					local: net.NetworkPeerID{
						Port: 8080,
					},
				}
				return conn, notFreshConnStatus
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment").expectEndpointFound("local-endpoint-id", 8080)
			},
			expectedResult: EnrichmentResultSkipped,
			expectedReason: EnrichmentReasonConnIncomingInternalConnection,
		},
		"External connection with Internet entity fallback should yield result EnrichmentResultSuccess with reason EnrichmentReasonConnSuccess": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    false,
					remote:      createEndpointWithAddress(net.ExternalIPv4Addr, 80), // considered external
				}
				return conn, notFreshConnStatus
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment").expectExternalNotFound()
			},
			expectedResult: EnrichmentResultSuccess,
			expectedReason: EnrichmentReasonConnSuccess,
			validateEnrichment: func(t *testing.T, enriched map[indicator.NetworkConn]timestamp.MicroTS) {
				assert.Len(t, enriched, 1, "Should have one enriched connection")
				for indicator := range enriched {
					assert.Equal(t, networkgraph.InternetEntity().ID, indicator.DstEntity.ID, "Should use Internet entity")
				}
			},
		},
		"Internal connection with fallback entity should yield result EnrichmentResultSuccess with reason EnrichmentReasonConnSuccess": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    false,
					remote:      createEndpoint("10.0.0.1", 80), // internal IP
				}
				return conn, notFreshConnStatus
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment").expectEndpointNotFound().expectExternalNotFound()
			},
			expectedResult: EnrichmentResultSuccess,
			expectedReason: EnrichmentReasonConnSuccess,
			validateEnrichment: func(t *testing.T, enriched map[indicator.NetworkConn]timestamp.MicroTS) {
				assert.Len(t, enriched, 1, "Should have one enriched connection")
				// For internal connections without external source, it creates a fallback entity
				// This tests the fallback behavior for unknown internal addresses
				for indicator := range enriched {
					// Should enrich successfully regardless of entity type
					assert.NotEmpty(t, indicator.DstEntity.ID, "Should have a valid destination entity")
				}
			},
		},
		"Connection with SensorCapturesIntermediateEvents disabled should yield result EnrichmentResultSuccess with reason EnrichmentReasonConnSuccess": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    false,
					remote:      createEndpoint("8.8.8.8", 80),
				}
				status := &connStatus{
					firstSeen:             timestamp.Now().Add(-time.Minute),
					lastSeen:              timestamp.InfiniteFuture, // active connection
					enrichmentConsumption: enrichmentConsumption{},
				}
				return conn, status
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment").expectEndpointFound("cluster-endpoint-id", 80)
			},
			setupFeatureFlags: func(t *testing.T) {
				t.Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), "false")
			},
			expectedResult: EnrichmentResultSuccess,
			expectedReason: EnrichmentReasonConnSuccess,
			validateEnrichment: func(t *testing.T, enriched map[indicator.NetworkConn]timestamp.MicroTS) {
				// Should still enrich even with feature disabled
				assert.Len(t, enriched, 1, "Should have one enriched connection")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup feature flags if needed
			if tt.setupFeatureFlags != nil {
				tt.setupFeatureFlags(t)
			}

			// Create mock controller and manager
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			enrichTickerC := make(chan time.Time)
			defer close(enrichTickerC)

			m, mockEntityStore, mockExternalSrc, _ := createManager(mockCtrl, enrichTickerC)

			// Setup mocks
			mocks := newMockExpectations(mockEntityStore, mockExternalSrc)
			tt.setupMocks(mocks)

			// Setup test data
			conn, status := tt.setupConnection()
			enrichedConnections := make(map[indicator.NetworkConn]timestamp.MicroTS)

			// Execute the enrichment
			result, reason := m.enrichConnection(timestamp.Now(), conn, status, enrichedConnections)

			// Assert results
			assert.Equal(t, tt.expectedResult, result, "Enrichment result mismatch")
			assert.Equal(t, tt.expectedReason, reason, "Enrichment reason mismatch")

			// Validate enrichment details if provided
			if tt.validateEnrichment != nil {
				tt.validateEnrichment(t, enrichedConnections)
			}
		})
	}
}

// TestEnrichContainerEndpoint_EdgeCases tests edge cases in enrichContainerEndpoint
// that might not be well covered despite 100% line coverage
func TestEnrichContainerEndpoint_EdgeCases(t *testing.T) {
	// freshConnStatus represents a connection that was just seen now (fresh)
	freshConnStatus := &connStatus{
		firstSeen:             timestamp.Now(), // fresh
		lastSeen:              timestamp.Now(),
		enrichmentConsumption: enrichmentConsumption{},
	}

	// notFreshConnStatus represents a connection that was first seen beyond the cluster entity resolution wait period (not fresh)
	notFreshConnStatus := &connStatus{
		firstSeen:             timestamp.Now().Add(-env.ClusterEntityResolutionWaitPeriod.DurationSetting() - time.Second), // not fresh
		lastSeen:              timestamp.Now().Add(-env.ClusterEntityResolutionWaitPeriod.DurationSetting() - time.Second), // older timestamp
		enrichmentConsumption: enrichmentConsumption{},
	}

	// Common endpoint configuration used by multiple test cases
	commonEndpoint := createEndpoint("8.8.8.8", 80)

	tests := map[string]struct {
		setupEndpoint      func() (*containerEndpoint, *connStatus)
		setupMocks         func(*mockExpectations)
		expectedResultNG   EnrichmentResult
		expectedResultPLOP EnrichmentResult
		expectedReasonNG   EnrichmentReasonEp
		prePopulateData    func(*testing.T, map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp)
	}{
		"Fresh endpoint with no process info should yield result EnrichmentResultSuccess for Network Graph and EnrichmentResultInvalidInput for PLOP": {
			setupEndpoint: func() (*containerEndpoint, *connStatus) {
				ep := &containerEndpoint{
					endpoint:    commonEndpoint,
					containerID: "test-container",
					processKey:  indicator.ProcessInfo{}, // empty process info
				}
				return ep, freshConnStatus
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment")
			},
			expectedResultNG:   EnrichmentResultSuccess,
			expectedResultPLOP: EnrichmentResultInvalidInput,
			expectedReasonNG:   EnrichmentReasonEpSuccessInactive,
		},
		"Endpoint with duplicate timestamp should be marked as duplicate with result EnrichmentResultSuccess for both Network Graph and PLOP": {
			setupEndpoint: func() (*containerEndpoint, *connStatus) {
				ep := &containerEndpoint{
					endpoint:    commonEndpoint,
					containerID: "test-container",
					processKey:  defaultProcessKey(),
				}
				return ep, notFreshConnStatus
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment")
			},
			expectedResultNG:   EnrichmentResultSuccess,
			expectedResultPLOP: EnrichmentResultSuccess,
			expectedReasonNG:   EnrichmentReasonEpDuplicate,
			prePopulateData: func(t *testing.T, data map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp) {
				// Pre-populate with newer timestamp to trigger duplicate detection
				endpointIndicator := indicator.ContainerEndpoint{
					Entity:   networkgraph.EntityForDeployment("test-deployment"),
					Port:     80,
					Protocol: net.TCP.ToProtobuf(),
				}
				processIndicator := indicator.ProcessListening{
					ContainerName: "test-container",
					DeploymentID:  "test-deployment",
					Process: indicator.ProcessInfo{
						ProcessName: "test-process",
						ProcessArgs: "test-args",
						ProcessExec: "test-exec",
					},
					Port:      80,
					Protocol:  net.TCP.ToProtobuf(),
					PodID:     "test-pod",
					PodUID:    "test-pod-uid",
					Namespace: "test-namespace",
				}
				data[endpointIndicator] = &indicator.ProcessListeningWithTimestamp{
					ProcessListening: &processIndicator,
					LastSeen:         timestamp.Now(), // a newer timestamp
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create mock controller and manager
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			enrichTickerC := make(chan time.Time)
			defer close(enrichTickerC)

			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)

			// Setup mocks
			mocks := newMockExpectations(mockEntityStore, nil)
			tt.setupMocks(mocks)

			// Setup test data
			ep, status := tt.setupEndpoint()
			enrichedEndpointsProcesses := make(map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp)

			// Pre-populate data if validation function needs it
			if tt.prePopulateData != nil {
				tt.prePopulateData(t, enrichedEndpointsProcesses)
			}

			// Execute the enrichment
			now := timestamp.Now()
			resultNG, resultPLOP, reasonNG, _ := m.enrichContainerEndpoint(
				now, ep, status, enrichedEndpointsProcesses, now)

			// Assert results
			assert.Equal(t, tt.expectedResultNG, resultNG, "Network graph enrichment result mismatch")
			assert.Equal(t, tt.expectedResultPLOP, resultPLOP, "PLOP enrichment result mismatch")
			assert.Equal(t, tt.expectedReasonNG, reasonNG, "Network graph enrichment reason mismatch")

			// Additional validation can be added here for specific test cases
		})
	}
}

func Test_connStatus_checkRemoveCondition(t *testing.T) {
	tests := map[string]struct {
		rotten     bool
		closed     bool
		useLegacy  bool
		isConsumed bool
		want       bool
	}{
		// Legacy
		"Legacy shall remove closed, consumed EEs": {
			rotten:     false,
			closed:     true,
			useLegacy:  true,
			isConsumed: true,
			want:       true,
		},
		"Legacy shall keep closed, unconsumed EEs": {
			rotten:     false,
			closed:     true,
			useLegacy:  true,
			isConsumed: false,
			want:       false,
		},
		"Legacy shall keep open, consumed EEs": {
			rotten:     false,
			closed:     false,
			useLegacy:  true,
			isConsumed: true,
			want:       false,
		},
		"Legacy shall keep open, unconsumed EEs": {
			rotten:     false,
			closed:     false,
			useLegacy:  true,
			isConsumed: false,
			want:       false,
		},
		// TransitionBased (current impl),
		"Current impl shall remove closed, consumed EEs": {
			rotten:     false,
			closed:     true,
			useLegacy:  false,
			isConsumed: true,
			want:       true,
		},
		"Current impl shall keep closed, unconsumed EEs": {
			rotten:     false,
			closed:     true,
			useLegacy:  false,
			isConsumed: false,
			want:       false,
		},
		"Current impl shall remove open, consumed EEs": { // difference to legacy
			rotten:     false,
			closed:     false,
			useLegacy:  false,
			isConsumed: true,
			want:       true,
		},
		"Current impl shall keep open, unconsumed EEs": {
			rotten:     false,
			closed:     false,
			useLegacy:  false,
			isConsumed: false,
			want:       false,
		},
		// Rotten
		"Legacy shall remove rotten": {
			rotten:     true,
			closed:     false,
			useLegacy:  true,
			isConsumed: false,
			want:       true,
		},
		"Current impl shall remove rotten": {
			rotten:     true,
			closed:     false,
			useLegacy:  false,
			isConsumed: false,
			want:       true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ts := timestamp.InfiniteFuture
			if tt.closed {
				ts = timestamp.Now()
			}
			c := &connStatus{
				rotten:   tt.rotten,
				lastSeen: ts,
			}
			assert.Equalf(t, tt.want,
				c.checkRemoveCondition(tt.useLegacy, tt.isConsumed),
				"checkRemoveCondition(%v, %v)", tt.useLegacy, tt.isConsumed)
		})
	}
}
