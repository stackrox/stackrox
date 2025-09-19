package connection

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/common/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_GetConnectionPreference(t *testing.T) {
	wrappedErr := errors.Wrap(status.Error(codes.ResourceExhausted, "gRPC exhausted"), "recv error")
	errToPreferenece := map[error]bool{
		wrappedErr: false,
		status.Error(codes.ResourceExhausted, "gRPC exhausted"): false,
		status.Error(codes.Canceled, "gRPC canceled"):           true,
		status.Error(codes.Internal, "gRPC internal"):           true,
		nil:                      true,
		errors.New("custom err"): true,
	}
	for err, pref := range errToPreferenece {
		var testName string
		if err == nil {
			testName = "nil"
		} else {
			testName = err.Error()
		}
		t.Run(testName, func(t *testing.T) {
			m := manager{}
			m.handleConnectionError("1234", err)
			assert.Equal(t, pref, m.GetConnectionPreference("1234").SendDeduperState)
		})
	}
}

func Test_GetConnectionPreference_DefaultsToTrue(t *testing.T) {
	m := manager{}
	assert.Equal(t, true, m.GetConnectionPreference("1234").SendDeduperState)
}

func Test_Manager_OnlyUpdatesInactiveHealthForLegacySensors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClusterManager := mocks.NewMockClusterManager(ctrl)

	// Create clusters with old timestamps to simulate the original bug scenario
	oldTimestamp := protocompat.TimestampNow()
	oldTimestamp.Seconds -= 3600 // 1 hour ago

	modernCluster := &storage.Cluster{
		Id:   "modern-cluster",
		Name: "test-modern-cluster",
		HealthStatus: &storage.ClusterHealthStatus{
			Id:                 "modern-cluster",
			LastContact:        oldTimestamp, // Old timestamp - should NOT be overwritten
			HealthInfoComplete: true,         // Modern sensor with HealthMonitoringCap
		},
	}

	legacyCluster := &storage.Cluster{
		Id:   "legacy-cluster",
		Name: "test-legacy-cluster",
		HealthStatus: &storage.ClusterHealthStatus{
			Id:                 "legacy-cluster",
			LastContact:        oldTimestamp, // Old timestamp - should be preserved in updateInactiveClusterHealth
			HealthInfoComplete: false,        // Legacy sensor without HealthMonitoringCap
		},
	}

	// Create manager instance
	m := &manager{
		connectionsByClusterID: make(map[string]connectionAndUpgradeController),
		clusters:               mockClusterManager,
	}

	// Set up mock expectations for Start() method - it initializes upgrade controllers
	mockClusterManager.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{
		modernCluster,
		legacyCluster,
	}, nil).AnyTimes()

	// Start() also calls GetCluster for each cluster to initialize upgrade controllers
	mockClusterManager.EXPECT().GetCluster(gomock.Any(), "modern-cluster").Return(modernCluster, true, nil).AnyTimes()
	mockClusterManager.EXPECT().GetCluster(gomock.Any(), "legacy-cluster").Return(legacyCluster, true, nil).AnyTimes()

	// Verify that ONLY legacy cluster gets UpdateClusterHealth called
	// The key test: modern cluster should NOT get its health updated due to HealthInfoComplete=true
	mockClusterManager.EXPECT().UpdateClusterHealth(gomock.Any(), "legacy-cluster", gomock.Any()).Return(nil).MinTimes(1)
	mockClusterManager.EXPECT().UpdateClusterHealth(gomock.Any(), "modern-cluster", gomock.Any()).Times(0)

	// Create a test ticker channel to control when health checks happen
	testTicker := make(chan time.Time, 1)
	// Trigger one health check cycle
	testTicker <- time.Now()
	close(testTicker)
	// Start the health check loop with our test ticker
	m.updateClusterHealthForever(testTicker, func() {})

	// Mock expectations are automatically verified when test ends:
	// - Legacy cluster (HealthInfoComplete=false) should get UpdateClusterHealth called
	// - Modern cluster (HealthInfoComplete=true) should NOT get UpdateClusterHealth called
}

func TestUpdateClusterHealthForever_ModernSensorWithCorruptedHealthData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClusterManager := mocks.NewMockClusterManager(ctrl)

	// Create a modern cluster with corrupted health data (ancient timestamp from 2021)
	// This simulates data that was corrupted by the bug in versions before the fix
	ancientTimestamp := protocompat.TimestampNow()
	ancientTimestamp.Seconds = 1617138742 // March 30, 2021 - matches the failing CI test timestamp

	modernClusterWithCorruptedData := &storage.Cluster{
		Id:   "modern-cluster-corrupted",
		Name: "modern-cluster-corrupted",
		HealthStatus: &storage.ClusterHealthStatus{
			SensorHealthStatus:    storage.ClusterHealthStatus_UNHEALTHY,
			CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
			LastContact:           ancientTimestamp,
			HealthInfoComplete:    true, // Modern sensor with HealthMonitoringCap
		},
	}

	m := &manager{
		connectionsByClusterID:      make(map[string]connectionAndUpgradeController),
		connectionsByClusterIDMutex: sync.RWMutex{},
		clusters:                    mockClusterManager,
	}

	// For this test, we simulate NO active connection but corrupted health data
	// This demonstrates the fixed behavior: corrupted data gets updated with proper status

	// Set up mock expectations
	mockClusterManager.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{
		modernClusterWithCorruptedData,
	}, nil).AnyTimes()

	mockClusterManager.EXPECT().GetCluster(gomock.Any(), "modern-cluster-corrupted").Return(modernClusterWithCorruptedData, true, nil).AnyTimes()

	// The key test: modern cluster with corrupted data SHOULD get UpdateClusterHealth called
	// to fix the corrupted ancient timestamp with appropriate disconnected status
	mockClusterManager.EXPECT().UpdateClusterHealth(gomock.Any(), "modern-cluster-corrupted", gomock.Any()).Times(1)

	// Create a test ticker channel to control when health checks happen
	testTicker := make(chan time.Time, 1)
	testTicker <- time.Now()
	close(testTicker)

	// Start the health check loop with our test ticker - this should detect and fix corrupted data
	m.updateClusterHealthForever(testTicker, func() {})
}
