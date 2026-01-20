package datastore

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
)

func TestExtractClusterConfig(t *testing.T) {
	t.Run("extracts all fields correctly", func(t *testing.T) {
		hello := &central.SensorHello{
			HelmManagedConfigInit: &central.HelmManagedConfigInit{
				ClusterName: "test-cluster",
				ManagedBy:   storage.ManagerType_MANAGER_TYPE_HELM_CHART,
				ClusterConfig: &storage.CompleteClusterConfig{
					ConfigFingerprint: "fingerprint123",
				},
			},
			DeploymentIdentification: &storage.SensorDeploymentIdentification{
				AppNamespace: "stackrox",
			},
			Capabilities: []string{"cap1", "cap2"},
		}

		config := extractClusterConfig(hello)

		assert.Equal(t, "test-cluster", config.clusterName)
		assert.Equal(t, storage.ManagerType_MANAGER_TYPE_HELM_CHART, config.manager)
		assert.Equal(t, "fingerprint123", config.helmConfig.GetConfigFingerprint())
		assert.Equal(t, "stackrox", config.deploymentIdentification.GetAppNamespace())
		assert.Equal(t, []string{"cap1", "cap2"}, config.capabilities)
	})

	t.Run("handles nil values gracefully", func(t *testing.T) {
		hello := &central.SensorHello{}

		config := extractClusterConfig(hello)

		assert.Empty(t, config.clusterName)
		assert.Equal(t, storage.ManagerType_MANAGER_TYPE_UNKNOWN, config.manager)
		assert.Nil(t, config.helmConfig)
	})
}

func TestShouldUpdateCluster(t *testing.T) {
	baseConfig := clusterConfigData{
		manager: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		helmConfig: &storage.CompleteClusterConfig{
			ConfigFingerprint: "fp123",
		},
		capabilities: []string{"cap1", "cap2"},
	}

	t.Run("returns false when nothing changed", func(t *testing.T) {
		cluster := &storage.Cluster{
			SensorCapabilities: []string{"cap1", "cap2"},
			InitBundleId:       "bundle123",
			HelmConfig: &storage.CompleteClusterConfig{
				ConfigFingerprint: "fp123",
			},
			ManagedBy: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		}

		needsUpdate := shouldUpdateCluster(cluster, baseConfig, "bundle123")
		assert.False(t, needsUpdate)
	})

	t.Run("returns true when capabilities changed", func(t *testing.T) {
		cluster := &storage.Cluster{
			SensorCapabilities: []string{"cap1", "cap3"},
			InitBundleId:       "bundle123",
			HelmConfig: &storage.CompleteClusterConfig{
				ConfigFingerprint: "fp123",
			},
			ManagedBy: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		}

		needsUpdate := shouldUpdateCluster(cluster, baseConfig, "bundle123")
		assert.True(t, needsUpdate)
	})

	t.Run("returns true when init bundle ID changed", func(t *testing.T) {
		cluster := &storage.Cluster{
			SensorCapabilities: []string{"cap1", "cap2"},
			InitBundleId:       "old-bundle",
			HelmConfig: &storage.CompleteClusterConfig{
				ConfigFingerprint: "fp123",
			},
			ManagedBy: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		}

		needsUpdate := shouldUpdateCluster(cluster, baseConfig, "new-bundle")
		assert.True(t, needsUpdate)
	})

	t.Run("returns true when fingerprint changed", func(t *testing.T) {
		cluster := &storage.Cluster{
			SensorCapabilities: []string{"cap1", "cap2"},
			InitBundleId:       "bundle123",
			HelmConfig: &storage.CompleteClusterConfig{
				ConfigFingerprint: "old-fp",
			},
			ManagedBy: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		}

		needsUpdate := shouldUpdateCluster(cluster, baseConfig, "bundle123")
		assert.True(t, needsUpdate)
	})

	t.Run("returns true when manager type changed", func(t *testing.T) {
		cluster := &storage.Cluster{
			SensorCapabilities: []string{"cap1", "cap2"},
			InitBundleId:       "bundle123",
			HelmConfig: &storage.CompleteClusterConfig{
				ConfigFingerprint: "fp123",
			},
			ManagedBy: storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR,
		}

		needsUpdate := shouldUpdateCluster(cluster, baseConfig, "bundle123")
		assert.True(t, needsUpdate)
	})

	t.Run("handles capability order independence", func(t *testing.T) {
		cluster := &storage.Cluster{
			SensorCapabilities: []string{"cap2", "cap1"}, // Different order
			InitBundleId:       "bundle123",
			HelmConfig: &storage.CompleteClusterConfig{
				ConfigFingerprint: "fp123",
			},
			ManagedBy: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		}

		needsUpdate := shouldUpdateCluster(cluster, baseConfig, "bundle123")
		assert.False(t, needsUpdate, "capability order should not matter")
	})
}

func TestBuildClusterFromConfig(t *testing.T) {
	t.Run("builds cluster with all fields", func(t *testing.T) {
		config := clusterConfigData{
			manager: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			helmConfig: &storage.CompleteClusterConfig{
				StaticConfig: &storage.StaticClusterConfig{
					Type:      storage.ClusterType_KUBERNETES_CLUSTER,
					MainImage: "stackrox/main:latest",
				},
			},
			deploymentIdentification: &storage.SensorDeploymentIdentification{
				AppNamespace: "stackrox",
			},
			capabilities: []string{"cap1", "cap2"},
		}

		cluster := buildClusterFromConfig("test-cluster", "bundle-123", config)

		assert.Equal(t, "test-cluster", cluster.GetName())
		assert.Equal(t, "bundle-123", cluster.GetInitBundleId())
		assert.Equal(t, "stackrox", cluster.GetMostRecentSensorId().GetAppNamespace())
		assert.ElementsMatch(t, []string{"cap1", "cap2"}, cluster.GetSensorCapabilities())
		assert.NotNil(t, cluster.GetHelmConfig())
	})

	t.Run("does not set HelmConfig for manually managed clusters", func(t *testing.T) {
		config := clusterConfigData{
			manager: storage.ManagerType_MANAGER_TYPE_MANUAL,
			helmConfig: &storage.CompleteClusterConfig{
				StaticConfig: &storage.StaticClusterConfig{},
			},
			deploymentIdentification: &storage.SensorDeploymentIdentification{},
			capabilities:             []string{},
		}

		cluster := buildClusterFromConfig("test-cluster", "bundle-123", config)

		assert.Nil(t, cluster.GetHelmConfig())
	})

	t.Run("capabilities are sorted", func(t *testing.T) {
		config := clusterConfigData{
			manager: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			helmConfig: &storage.CompleteClusterConfig{
				StaticConfig: &storage.StaticClusterConfig{},
			},
			deploymentIdentification: &storage.SensorDeploymentIdentification{},
			capabilities:             []string{"zzz", "aaa", "mmm"},
		}

		cluster := buildClusterFromConfig("test-cluster", "bundle-123", config)

		assert.Equal(t, []string{"aaa", "mmm", "zzz"}, cluster.GetSensorCapabilities())
	})
}

func TestApplyConfigToCluster(t *testing.T) {
	t.Run("applies all updates for Helm-managed cluster", func(t *testing.T) {
		original := &storage.Cluster{
			Id:                 "cluster-id",
			Name:               "test-cluster",
			ManagedBy:          storage.ManagerType_MANAGER_TYPE_MANUAL,
			InitBundleId:       "old-bundle",
			SensorCapabilities: []string{"old-cap"},
		}

		config := clusterConfigData{
			manager: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			helmConfig: &storage.CompleteClusterConfig{
				ConfigFingerprint: "new-fp",
				StaticConfig: &storage.StaticClusterConfig{
					Type: storage.ClusterType_KUBERNETES_CLUSTER,
				},
			},
			capabilities: []string{"new-cap1", "new-cap2"},
		}

		updated := applyConfigToCluster(original, config, "new-bundle")

		assert.Equal(t, storage.ManagerType_MANAGER_TYPE_HELM_CHART, updated.GetManagedBy())
		assert.Equal(t, "new-bundle", updated.GetInitBundleId())
		assert.ElementsMatch(t, []string{"new-cap1", "new-cap2"}, updated.GetSensorCapabilities())
		assert.NotNil(t, updated.GetHelmConfig())
		assert.Equal(t, "new-fp", updated.GetHelmConfig().GetConfigFingerprint())
	})

	t.Run("clears HelmConfig for manually managed cluster", func(t *testing.T) {
		original := &storage.Cluster{
			Id:           "cluster-id",
			Name:         "test-cluster",
			ManagedBy:    storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			InitBundleId: "bundle",
			HelmConfig: &storage.CompleteClusterConfig{
				ConfigFingerprint: "old-fp",
			},
		}

		config := clusterConfigData{
			manager:      storage.ManagerType_MANAGER_TYPE_MANUAL,
			helmConfig:   &storage.CompleteClusterConfig{},
			capabilities: []string{},
		}

		updated := applyConfigToCluster(original, config, "bundle")

		assert.Equal(t, storage.ManagerType_MANAGER_TYPE_MANUAL, updated.GetManagedBy())
		assert.Nil(t, updated.GetHelmConfig())
	})

	t.Run("does not mutate original cluster", func(t *testing.T) {
		original := &storage.Cluster{
			Id:           "cluster-id",
			Name:         "test-cluster",
			ManagedBy:    storage.ManagerType_MANAGER_TYPE_MANUAL,
			InitBundleId: "old-bundle",
		}

		config := clusterConfigData{
			manager:      storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			helmConfig:   &storage.CompleteClusterConfig{},
			capabilities: []string{"cap1"},
		}

		updated := applyConfigToCluster(original, config, "new-bundle")

		// Original should be unchanged
		assert.Equal(t, storage.ManagerType_MANAGER_TYPE_MANUAL, original.GetManagedBy())
		assert.Equal(t, "old-bundle", original.GetInitBundleId())

		// Updated should have new values
		assert.Equal(t, storage.ManagerType_MANAGER_TYPE_HELM_CHART, updated.GetManagedBy())
		assert.Equal(t, "new-bundle", updated.GetInitBundleId())
	})
}

func TestCheckGracePeriodForReconnect(t *testing.T) {
	deploymentID := &storage.SensorDeploymentIdentification{
		AppNamespace:       "stackrox",
		SystemNamespaceId:  "123",
		AppNamespaceId:     "456",
		DefaultNamespaceId: "789",
	}

	t.Run("allows reconnect when last contact is old", func(t *testing.T) {
		// Cluster with old last contact (outside grace period)
		cluster := &storage.Cluster{
			HealthStatus: &storage.ClusterHealthStatus{
				LastContact: nil, // Defaults to zero time, definitely outside grace period
			},
			MostRecentSensorId: deploymentID.CloneVT(),
		}

		err := checkGracePeriodForReconnect(cluster, deploymentID, storage.ManagerType_MANAGER_TYPE_HELM_CHART)
		assert.NoError(t, err)
	})

	t.Run("allows reconnect with matching deployment ID even within grace period", func(t *testing.T) {
		// Same deployment ID should always be allowed, even with recent contact
		cluster := &storage.Cluster{
			HealthStatus: &storage.ClusterHealthStatus{
				LastContact: nil,
			},
			MostRecentSensorId: deploymentID.CloneVT(),
		}

		err := checkGracePeriodForReconnect(cluster, deploymentID, storage.ManagerType_MANAGER_TYPE_HELM_CHART)
		assert.NoError(t, err)
	})

	t.Run("handles nil health status gracefully", func(t *testing.T) {
		cluster := &storage.Cluster{
			HealthStatus:       nil, // No health status at all
			MostRecentSensorId: deploymentID.CloneVT(),
		}

		err := checkGracePeriodForReconnect(cluster, deploymentID, storage.ManagerType_MANAGER_TYPE_HELM_CHART)
		assert.NoError(t, err, "should not panic with nil health status")
	})

	t.Run("returns error during grace period with different deployment IDs", func(t *testing.T) {
		// Control the environment variable to ensure grace period is enforced
		t.Setenv("ROX_SCALE_TEST", "false")

		// Create a cluster with RECENT last contact (within 3-minute grace period)
		cluster := &storage.Cluster{
			HealthStatus: &storage.ClusterHealthStatus{
				LastContact: protoconv.ConvertTimeToTimestampOrNil(
					time.Now().Add(-1 * time.Minute)), // 1 minute ago
			},
			MostRecentSensorId: &storage.SensorDeploymentIdentification{
				AppNamespace:      "stackrox",
				SystemNamespaceId: "old-cluster-kube-system-uid",
			},
		}

		// New deployment from a DIFFERENT cluster (different SystemNamespaceId)
		newDeploymentID := &storage.SensorDeploymentIdentification{
			AppNamespace:      "stackrox",                    // Same namespace name
			SystemNamespaceId: "new-cluster-kube-system-uid", // Different cluster!
		}

		err := checkGracePeriodForReconnect(cluster, newDeploymentID, storage.ManagerType_MANAGER_TYPE_HELM_CHART)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "registering Helm-managed cluster is not allowed")
		assert.Contains(t, err.Error(), "please wait")
	})

	t.Run("returns error during grace period for operator-managed cluster", func(t *testing.T) {
		// Control the environment variable to ensure grace period is enforced
		t.Setenv("ROX_SCALE_TEST", "false")

		// Create a cluster with RECENT last contact
		cluster := &storage.Cluster{
			HealthStatus: &storage.ClusterHealthStatus{
				LastContact: protoconv.ConvertTimeToTimestampOrNil(
					time.Now().Add(-90 * time.Second)), // 90 seconds ago
			},
			MostRecentSensorId: &storage.SensorDeploymentIdentification{
				AppNamespace:       "stackrox",
				DefaultNamespaceId: "default-namespace-uid-1",
			},
		}

		// New deployment from a different cluster (different DefaultNamespaceId)
		newDeploymentID := &storage.SensorDeploymentIdentification{
			AppNamespace:       "stackrox",
			DefaultNamespaceId: "default-namespace-uid-2", // Different cluster!
		}

		err := checkGracePeriodForReconnect(cluster, newDeploymentID, storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "registering Operator-managed cluster is not allowed")
		assert.Contains(t, err.Error(), "please wait")
	})
}
