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
	cases := map[string]struct {
		health   *storage.ClusterHealthStatus
		expected storage.ClusterHealthStatus_HealthStatusLabel
	}{
		"sensor degraded, collector unhealthy": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_DEGRADED,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		"sensor unhealthy, collector degraded": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_UNHEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_DEGRADED,
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		"sensor degraded, collector healthy": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_DEGRADED,
				CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			},
			expected: storage.ClusterHealthStatus_DEGRADED,
		},
		"sensor healthy, collector degraded": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_DEGRADED,
			},
			expected: storage.ClusterHealthStatus_DEGRADED,
		},
		"sensor healthy, collector unavailable": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
			},
			expected: storage.ClusterHealthStatus_HEALTHY,
		},
		"sensor healthy, collector healthy": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			},
			expected: storage.ClusterHealthStatus_HEALTHY,
		},
		"sensor uninitialized, collector unhealthy: unexpected states": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_UNINITIALIZED,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
			},
			expected: storage.ClusterHealthStatus_UNINITIALIZED,
		},
		"scanner uninitialized does not affect overall health": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				ScannerHealthStatus:   storage.ClusterHealthStatus_UNINITIALIZED,
			},
			expected: storage.ClusterHealthStatus_HEALTHY,
		},
		"scanner healthy does not degrade overall health": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				ScannerHealthStatus:   storage.ClusterHealthStatus_HEALTHY,
			},
			expected: storage.ClusterHealthStatus_HEALTHY,
		},
		"scanner unhealthy makes overall unhealthy": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				ScannerHealthStatus:   storage.ClusterHealthStatus_UNHEALTHY,
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		"scanner degraded makes overall degraded": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				ScannerHealthStatus:   storage.ClusterHealthStatus_DEGRADED,
			},
			expected: storage.ClusterHealthStatus_DEGRADED,
		},
		"scanner degraded with sensor unhealthy makes overall unhealthy": {
			health: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_UNHEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				ScannerHealthStatus:   storage.ClusterHealthStatus_DEGRADED,
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, PopulateOverallClusterStatus(c.health))
		})
	}
}

func TestPopulateLocalScannerStatus(t *testing.T) {
	cases := map[string]struct {
		scannerHealthInfo *storage.ScannerHealthInfo
		expected          storage.ClusterHealthStatus_HealthStatusLabel
	}{
		"nil: local scanning not enabled (optional component)": {
			scannerHealthInfo: nil,
			expected:          storage.ClusterHealthStatus_UNINITIALIZED,
		},
		"status errors: enabled but misconfigured (no scanner v4)": {
			scannerHealthInfo: &storage.ScannerHealthInfo{
				StatusErrors: []string{"local image scanning is enabled but Scanner V4 is not enabled"},
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		"status errors take precedence over pod counts": {
			scannerHealthInfo: &storage.ScannerHealthInfo{
				StatusErrors:                []string{"unable to find scanner deployment"},
				TotalDesiredAnalyzerPodsOpt: analyzerPodsDesired(3),
				TotalReadyAnalyzerPodsOpt:   analyzerPodsReady(3),
				TotalReadyDbPodsOpt:         dbPodsReady(1),
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		"no pod counts, no errors: uninitialized": {
			scannerHealthInfo: &storage.ScannerHealthInfo{},
			expected:          storage.ClusterHealthStatus_UNINITIALIZED,
		},
		"desired analyzer pods zero: uninitialized": {
			scannerHealthInfo: &storage.ScannerHealthInfo{
				TotalDesiredAnalyzerPodsOpt: analyzerPodsDesired(0),
				TotalReadyAnalyzerPodsOpt:   analyzerPodsReady(0),
			},
			expected: storage.ClusterHealthStatus_UNINITIALIZED,
		},
		"no ready db pods: unhealthy": {
			scannerHealthInfo: &storage.ScannerHealthInfo{
				TotalDesiredAnalyzerPodsOpt: analyzerPodsDesired(3),
				TotalReadyAnalyzerPodsOpt:   analyzerPodsReady(3),
				TotalReadyDbPodsOpt:         dbPodsReady(0),
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
		"all analyzer and db pods ready: healthy": {
			scannerHealthInfo: &storage.ScannerHealthInfo{
				TotalDesiredAnalyzerPodsOpt: analyzerPodsDesired(3),
				TotalReadyAnalyzerPodsOpt:   analyzerPodsReady(3),
				TotalReadyDbPodsOpt:         dbPodsReady(1),
			},
			expected: storage.ClusterHealthStatus_HEALTHY,
		},
		"some analyzer pods not ready: degraded": {
			scannerHealthInfo: &storage.ScannerHealthInfo{
				TotalDesiredAnalyzerPodsOpt: analyzerPodsDesired(3),
				TotalReadyAnalyzerPodsOpt:   analyzerPodsReady(2),
				TotalReadyDbPodsOpt:         dbPodsReady(1),
			},
			expected: storage.ClusterHealthStatus_DEGRADED,
		},
		"most analyzer pods not ready: unhealthy": {
			scannerHealthInfo: &storage.ScannerHealthInfo{
				TotalDesiredAnalyzerPodsOpt: analyzerPodsDesired(3),
				TotalReadyAnalyzerPodsOpt:   analyzerPodsReady(1),
				TotalReadyDbPodsOpt:         dbPodsReady(1),
			},
			expected: storage.ClusterHealthStatus_UNHEALTHY,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, PopulateLocalScannerStatus(c.scannerHealthInfo))
		})
	}
}

func analyzerPodsDesired(num int32) *storage.ScannerHealthInfo_TotalDesiredAnalyzerPods {
	return &storage.ScannerHealthInfo_TotalDesiredAnalyzerPods{TotalDesiredAnalyzerPods: num}
}
func analyzerPodsReady(num int32) *storage.ScannerHealthInfo_TotalReadyAnalyzerPods {
	return &storage.ScannerHealthInfo_TotalReadyAnalyzerPods{TotalReadyAnalyzerPods: num}
}
func dbPodsReady(num int32) *storage.ScannerHealthInfo_TotalReadyDbPods {
	return &storage.ScannerHealthInfo_TotalReadyDbPods{TotalReadyDbPods: num}
}
