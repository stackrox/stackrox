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
	tests := map[string]struct {
		setupConnection    func() (*connection, *connStatus)
		setupMocks         func(*mockExpectations)
		setupFeatureFlags  func(*testing.T)
		expectedResult     EnrichmentResult
		expectedReason     EnrichmentReasonConn
		validateEnrichment func(*testing.T, map[networkConnIndicator]timestamp.MicroTS)
	}{
		"IP parsing error - malformed address": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    false,
					remote: net.NumericEndpoint{
						IPAndPort: net.NetworkPeerID{
							Address: net.ParseIP("invalid-ip"), // This will cause IsExternal() to fail
							Port:    80,
						},
						L4Proto: net.TCP,
					},
				}
				status := &connStatus{
					firstSeen:             timestamp.Now().Add(-time.Minute), // not fresh
					lastSeen:              timestamp.Now(),
					enrichmentConsumption: enrichmentConsumption{},
				}
				return conn, status
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment").expectEndpointNotFound().expectExternalNotFound()
			},
			expectedResult: EnrichmentResultInvalidInput,
			expectedReason: EnrichmentReasonConnParsingIPFailed,
		},
		"External source network found": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    false,
					remote: net.NumericEndpoint{
						IPAndPort: net.NetworkPeerID{
							Address: net.ParseIP("192.168.1.100"), // internal but with external source
							Port:    80,
						},
						L4Proto: net.TCP,
					},
				}
				status := &connStatus{
					firstSeen:             timestamp.Now().Add(-time.Minute), // not fresh
					lastSeen:              timestamp.Now(),
					enrichmentConsumption: enrichmentConsumption{},
				}
				return conn, status
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment").expectEndpointNotFound().expectExternalFound("external-network-id")
			},
			expectedResult: EnrichmentResultSuccess,
			expectedReason: EnrichmentReasonConnSuccess,
			validateEnrichment: func(t *testing.T, enriched map[networkConnIndicator]timestamp.MicroTS) {
				assert.Len(t, enriched, 1, "Should have one enriched connection")
				for indicator := range enriched {
					assert.Equal(t, "external-network-id", indicator.dstEntity.ID, "Should use external source entity")
				}
			},
		},
		"Incoming local connection - should be skipped": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    true, // incoming connection
					remote: net.NumericEndpoint{
						IPAndPort: net.NetworkPeerID{
							Address: net.ParseIP("10.0.0.1"), // internal IP
							Port:    80,
						},
						L4Proto: net.TCP,
					},
					local: net.NetworkPeerID{
						Port: 8080,
					},
				}
				status := &connStatus{
					firstSeen:             timestamp.Now().Add(-time.Minute), // not fresh
					lastSeen:              timestamp.Now(),
					enrichmentConsumption: enrichmentConsumption{},
				}
				return conn, status
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment").expectEndpointFound("local-endpoint-id", 8080)
			},
			expectedResult: EnrichmentResultSkipped,
			expectedReason: EnrichmentReasonConnIncomingInternalConnection,
		},
		"External connection with Internet entity fallback": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    false,
					remote: net.NumericEndpoint{
						IPAndPort: net.NetworkPeerID{
							Address: net.ExternalIPv4Addr, // considered external
							Port:    80,
						},
						L4Proto: net.TCP,
					},
				}
				status := &connStatus{
					firstSeen:             timestamp.Now().Add(-time.Minute), // not fresh
					lastSeen:              timestamp.Now(),
					enrichmentConsumption: enrichmentConsumption{},
				}
				return conn, status
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment").expectExternalNotFound()
			},
			expectedResult: EnrichmentResultSuccess,
			expectedReason: EnrichmentReasonConnSuccess,
			validateEnrichment: func(t *testing.T, enriched map[networkConnIndicator]timestamp.MicroTS) {
				assert.Len(t, enriched, 1, "Should have one enriched connection")
				for indicator := range enriched {
					assert.Equal(t, networkgraph.InternetEntity().ID, indicator.dstEntity.ID, "Should use Internet entity")
				}
			},
		},
		"Internal connection with fallback entity": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    false,
					remote: net.NumericEndpoint{
						IPAndPort: net.NetworkPeerID{
							Address: net.ParseIP("10.0.0.1"), // internal IP
							Port:    80,
						},
						L4Proto: net.TCP,
					},
				}
				status := &connStatus{
					firstSeen:             timestamp.Now().Add(-time.Minute), // not fresh
					lastSeen:              timestamp.Now(),
					enrichmentConsumption: enrichmentConsumption{},
				}
				return conn, status
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment").expectEndpointNotFound().expectExternalNotFound()
			},
			expectedResult: EnrichmentResultSuccess,
			expectedReason: EnrichmentReasonConnSuccess,
			validateEnrichment: func(t *testing.T, enriched map[networkConnIndicator]timestamp.MicroTS) {
				assert.Len(t, enriched, 1, "Should have one enriched connection")
				// For internal connections without external source, it creates a fallback entity
				// This tests the fallback behavior for unknown internal addresses
				for indicator := range enriched {
					// Should enrich successfully regardless of entity type
					assert.NotEmpty(t, indicator.dstEntity.ID, "Should have a valid destination entity")
				}
			},
		},
		"Connection with SensorCapturesIntermediateEvents disabled": {
			setupConnection: func() (*connection, *connStatus) {
				conn := &connection{
					containerID: "test-container",
					incoming:    false,
					remote: net.NumericEndpoint{
						IPAndPort: net.NetworkPeerID{
							Address: net.ParseIP("8.8.8.8"),
							Port:    80,
						},
						L4Proto: net.TCP,
					},
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
			validateEnrichment: func(t *testing.T, enriched map[networkConnIndicator]timestamp.MicroTS) {
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
			enrichedConnections := make(map[networkConnIndicator]timestamp.MicroTS)

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
	tests := map[string]struct {
		setupEndpoint      func() (*containerEndpoint, *connStatus)
		setupMocks         func(*mockExpectations)
		setupFeatureFlags  func(*testing.T)
		expectedResultNG   EnrichmentResult
		expectedResultPLOP EnrichmentResult
		expectedReasonNG   EnrichmentReasonEp
		validateEnrichment func(*testing.T, map[containerEndpointIndicator]timestamp.MicroTS, map[processListeningIndicator]timestamp.MicroTS)
	}{
		"Fresh endpoint with no process info - PLOP should fail": {
			setupEndpoint: func() (*containerEndpoint, *connStatus) {
				ep := &containerEndpoint{
					endpoint: net.NumericEndpoint{
						IPAndPort: net.NetworkPeerID{
							Address: net.ParseIP("8.8.8.8"),
							Port:    80,
						},
						L4Proto: net.TCP,
					},
					containerID: "test-container",
					processKey:  processInfo{}, // empty process info
				}
				status := &connStatus{
					firstSeen:             timestamp.Now(), // fresh
					lastSeen:              timestamp.Now(),
					enrichmentConsumption: enrichmentConsumption{},
				}
				return ep, status
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment")
			},
			setupFeatureFlags: func(t *testing.T) {
				t.Setenv(env.ProcessesListeningOnPort.EnvVar(), "true")
			},
			expectedResultNG:   EnrichmentResultSuccess,
			expectedResultPLOP: EnrichmentResultInvalidInput,
			expectedReasonNG:   EnrichmentReasonEpSuccessInactive,
		},
		"Endpoint with duplicate timestamp - should be marked as duplicate": {
			setupEndpoint: func() (*containerEndpoint, *connStatus) {
				ep := &containerEndpoint{
					endpoint: net.NumericEndpoint{
						IPAndPort: net.NetworkPeerID{
							Address: net.ParseIP("8.8.8.8"),
							Port:    80,
						},
						L4Proto: net.TCP,
					},
					containerID: "test-container",
					processKey:  defaultProcessKey(),
				}
				status := &connStatus{
					firstSeen:             timestamp.Now().Add(-time.Minute),
					lastSeen:              timestamp.Now().Add(-time.Minute), // older timestamp
					enrichmentConsumption: enrichmentConsumption{},
				}
				return ep, status
			},
			setupMocks: func(m *mockExpectations) {
				m.expectContainerFound("test-deployment")
			},
			expectedResultNG:   EnrichmentResultSuccess,
			expectedResultPLOP: EnrichmentResultSuccess,
			expectedReasonNG:   EnrichmentReasonEpDuplicate,
			validateEnrichment: func(t *testing.T, enrichedEndpoints map[containerEndpointIndicator]timestamp.MicroTS, processesListening map[processListeningIndicator]timestamp.MicroTS) {
				// Pre-populate with newer timestamp to trigger duplicate detection
				indicator := containerEndpointIndicator{
					entity:   networkgraph.EntityForDeployment("test-deployment"),
					port:     80,
					protocol: net.TCP.ToProtobuf(),
				}
				enrichedEndpoints[indicator] = timestamp.Now() // newer timestamp
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

			m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)

			// Setup mocks
			mocks := newMockExpectations(mockEntityStore, nil)
			tt.setupMocks(mocks)

			// Setup test data
			ep, status := tt.setupEndpoint()
			enrichedEndpoints := make(map[containerEndpointIndicator]timestamp.MicroTS)
			processesListening := make(map[processListeningIndicator]timestamp.MicroTS)

			// Pre-populate data if validation function needs it
			if tt.validateEnrichment != nil {
				tt.validateEnrichment(t, enrichedEndpoints, processesListening)
			}

			// Execute the enrichment
			resultNG, resultPLOP, reasonNG, _ := m.enrichContainerEndpoint(
				timestamp.Now(), ep, status, enrichedEndpoints, processesListening, timestamp.Now())

			// Assert results
			assert.Equal(t, tt.expectedResultNG, resultNG, "Network graph enrichment result mismatch")
			assert.Equal(t, tt.expectedResultPLOP, resultPLOP, "PLOP enrichment result mismatch")
			assert.Equal(t, tt.expectedReasonNG, reasonNG, "Network graph enrichment reason mismatch")

			// Additional validation can be added here for specific test cases
		})
	}
}
