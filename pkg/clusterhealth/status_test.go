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
			assert.Equal(t, c.expectedStatus, GetSensorStatus(c.previousContact, c.newContact))
		})
	}

}

func TestCollectorStatus(t *testing.T) {
	cases := []struct {
		name           string
		desired        int64
		ready          int64
		expectedStatus storage.ClusterHealthStatus_HealthStatusLabel
	}{
		{
			name:           "collector: no data",
			expectedStatus: storage.ClusterHealthStatus_UNINITIALIZED,
		},
		{
			name:           "collector: unhealthy",
			desired:        0,
			ready:          5,
			expectedStatus: storage.ClusterHealthStatus_UNINITIALIZED,
		},
		{
			name:           "collector: healthy",
			desired:        10,
			ready:          10,
			expectedStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name:           "collector: degraded",
			desired:        10,
			ready:          9,
			expectedStatus: storage.ClusterHealthStatus_DEGRADED,
		},
		{
			name:           "collector: unhealthy",
			desired:        10,
			ready:          5,
			expectedStatus: storage.ClusterHealthStatus_UNHEALTHY,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expectedStatus, GetCollectorStatus(c.desired, c.ready))
		})
	}
}
