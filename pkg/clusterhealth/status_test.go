package clusterhealth

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestPopulateInactiveSensorStatus(t *testing.T) {
	cases := []struct {
		name           string
		lastContact    time.Time
		expectedStatus storage.ClusterHealthStatus_HealthStatusLabel
	}{
		{
			name:           "sensor never connected",
			lastContact:    time.Time{},
			expectedStatus: storage.ClusterHealthStatus_UNINITIALIZED,
		},
		{
			name:           "first ever sensor contact",
			lastContact:    time.Now(),
			expectedStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name:           "sensor contact: still healthy",
			lastContact:    time.Now().Add(-45 * time.Second),
			expectedStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name:           "no sensor contact: still healthy",
			lastContact:    time.Now().Add(-50 * time.Second),
			expectedStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name:           "no sensor contact: healthy to degraded",
			lastContact:    time.Now().Add(-120 * time.Second),
			expectedStatus: storage.ClusterHealthStatus_DEGRADED,
		},
		{
			name:           "no sensor contact: still degraded",
			lastContact:    time.Now().Add(-170 * time.Second),
			expectedStatus: storage.ClusterHealthStatus_DEGRADED,
		},
		{
			name:           "no sensor contact: degraded to unhealthy",
			lastContact:    time.Now().Add(-4 * time.Minute),
			expectedStatus: storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name:           "no sensor contact: still unhealthy",
			lastContact:    time.Now().Add(-1 * time.Hour),
			expectedStatus: storage.ClusterHealthStatus_UNHEALTHY,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expectedStatus, PopulateInactiveSensorStatus(c.lastContact))
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
			name: "collector: uninitialized - 5/0",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPodsOpt: podsDesired(0),
				TotalReadyPodsOpt:   podsReady(5),
			},
			expectedStatus: storage.ClusterHealthStatus_UNINITIALIZED,
		},
		{
			name: "collector: uninitialized - 0/0",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPodsOpt: podsDesired(0),
				TotalReadyPodsOpt:   podsReady(0),
			},
			expectedStatus: storage.ClusterHealthStatus_UNINITIALIZED,
		},
		{
			name: "collector: healthy - 10/10",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPodsOpt: podsDesired(10),
				TotalReadyPodsOpt:   podsReady(10),
			},
			expectedStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name: "collector: healthy - 12/10 (anomaly)",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPodsOpt: podsDesired(10),
				TotalReadyPodsOpt:   podsReady(12),
			},
			expectedStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name: "collector: degraded - 9/10",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPodsOpt: podsDesired(10),
				TotalReadyPodsOpt:   podsReady(9),
			},
			expectedStatus: storage.ClusterHealthStatus_DEGRADED,
		},
		{
			name: "collector: unhealthy - 5/10",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPodsOpt: podsDesired(10),
				TotalReadyPodsOpt:   podsReady(5),
			},
			expectedStatus: storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name: "collector: unhealthy - 10/n.a. can't get count of desired pods",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPodsOpt: nil,
				TotalReadyPodsOpt:   podsReady(10),
			},
			expectedStatus: storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name: "collector: unhealthy - n.a./10 can't get count of ready pods",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPodsOpt: podsDesired(10),
				TotalReadyPodsOpt:   nil,
			},
			expectedStatus: storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name: "collector: unhealthy - n.a./0 can't get count of ready pods",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPodsOpt: podsDesired(0),
				TotalReadyPodsOpt:   nil,
			},
			expectedStatus: storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name: "collector: unhealthy - n.a./n.a. can't get both counts",
			collectorHealthInfo: &storage.CollectorHealthInfo{
				TotalDesiredPodsOpt: nil,
				TotalReadyPodsOpt:   nil,
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

func podsDesired(num int32) *storage.CollectorHealthInfo_TotalDesiredPods {
	return &storage.CollectorHealthInfo_TotalDesiredPods{TotalDesiredPods: num}
}
func podsReady(num int32) *storage.CollectorHealthInfo_TotalReadyPods {
	return &storage.CollectorHealthInfo_TotalReadyPods{TotalReadyPods: num}
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

func TestLocalScannerHealth(t *testing.T) {

	oneReadyPod := &storage.ScannerHealthInfo_TotalReadyAnalyzerPods{TotalReadyAnalyzerPods: 1}
	twoReadyPod := &storage.ScannerHealthInfo_TotalReadyAnalyzerPods{TotalReadyAnalyzerPods: 2}
	oneDesiredPod := &storage.ScannerHealthInfo_TotalDesiredAnalyzerPods{TotalDesiredAnalyzerPods: 1}
	threeDesiredPod := &storage.ScannerHealthInfo_TotalDesiredAnalyzerPods{TotalDesiredAnalyzerPods: 3}
	oneReadyDBPod := &storage.ScannerHealthInfo_TotalReadyDbPods{TotalReadyDbPods: 1}
	oneDesiredDBPod := &storage.ScannerHealthInfo_TotalDesiredDbPods{TotalDesiredDbPods: 1}

	oneReadyV4Pod := &storage.ScannerHealthInfo_TotalReadyV4IndexerPods{TotalReadyV4IndexerPods: 1}
	twoReadyV4Pod := &storage.ScannerHealthInfo_TotalReadyV4IndexerPods{TotalReadyV4IndexerPods: 2}
	oneDesiredV4Pod := &storage.ScannerHealthInfo_TotalDesiredV4IndexerPods{TotalDesiredV4IndexerPods: 1}
	threeDesiredV4Pod := &storage.ScannerHealthInfo_TotalDesiredV4IndexerPods{TotalDesiredV4IndexerPods: 3}

	oneReadyV4DBPod := &storage.ScannerHealthInfo_TotalReadyV4DbPods{TotalReadyV4DbPods: 1}
	oneDesiredV4DBPod := &storage.ScannerHealthInfo_TotalDesiredV4DbPods{TotalDesiredV4DbPods: 1}

	cases := []struct {
		name     string
		health   *storage.ScannerHealthInfo
		expected storage.ClusterHealthStatus_HealthStatusLabel
	}{
		{
			name:     "nil health info",
			health:   nil,
			expected: storage.ClusterHealthStatus_UNINITIALIZED,
		},
		{
			name:     "empty health info",
			health:   &storage.ScannerHealthInfo{},
			expected: storage.ClusterHealthStatus_UNINITIALIZED,
		},
		{
			name: "scanner v2 db pods not ready",
			health: &storage.ScannerHealthInfo{
				TotalDesiredDbPodsOpt:       oneDesiredDBPod,
				TotalReadyAnalyzerPodsOpt:   oneReadyPod,
				TotalDesiredAnalyzerPodsOpt: oneDesiredPod,
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name: "scanner v4 db pods not ready",
			health: &storage.ScannerHealthInfo{
				TotalDesiredV4DbPodsOpt:      oneDesiredV4DBPod,
				TotalReadyV4IndexerPodsOpt:   oneReadyV4Pod,
				TotalDesiredV4IndexerPodsOpt: oneDesiredV4Pod,
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name: "scanner v2 unhealthy",
			health: &storage.ScannerHealthInfo{
				TotalReadyDbPodsOpt:         oneReadyDBPod,
				TotalDesiredDbPodsOpt:       oneDesiredDBPod,
				TotalReadyAnalyzerPodsOpt:   oneReadyPod,
				TotalDesiredAnalyzerPodsOpt: threeDesiredPod,
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name: "scanner v2 degraded",
			health: &storage.ScannerHealthInfo{
				TotalReadyDbPodsOpt:         oneReadyDBPod,
				TotalDesiredDbPodsOpt:       oneDesiredDBPod,
				TotalReadyAnalyzerPodsOpt:   twoReadyPod,
				TotalDesiredAnalyzerPodsOpt: threeDesiredPod,
			},
			expected: storage.ClusterHealthStatus_DEGRADED,
		},
		{
			name: "scanner v4 unhealthy",
			health: &storage.ScannerHealthInfo{
				TotalReadyV4DbPodsOpt:        oneReadyV4DBPod,
				TotalDesiredV4DbPodsOpt:      oneDesiredV4DBPod,
				TotalReadyV4IndexerPodsOpt:   oneReadyV4Pod,
				TotalDesiredV4IndexerPodsOpt: threeDesiredV4Pod,
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		{
			name: "scanner v4 degraded",
			health: &storage.ScannerHealthInfo{
				TotalReadyV4DbPodsOpt:        oneReadyV4DBPod,
				TotalDesiredV4DbPodsOpt:      oneDesiredV4DBPod,
				TotalReadyV4IndexerPodsOpt:   twoReadyV4Pod,
				TotalDesiredV4IndexerPodsOpt: threeDesiredV4Pod,
			},
			expected: storage.ClusterHealthStatus_DEGRADED,
		},
		{
			name: "scanner v2 healthy",
			health: &storage.ScannerHealthInfo{
				TotalReadyDbPodsOpt:         oneReadyDBPod,
				TotalDesiredDbPodsOpt:       oneDesiredDBPod,
				TotalReadyAnalyzerPodsOpt:   oneReadyPod,
				TotalDesiredAnalyzerPodsOpt: oneDesiredPod,
			},
			expected: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			name: "scanner v4 healthy",
			health: &storage.ScannerHealthInfo{
				TotalReadyV4DbPodsOpt:        oneReadyV4DBPod,
				TotalDesiredV4DbPodsOpt:      oneDesiredV4DBPod,
				TotalReadyV4IndexerPodsOpt:   oneReadyV4Pod,
				TotalDesiredV4IndexerPodsOpt: oneDesiredV4Pod,
			},
			expected: storage.ClusterHealthStatus_HEALTHY,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, PopulateLocalScannerStatus(c.health))
		})
	}
}
