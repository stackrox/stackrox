package datastore

import (
	"errors"
	"testing"
	"time"

	alertDataStoreMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	clusterStoreMocks "github.com/stackrox/rox/central/cluster/store/cluster/mocks"
	clusterHealthStoreMocks "github.com/stackrox/rox/central/cluster/store/clusterhealth/mocks"
	compliancePrunerMocks "github.com/stackrox/rox/central/complianceoperator/v2/pruner/mocks"
	clusterCVEDataStoreMocks "github.com/stackrox/rox/central/cve/cluster/datastore/mocks"
	deploymentDataStoreMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageIntegrationDataStoreMocks "github.com/stackrox/rox/central/imageintegration/datastore/mocks"
	namespaceDataStoreMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	networkBaselineManagerMocks "github.com/stackrox/rox/central/networkbaseline/manager/mocks"
	netEntityDataStoreMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	netFlowDataStoreMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	nodeDataStoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	podDataStoreMocks "github.com/stackrox/rox/central/pod/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	roleDataStoreMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	roleBindingDataStoreMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	secretDataStoreMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	serviceAccountDataStoreMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/protomock"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/simplecache"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestClusterDataStore(t *testing.T) {
	suite.Run(t, new(clusterDataStoreTestSuite))
}

type clusterDataStoreTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	sensorConnectionMgr *connectionMocks.MockManager

	clusterStore       *clusterStoreMocks.MockStore
	clusterHealthStore *clusterHealthStoreMocks.MockStore

	alertDS              *alertDataStoreMocks.MockDataStore
	clusterCVEDS         *clusterCVEDataStoreMocks.MockDataStore
	deploymentDS         *deploymentDataStoreMocks.MockDataStore
	imageIntegrationDS   *imageIntegrationDataStoreMocks.MockDataStore
	namespaceDS          *namespaceDataStoreMocks.MockDataStore
	networkEntityDS      *netEntityDataStoreMocks.MockEntityDataStore
	networkFlowClusterDS *netFlowDataStoreMocks.MockClusterDataStore
	nodeDS               *nodeDataStoreMocks.MockDataStore
	podDS                *podDataStoreMocks.MockDataStore
	k8sRoleDS            *roleDataStoreMocks.MockDataStore
	k8sRoleBindingDS     *roleBindingDataStoreMocks.MockDataStore
	secretDS             *secretDataStoreMocks.MockDataStore
	serviceAccountDS     *serviceAccountDataStoreMocks.MockDataStore

	networkBaselineMgr *networkBaselineManagerMocks.MockManager

	notifierProcessor *notifierMocks.MockProcessor

	compliancePruner *compliancePrunerMocks.MockPruner

	datastore *datastoreImpl
}

func (s *clusterDataStoreTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.sensorConnectionMgr = connectionMocks.NewMockManager(s.mockCtrl)

	s.clusterStore = clusterStoreMocks.NewMockStore(s.mockCtrl)
	s.clusterHealthStore = clusterHealthStoreMocks.NewMockStore(s.mockCtrl)

	s.alertDS = alertDataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.clusterCVEDS = clusterCVEDataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.deploymentDS = deploymentDataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.imageIntegrationDS = imageIntegrationDataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.namespaceDS = namespaceDataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.networkEntityDS = netEntityDataStoreMocks.NewMockEntityDataStore(s.mockCtrl)
	s.networkFlowClusterDS = netFlowDataStoreMocks.NewMockClusterDataStore(s.mockCtrl)
	s.nodeDS = nodeDataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.podDS = podDataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.k8sRoleDS = roleDataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.k8sRoleBindingDS = roleBindingDataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.secretDS = secretDataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.serviceAccountDS = serviceAccountDataStoreMocks.NewMockDataStore(s.mockCtrl)

	s.networkBaselineMgr = networkBaselineManagerMocks.NewMockManager(s.mockCtrl)

	s.notifierProcessor = notifierMocks.NewMockProcessor(s.mockCtrl)

	s.compliancePruner = compliancePrunerMocks.NewMockPruner(s.mockCtrl)

	// Create datastore instance with all mocks
	s.datastore = &datastoreImpl{
		clusterStorage:            s.clusterStore,
		clusterHealthStorage:      s.clusterHealthStore,
		clusterCVEDataStore:       s.clusterCVEDS,
		alertDataStore:            s.alertDS,
		imageIntegrationDataStore: s.imageIntegrationDS,
		namespaceDataStore:        s.namespaceDS,
		deploymentDataStore:       s.deploymentDS,
		nodeDataStore:             s.nodeDS,
		podDataStore:              s.podDS,
		secretsDataStore:          s.secretDS,
		netFlowsDataStore:         s.networkFlowClusterDS,
		netEntityDataStore:        s.networkEntityDS,
		serviceAccountDataStore:   s.serviceAccountDS,
		roleDataStore:             s.k8sRoleDS,
		roleBindingDataStore:      s.k8sRoleBindingDS,
		cm:                        s.sensorConnectionMgr,
		notifier:                  s.notifierProcessor,
		clusterRanker:             ranking.NewRanker(),
		networkBaselineMgr:        s.networkBaselineMgr,
		compliancePruner:          s.compliancePruner,
		idToNameCache:             simplecache.New(),
		nameToIDCache:             simplecache.New(),
	}
}

func (s *clusterDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

type lookupResult struct {
	ID     string
	status error
}

type lookupPattern struct {
	status  error
	results []lookupResult
}

func (s *clusterDataStoreTestSuite) TestPostRemoveCluster() {
	clusterID := fixtureconsts.Cluster1
	removedCluster := &storage.Cluster{
		Id: clusterID,
	}

	clusterIDSearchQuery := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, clusterID).ProtoQuery()
	matchClusterIDSearchQuery := protomock.GoMockMatcherEqualMessage(clusterIDSearchQuery)

	testError := errors.New("test error")

	for name, tc := range map[string]lookupPattern{
		"All Success": {
			status: nil,
			results: []lookupResult{
				{ID: uuid.NewTestUUID(1).String(), status: nil},
				{ID: uuid.NewTestUUID(2).String(), status: nil},
			},
		},
		"All Errors": {
			status: testError,
		},
		"Lookup Success, Removal Error": {
			status: nil,
			results: []lookupResult{
				{ID: uuid.NewTestUUID(3).String(), status: testError},
				{ID: uuid.NewTestUUID(4).String(), status: testError},
			},
		},
	} {
		s.Run(name, func() {
			var searchResults []pkgSearch.Result
			var resultIDs []string
			for _, result := range tc.results {
				searchResults = append(searchResults, pkgSearch.Result{ID: result.ID})
				resultIDs = append(resultIDs, result.ID)
			}

			// Set up expectations for postRemoveCluster calls
			// 1. Close connection
			s.sensorConnectionMgr.EXPECT().CloseConnection(clusterID).Times(1)

			// 2. Remove image integrations
			s.imageIntegrationDS.EXPECT().
				Search(gomock.Any(), matchClusterIDSearchQuery).
				Times(1).
				Return(searchResults, tc.status)
			for _, fetched := range tc.results {
				s.imageIntegrationDS.EXPECT().RemoveImageIntegration(gomock.Any(), fetched.ID).Times(1).Return(fetched.status)
			}

			// 3. Delete cluster health
			s.clusterHealthStore.EXPECT().
				Delete(gomock.Any(), clusterID).
				Times(1).
				Return(tc.status)

			// 4. Remove from ranker (no mock needed, it's a real object)
			// s.clusterRanker.Remove(clusterID) - will be called

			// 5. Remove namespaces
			s.namespaceDS.EXPECT().
				Search(gomock.Any(), matchClusterIDSearchQuery).
				Times(1).
				Return(searchResults, tc.status)
			for _, result := range tc.results {
				s.namespaceDS.EXPECT().
					RemoveNamespace(gomock.Any(), result.ID).
					Times(1).
					Return(result.status)
			}

			// 6. Remove deployments
			s.deploymentDS.EXPECT().
				Search(gomock.Any(), matchClusterIDSearchQuery).
				Times(1).
				Return(searchResults, tc.status)
			for _, result := range tc.results {
				s.deploymentDS.EXPECT().
					RemoveDeployment(gomock.Any(), clusterID, result.ID).
					Times(1).
					Return(result.status)
				// For each deployment, get alerts and mark them stale
				var alertIDs []string
				var resolvedAlerts []*storage.Alert
				for _, alertResult := range tc.results {
					alertID := uuid.NewV5FromNonUUIDs(result.ID, alertResult.ID).String()
					alertIDs = append(alertIDs, alertID)
					resolvedAlerts = append(resolvedAlerts, &storage.Alert{Id: alertID})
				}
				alertQuery := pkgSearch.NewQueryBuilder().
					AddExactMatches(pkgSearch.ViolationState, storage.ViolationState_ACTIVE.String()).
					AddExactMatches(pkgSearch.DeploymentID, result.ID).ProtoQuery()
				matchAlertQuery := protomock.GoMockMatcherEqualMessage(alertQuery)
				s.alertDS.EXPECT().
					SearchRawAlerts(gomock.Any(), matchAlertQuery, true).
					Times(1).
					Return(resolvedAlerts, tc.status)
				if tc.status == nil {
					s.alertDS.EXPECT().
						MarkAlertsResolvedBatch(gomock.Any(), alertIDs).
						Times(1).
						Return(resolvedAlerts, nil)
					for _, alertResult := range resolvedAlerts {
						s.notifierProcessor.EXPECT().
							ProcessAlert(gomock.Any(), alertResult).
							Times(1)
					}
				}
			}

			// 7. Remove pods
			s.podDS.EXPECT().
				Search(gomock.Any(), matchClusterIDSearchQuery).
				Times(1).
				Return(searchResults, tc.status)
			if tc.status == nil {
				for _, result := range tc.results {
					s.podDS.EXPECT().
						RemovePod(gomock.Any(), result.ID).
						Times(1).
						Return(result.status)
				}
			}

			// 8. Delete all nodes for cluster
			s.nodeDS.EXPECT().
				DeleteAllNodesForCluster(gomock.Any(), clusterID).
				Times(1).
				Return(tc.status)

			// 9. Delete external network entities for cluster
			s.networkEntityDS.EXPECT().
				DeleteExternalNetworkEntitiesForCluster(gomock.Any(), clusterID).
				Times(1).
				Return(tc.status)

			// 10. Remove flow store
			s.networkFlowClusterDS.EXPECT().
				RemoveFlowStore(gomock.Any(), clusterID).
				Times(1).
				Return(tc.status)

			// 11. Remove compliance resources (if feature enabled)
			if features.ComplianceEnhancements.Enabled() {
				s.compliancePruner.EXPECT().
					RemoveComplianceResourcesByCluster(gomock.Any(), clusterID).
					Times(1)
			}

			// 12. Process network baseline deletion
			if len(resultIDs) > 0 {
				s.networkBaselineMgr.EXPECT().
					ProcessPostClusterDelete(resultIDs).
					Times(1).
					Return(tc.status)
			} else {
				s.networkBaselineMgr.EXPECT().
					ProcessPostClusterDelete(gomock.Any()).
					Times(1).
					Return(tc.status)
			}

			// 13. Remove secrets
			var listSecrets []*storage.ListSecret
			for _, result := range tc.results {
				listSecrets = append(listSecrets, &storage.ListSecret{Id: result.ID})
				s.secretDS.EXPECT().
					RemoveSecret(gomock.Any(), result.ID).
					Times(1).
					Return(result.status)
			}
			s.secretDS.EXPECT().
				SearchListSecrets(gomock.Any(), matchClusterIDSearchQuery).
				Times(1).
				Return(listSecrets, tc.status)

			// 14. Remove service accounts
			s.serviceAccountDS.EXPECT().
				Search(gomock.Any(), matchClusterIDSearchQuery).
				Times(1).
				Return(searchResults, tc.status)
			if tc.status == nil {
				for _, result := range tc.results {
					s.serviceAccountDS.EXPECT().
						RemoveServiceAccount(gomock.Any(), result.ID).
						Times(1).
						Return(result.status)
				}
			}

			// 15. Remove K8S roles
			s.k8sRoleDS.EXPECT().
				Search(gomock.Any(), matchClusterIDSearchQuery).
				Times(1).
				Return(searchResults, tc.status)
			if tc.status == nil {
				for _, result := range tc.results {
					s.k8sRoleDS.EXPECT().
						RemoveRole(gomock.Any(), result.ID).
						Times(1).
						Return(result.status)
				}
			}

			// 16. Remove role bindings
			s.k8sRoleBindingDS.EXPECT().
				Search(gomock.Any(), matchClusterIDSearchQuery).
				Times(1).
				Return(searchResults, tc.status)
			if tc.status == nil {
				for _, result := range tc.results {
					s.k8sRoleBindingDS.EXPECT().
						RemoveRoleBinding(gomock.Any(), result.ID).
						Times(1).
						Return(result.status)
				}
			}

			// 17. Delete cluster CVEs
			s.clusterCVEDS.EXPECT().
				DeleteClusterCVEsInternal(gomock.Any(), clusterID).
				Times(1).
				Return(tc.status)

			ctx := sac.WithAllAccess(s.T().Context())
			doneSignal := concurrency.NewSignal()
			s.datastore.postRemoveCluster(ctx, removedCluster, &doneSignal)

			doneSignal.Wait()
		})
	}
}

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
		assert.True(t, config.isNotManagedManually)
	})

	t.Run("handles nil values gracefully", func(t *testing.T) {
		hello := &central.SensorHello{}

		config := extractClusterConfig(hello)

		assert.Empty(t, config.clusterName)
		assert.Equal(t, storage.ManagerType_MANAGER_TYPE_UNKNOWN, config.manager)
		assert.Nil(t, config.helmConfig)
		assert.False(t, config.isNotManagedManually)
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
			isNotManagedManually: true,
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
			isNotManagedManually:     false,
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
			isNotManagedManually:     true,
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
			isNotManagedManually: true,
			capabilities:         []string{"new-cap1", "new-cap2"},
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
			manager:              storage.ManagerType_MANAGER_TYPE_MANUAL,
			helmConfig:           &storage.CompleteClusterConfig{},
			isNotManagedManually: false,
			capabilities:         []string{},
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
			manager:              storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			helmConfig:           &storage.CompleteClusterConfig{},
			isNotManagedManually: true,
			capabilities:         []string{"cap1"},
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
				LastContact: protoconv.ConvertTimeToTimestampOrNil(
					time.Now().Add(-1 * time.Minute)), // 1 minute ago
			},
			MostRecentSensorId: deploymentID.CloneVT(),
		}

		err := checkGracePeriodForReconnect(cluster, deploymentID, storage.ManagerType_MANAGER_TYPE_HELM_CHART)
		assert.NoError(t, err)
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
