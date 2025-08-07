package manager

import (
	"strconv"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
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
		prePopulateData    func(*testing.T, map[containerEndpointIndicator]timestamp.MicroTS, map[processListeningIndicator]timestamp.MicroTS)
	}{
		"Fresh endpoint with no process info should yield result EnrichmentResultSuccess for Network Graph and EnrichmentResultInvalidInput for PLOP": {
			setupEndpoint: func() (*containerEndpoint, *connStatus) {
				ep := &containerEndpoint{
					endpoint:    commonEndpoint,
					containerID: "test-container",
					processKey:  processInfo{}, // empty process info
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
			prePopulateData: func(t *testing.T, enrichedEndpoints map[containerEndpointIndicator]timestamp.MicroTS, processesListening map[processListeningIndicator]timestamp.MicroTS) {
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
			if tt.prePopulateData != nil {
				tt.prePopulateData(t, enrichedEndpoints, processesListening)
			}

			// Execute the enrichment
			now := timestamp.Now()
			resultNG, resultPLOP, reasonNG, _ := m.enrichContainerEndpoint(
				now, ep, status, enrichedEndpoints, processesListening, now)

			// Assert results
			assert.Equal(t, tt.expectedResultNG, resultNG, "Network graph enrichment result mismatch")
			assert.Equal(t, tt.expectedResultPLOP, resultPLOP, "PLOP enrichment result mismatch")
			assert.Equal(t, tt.expectedReasonNG, reasonNG, "Network graph enrichment reason mismatch")

			// Additional validation can be added here for specific test cases
		})
	}
}
