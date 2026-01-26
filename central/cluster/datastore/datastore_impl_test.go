package datastore

import (
	"errors"
	"testing"

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
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/protomock"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/simplecache"
	"github.com/stackrox/rox/pkg/uuid"
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
