package manager

import (
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
		validateEnrichment func(*testing.T, map[networkConnIndicator]timestamp.MicroTS)
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
			validateEnrichment: func(t *testing.T, enriched map[networkConnIndicator]timestamp.MicroTS) {
				assert.Len(t, enriched, 1, "Should have one enriched connection")
				for indicator := range enriched {
					assert.Equal(t, "external-network-id", indicator.dstEntity.ID, "Should use external source entity")
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
			validateEnrichment: func(t *testing.T, enriched map[networkConnIndicator]timestamp.MicroTS) {
				assert.Len(t, enriched, 1, "Should have one enriched connection")
				for indicator := range enriched {
					assert.Equal(t, networkgraph.InternetEntity().ID, indicator.dstEntity.ID, "Should use Internet entity")
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
			result, reason := m.connectionManager.enrichConnection(timestamp.Now(), conn, status, enrichedConnections)

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

