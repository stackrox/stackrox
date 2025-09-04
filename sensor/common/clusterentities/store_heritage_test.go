package clusterentities

import (
	"context"
	"maps"
	"testing"
	"time"

	"slices"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/sensor/common/heritage"
	"github.com/stretchr/testify/assert"
)

// mockHeritageManager implements HeritageManager for testing
type mockHeritageManager struct {
	data               []*heritage.SensorMetadata
	currentPodIP       string
	currentContainerID string
	setCalled          bool
}

func (m *mockHeritageManager) GetData(ctx context.Context) []*heritage.SensorMetadata {
	return m.data
}

func (m *mockHeritageManager) SetCurrentSensorData(podIP, containerID string) {
	m.currentPodIP = podIP
	m.currentContainerID = containerID
	m.setCalled = true
}

func TestStore_ApplyHeritageDataOnce(t *testing.T) {
	tests := map[string]struct {
		setupPastData        []*heritage.SensorMetadata
		setupCurrentMetadata bool
		expectedSignalDone   bool
	}{
		"should signal done when heritage data applied successfully once": {
			setupPastData: []*heritage.SensorMetadata{
				{ContainerID: "past123", PodIP: "10.1.1.1", SensorStart: time.Now().Add(-time.Hour)},
			},
			setupCurrentMetadata: true,
			expectedSignalDone:   true,
		},
		"should not signal done when no heritage data available": {
			setupPastData:        []*heritage.SensorMetadata{},
			setupCurrentMetadata: true,
			expectedSignalDone:   false,
		},
		"should not signal done when missing current sensor metadata": {
			setupPastData: []*heritage.SensorMetadata{
				{ContainerID: "past123", PodIP: "10.1.1.1", SensorStart: time.Now().Add(-time.Hour)},
			},
			setupCurrentMetadata: false,
			expectedSignalDone:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockHM := &mockHeritageManager{data: tt.setupPastData}
			store := NewStore(0, mockHM, true)

			// Setup current sensor metadata if required
			if tt.setupCurrentMetadata {
				currentData := createSensorEntityData("current123", "10.2.2.2")
				store.RememberCurrentSensorMetadata("sensor-deploy-1", currentData)
			}

			// Call multiple times to verify single execution
			store.ApplyHeritageDataOnce()
			store.ApplyHeritageDataOnce()
			store.ApplyHeritageDataOnce()

			// Verify signal state
			assert.Equal(t, tt.expectedSignalDone, store.heritageApplied.IsDone())
		})
	}
}

func TestApplyPastToEntityData(t *testing.T) {
	tests := map[string]struct {
		currentData    *EntityData
		pastSensor     *heritage.SensorMetadata
		expectedResult bool

		expectedIPs   []net.IPAddress
		unexpectedIPs []net.IPAddress

		expectedEndpoints []net.NumericEndpoint

		expectedContainerIDs   []string
		unexpectedContainerIDs []string
	}{
		"should add new heritage data and return true": {
			currentData:            createSensorEntityData("current123", "10.2.2.2"),
			pastSensor:             &heritage.SensorMetadata{ContainerID: "past456", PodIP: "10.1.1.1"},
			expectedResult:         true,
			expectedIPs:            []net.IPAddress{net.ParseIP("10.2.2.2"), net.ParseIP("10.1.1.1")},
			unexpectedIPs:          []net.IPAddress{},
			expectedContainerIDs:   []string{"current123", "past456"},
			unexpectedContainerIDs: []string{},
		},
		"should skip existing container and return false": {
			currentData:            createSensorEntityData("duplicate123", "10.2.2.2"),
			pastSensor:             &heritage.SensorMetadata{ContainerID: "duplicate123", PodIP: "10.1.1.1"}, // Same container ID
			expectedResult:         false,
			expectedIPs:            []net.IPAddress{net.ParseIP("10.2.2.2")},
			unexpectedIPs:          []net.IPAddress{net.ParseIP("10.1.1.1")},
			expectedContainerIDs:   []string{"duplicate123"},
			unexpectedContainerIDs: []string{},
		},
		"should generate heritage endpoints for past IP": {
			currentData: func() *EntityData {
				data := createSensorEntityData("current123", "10.2.2.2")
				// Add some endpoints to current data
				data.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 8443, net.TCP), EndpointTargetInfo{ContainerPort: 8443})
				data.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 9090, net.TCP), EndpointTargetInfo{ContainerPort: 9090})
				return data
			}(),
			pastSensor:     &heritage.SensorMetadata{ContainerID: "past456", PodIP: "10.1.1.1"},
			expectedResult: true,
			expectedIPs:    []net.IPAddress{net.ParseIP("10.2.2.2"), net.ParseIP("10.1.1.1")},
			unexpectedIPs:  []net.IPAddress{},
			expectedEndpoints: []net.NumericEndpoint{
				net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 8443, net.TCP),
				net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 9090, net.TCP),
				net.MakeNumericEndpoint(net.ParseIP("10.1.1.1"), 8443, net.TCP),
				net.MakeNumericEndpoint(net.ParseIP("10.1.1.1"), 9090, net.TCP),
			},
			expectedContainerIDs:   []string{"current123", "past456"},
			unexpectedContainerIDs: []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := applyPastToEntityData(tt.currentData, tt.pastSensor)
			assert.Equal(t, tt.expectedResult, result)
			containerIDs, podIPs := tt.currentData.GetDetails()
			gotEndpoints := slices.Collect(maps.Keys(tt.currentData.endpoints))

			// Sort before asserting with ElementsMatch
			slices.SortFunc(podIPs, net.IPAddressCompare)
			slices.SortFunc(tt.expectedIPs, net.IPAddressCompare)

			slices.Sort(containerIDs)
			slices.Sort(tt.expectedContainerIDs)
			slices.SortFunc(tt.expectedEndpoints, net.NumericEndpointCompare)
			slices.SortFunc(gotEndpoints, net.NumericEndpointCompare)

			assert.ElementsMatch(t, tt.expectedIPs, podIPs, "IP should be added")
			assert.ElementsMatch(t, tt.expectedEndpoints, gotEndpoints, "Endpoints should be added")
			for _, ip := range tt.unexpectedIPs {
				assert.NotContains(t, podIPs, ip, "IP should not be added")
			}
			assert.ElementsMatch(t, tt.expectedContainerIDs, containerIDs, "Container IDs should be added")

			for _, containerID := range tt.unexpectedContainerIDs {
				assert.NotContains(t, containerIDs, containerID, "Container ID should not be added")
			}
		})
	}
}

func TestEntityData_String_SlicesCollectFix(t *testing.T) {
	// Test justification: Validates the slices.Collect fix for proper formatting
	tests := map[string]struct {
		setupData  func() *EntityData
		expectsNil bool
	}{
		"should format output with multiple elements properly": {
			setupData: func() *EntityData {
				data := &EntityData{}
				data.AddIP(net.ParseIP("10.1.1.1"))
				data.AddIP(net.ParseIP("10.2.2.2"))
				data.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.1.1.1"), 8443, net.TCP), EndpointTargetInfo{})
				data.AddContainerID("container123", ContainerMetadata{})
				return data
			},
			expectsNil: false,
		},
		"should return nil string for nil entity data": {
			setupData: func() *EntityData {
				return nil
			},
			expectsNil: true,
		},
		"should format empty entity data without nil string": {
			setupData: func() *EntityData {
				return &EntityData{}
			},
			expectsNil: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			data := tt.setupData()
			result := data.String()

			if tt.expectsNil {
				assert.Equal(t, "nil", result)
			} else {
				// Verify string contains expected structure - validates slices.Collect fix
				assert.Contains(t, result, "ips:")
				assert.Contains(t, result, "endpoints:")
				assert.Contains(t, result, "containerIDs:")
				assert.NotEmpty(t, result)
				assert.NotContains(t, result, "0x") // Should not contain memory addresses
			}
		})
	}
}

// Helper functions for test setup

func createSensorEntityData(containerID, podIP string) *EntityData {
	data := &EntityData{}
	data.AddIP(net.ParseIP(podIP))
	data.AddContainerID(containerID, ContainerMetadata{
		ContainerName: "sensor",
		ContainerID:   containerID,
	})
	return data
}
