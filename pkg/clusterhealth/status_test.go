package clusterhealth

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestGetSensorStatus(t *testing.T) {
	cases := []struct {
		name            string
		previousContact time.Time
		newContact      time.Time
		expectedStatus  storage.ClusterHealthStatus_HealthStatusLabel
	}{
		{
			name:            "sensor never connected",
			previousContact: time.Time{},
			newContact:      time.Time{},
			expectedStatus:  storage.ClusterHealthStatus_UNINITIALIZED,
		},
		{
			name:            "first ever sensor contact",
			previousContact: time.Time{},
			newContact:      time.Now(),
			expectedStatus:  storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name:            "sensor contact: still healthy",
			previousContact: time.Now().Add(-45 * time.Second),
			newContact:      time.Now(),
			expectedStatus:  storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name:            "no sensor contact: still healthy",
			previousContact: time.Now().Add(-50 * time.Second),
			newContact:      time.Time{},
			expectedStatus:  storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name:            "no sensor contact: healthy to degraded",
			previousContact: time.Now().Add(-120 * time.Second),
			newContact:      time.Time{},
			expectedStatus:  storage.ClusterHealthStatus_DEGRADED,
		},
		{
			name:            "no sensor contact: still degraded",
			previousContact: time.Now().Add(-170 * time.Second),
			newContact:      time.Time{},
			expectedStatus:  storage.ClusterHealthStatus_DEGRADED,
		},
		{
			name:            "no sensor contact: degraded to unhealthy",
			previousContact: time.Now().Add(-4 * time.Minute),
			newContact:      time.Time{},
			expectedStatus:  storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name:            "no sensor contact: still unhealthy",
			previousContact: time.Now().Add(-1 * time.Hour),
			newContact:      time.Time{},
			expectedStatus:  storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name:            "sensor contact: unhealthy to healthy",
			previousContact: time.Now().Add(-1 * time.Hour),
			newContact:      time.Now(),
			expectedStatus:  storage.ClusterHealthStatus_HEALTHY,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expectedStatus, PopulateSensorStatus(c.previousContact, c.newContact))
		})
	}

}

func TestCollectorStatus(t *testing.T) {
	cases := []struct {
		name                string
		collectorHealthInfo *storage.CollectorHealthInfo
		expectedStatus      storage.ClusterHealthStatus_HealthStatusLabel
	}{
		{
			name:           "collector: no data",
			expectedStatus: storage.ClusterHealthStatus_UNINITIALIZED,
		},
		{
			name: "collector: unhealthy",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPods: 0,
				TotalReadyPods:   5,
			},
			expectedStatus: storage.ClusterHealthStatus_UNINITIALIZED,
		},
		{
			name: "collector: healthy",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPods: 10,
				TotalReadyPods:   10,
			},
			expectedStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name: "collector: degraded",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPods: 10,
				TotalReadyPods:   9,
			},
			expectedStatus: storage.ClusterHealthStatus_DEGRADED,
		},
		{
			name: "collector: unhealthy",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPods: 10,
				TotalReadyPods:   5,
			},
			expectedStatus: storage.ClusterHealthStatus_UNHEALTHY,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expectedStatus, PopulateCollectorStatus(c.collectorHealthInfo))
		})
	}
}

func TestOverallHealth(t *testing.T) {
	cases := []struct {
		name     string
		health   *storage.ClusterHealthStatus
		expected storage.ClusterHealthStatus_HealthStatusLabel
	}{
		{
			name: "sensor degraded, collector unhealthy",
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_DEGRADED,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name: "sensor unhealthy, collector degraded",
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_UNHEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_DEGRADED,
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name: "sensor degraded, collector healthy",
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_DEGRADED,
				CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			},
			expected: storage.ClusterHealthStatus_DEGRADED,
		},
		{
			name: "sensor healthy, collector degraded",
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_DEGRADED,
			},
			expected: storage.ClusterHealthStatus_DEGRADED,
		},
		{
			name: "sensor healthy, collector unavailable",
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
			},
			expected: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name: "sensor healthy, collector healthy",
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			},
			expected: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name: "sensor unintialized, collector unhealthy: unexpected states",
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_UNINITIALIZED,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
			},
			expected: storage.ClusterHealthStatus_UNINITIALIZED,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, PopulateOverallClusterStatus(c.health))
		})
	}

}
