package resources

import (
	"testing"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractHeritageData(t *testing.T) {
	// Test justification: Validates heritage data extraction logic for Sensor deployments
	tests := map[string]struct {
		entityData          *clusterentities.EntityData
		expectedContainerID string
		expectedPodIP       string
		expectedError       string
	}{
		"should extract heritage data successfully": {
			entityData:          createTestSensorEntityData([]string{"container123"}, []string{"10.1.1.1"}),
			expectedContainerID: "container123",
			expectedPodIP:       "10.1.1.1",
		},
		"should return error for nil entity data": {
			entityData:    nil,
			expectedError: "Empty entity data",
		},
		"should return error when no container IDs found": {
			entityData:    createTestSensorEntityData([]string{}, []string{"10.1.1.1"}),
			expectedError: "No container IDs found",
		},
		"should return error when no pod IPs found": {
			entityData:    createTestSensorEntityData([]string{"container123"}, []string{}),
			expectedError: "No pod IPs found",
		},
		"should select first container alphabetically from multiple containers": {
			entityData:          createTestSensorEntityData([]string{"zzz-container", "aaa-container"}, []string{"10.1.1.1"}),
			expectedContainerID: "aaa-container", // Should pick first alphabetically
			expectedPodIP:       "10.1.1.1",
		},
		"should select first IP in sorted order from multiple IPs": {
			entityData:          createTestSensorEntityData([]string{"container123"}, []string{"10.2.2.2", "10.1.1.1"}),
			expectedContainerID: "container123",
			expectedPodIP:       "10.1.1.1", // Should pick first in sorted order
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			containerID, podIP, err := extractHeritageData(tt.entityData)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedContainerID, containerID)
				assert.Equal(t, tt.expectedPodIP, podIP)
			}
		})
	}
}

// Helper functions for test setup
func createTestSensorEntityData(containerIDs, podIPs []string) *clusterentities.EntityData {
	data := &clusterentities.EntityData{}
	for _, containerID := range containerIDs {
		data.AddContainerID(containerID, clusterentities.ContainerMetadata{
			ContainerName: "sensor",
			ContainerID:   containerID,
		})
	}
	for _, podIP := range podIPs {
		data.AddIP(net.ParseIP(podIP))
	}

	return data
}
