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

func (s *clusterDataStoreTestSuite) TestPostRemoveCluster_allSuccess() {
	clusterID := fixtureconsts.Cluster1
	removedCluster := &storage.Cluster{
		Id: clusterID,
	}

	clusterIDSearchQuery := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, clusterID).ProtoQuery()
	matchClusterIDSearchQuery := protomock.GoMockMatcherEqualMessage(clusterIDSearchQuery)

	// Set up expectations for postRemoveCluster calls
	// 1. Close connection
	s.sensorConnectionMgr.EXPECT().CloseConnection(clusterID).Times(1)

	// 2. Remove image integrations
	imageIntegration1ID := uuid.NewTestUUID(1).String()
	imageIntegration2ID := uuid.NewTestUUID(2).String()
	s.imageIntegrationDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: imageIntegration1ID},
				{ID: imageIntegration2ID},
			},
			nil,
		)
	s.imageIntegrationDS.EXPECT().
		RemoveImageIntegration(gomock.Any(), imageIntegration1ID).
		Times(1).
		Return(nil)
	s.imageIntegrationDS.EXPECT().
		RemoveImageIntegration(gomock.Any(), imageIntegration2ID).
		Times(1).
		Return(nil)

	// 3. Delete cluster health
	s.clusterHealthStore.EXPECT().
		Delete(gomock.Any(), clusterID).
		Times(1).
		Return(nil)

	// 4. Remove from ranker (no mock needed, it's a real object)
	// s.clusterRanker.Remove(clusterID) - will be called

	// 5. Remove namespaces
	namespace1ID := fixtureconsts.Namespace1
	namespace2ID := fixtureconsts.Namespace2
	s.namespaceDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: namespace1ID},
				{ID: namespace2ID},
			},
			nil,
		)
	s.namespaceDS.EXPECT().
		RemoveNamespace(gomock.Any(), namespace1ID).
		Times(1).
		Return(nil)
	s.namespaceDS.EXPECT().
		RemoveNamespace(gomock.Any(), namespace2ID).
		Times(1).
		Return(nil)

	// 6. Remove deployments
	deployment1ID := uuid.NewTestUUID(3).String()
	deployment2ID := uuid.NewTestUUID(4).String()
	s.deploymentDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: deployment1ID},
				{ID: deployment2ID},
			},
			nil,
		)

	s.deploymentDS.EXPECT().
		RemoveDeployment(gomock.Any(), clusterID, deployment1ID).
		Times(1).
		Return(nil)
	s.deploymentDS.EXPECT().
		RemoveDeployment(gomock.Any(), clusterID, deployment2ID).
		Times(1).
		Return(nil)

	// For each deployment, get alerts and mark them stale
	alert1ID := uuid.NewTestUUID(5).String()
	alert1 := &storage.Alert{Id: alert1ID}
	matchAlert1 := protomock.GoMockMatcherEqualMessage(alert1)
	alert2ID := uuid.NewTestUUID(6).String()
	alert2 := &storage.Alert{Id: alert2ID}
	matchAlert2 := protomock.GoMockMatcherEqualMessage(alert2)
	deployment1AlertQuery := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddExactMatches(pkgSearch.DeploymentID, deployment1ID).ProtoQuery()
	matchDeployment1AlertQuery := protomock.GoMockMatcherEqualMessage(deployment1AlertQuery)
	s.alertDS.EXPECT().
		SearchRawAlerts(gomock.Any(), matchDeployment1AlertQuery, true).
		Times(1).
		Return(
			[]*storage.Alert{
				alert1,
			},
			nil,
		)
	s.alertDS.EXPECT().
		MarkAlertsResolvedBatch(gomock.Any(), alert1ID).
		Times(1).
		Return(
			[]*storage.Alert{
				alert1,
			},
			nil,
		)
	s.notifierProcessor.EXPECT().
		ProcessAlert(gomock.Any(), matchAlert1).
		Times(1)

	deployment2AlertQuery := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddExactMatches(pkgSearch.DeploymentID, deployment2ID).ProtoQuery()
	matchDeployment2AlertQuery := protomock.GoMockMatcherEqualMessage(deployment2AlertQuery)
	s.alertDS.EXPECT().
		SearchRawAlerts(gomock.Any(), matchDeployment2AlertQuery, true).
		Times(1).
		Return(
			[]*storage.Alert{
				alert2,
			},
			nil,
		)
	s.alertDS.EXPECT().
		MarkAlertsResolvedBatch(gomock.Any(), alert2ID).
		Times(1).
		Return(
			[]*storage.Alert{
				alert2,
			},
			nil,
		)
	s.notifierProcessor.EXPECT().
		ProcessAlert(gomock.Any(), matchAlert2).
		Times(1)

	// 7. Remove pods
	podID1 := uuid.NewTestUUID(7).String()
	s.podDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: podID1},
			},
			nil,
		)
	s.podDS.EXPECT().
		RemovePod(gomock.Any(), podID1).
		Times(1).
		Return(nil)

	// 8. Delete all nodes for cluster
	s.nodeDS.EXPECT().
		DeleteAllNodesForCluster(gomock.Any(), clusterID).
		Times(1).
		Return(nil)

	// 9. Delete external network entities for cluster
	s.networkEntityDS.EXPECT().
		DeleteExternalNetworkEntitiesForCluster(gomock.Any(), clusterID).
		Times(1).
		Return(nil)

	// 10. Remove flow store
	s.networkFlowClusterDS.EXPECT().
		RemoveFlowStore(gomock.Any(), clusterID).
		Times(1).
		Return(nil)

	// 11. Remove compliance resources (if feature enabled)
	if features.ComplianceEnhancements.Enabled() {
		s.compliancePruner.EXPECT().
			RemoveComplianceResourcesByCluster(gomock.Any(), clusterID).
			Times(1)
	}

	// 12. Process network baseline deletion
	s.networkBaselineMgr.EXPECT().
		ProcessPostClusterDelete([]string{deployment1ID, deployment2ID}).
		Times(1).
		Return(nil)

	// 13. Remove secrets
	secretID1 := uuid.NewTestUUID(8).String()
	listSecret1 := &storage.ListSecret{Id: secretID1}
	s.secretDS.EXPECT().
		SearchListSecrets(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]*storage.ListSecret{
				listSecret1,
			},
			nil,
		)
	s.secretDS.EXPECT().
		RemoveSecret(gomock.Any(), secretID1).
		Times(1).
		Return(nil)

	// 14. Remove service accounts
	serviceAccount1ID := uuid.NewTestUUID(9).String()
	serviceAccount2ID := uuid.NewTestUUID(10).String()
	s.serviceAccountDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: serviceAccount1ID},
				{ID: serviceAccount2ID},
			},
			nil,
		)
	s.serviceAccountDS.EXPECT().
		RemoveServiceAccount(gomock.Any(), serviceAccount1ID).
		Times(1).
		Return(nil)
	s.serviceAccountDS.EXPECT().
		RemoveServiceAccount(gomock.Any(), serviceAccount2ID).
		Times(1).
		Return(nil)

	// 15. Remove K8S roles
	k8sRole1ID := uuid.NewTestUUID(11).String()
	s.k8sRoleDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: k8sRole1ID},
			},
			nil,
		)
	s.k8sRoleDS.EXPECT().
		RemoveRole(gomock.Any(), k8sRole1ID).
		Times(1).
		Return(nil)

	// 16. Remove role bindings
	k8soleBinding1ID := uuid.NewTestUUID(12).String()
	s.k8sRoleBindingDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: k8soleBinding1ID},
			},
			nil,
		)
	s.k8sRoleBindingDS.EXPECT().
		RemoveRoleBinding(gomock.Any(), k8soleBinding1ID).
		Times(1).
		Return(nil)

	// 17. Delete cluster CVEs
	s.clusterCVEDS.EXPECT().
		DeleteClusterCVEsInternal(gomock.Any(), clusterID).
		Times(1).
		Return(nil)

	ctx := sac.WithAllAccess(s.T().Context())
	doneSignal := concurrency.NewSignal()
	s.datastore.postRemoveCluster(ctx, removedCluster, &doneSignal)

	doneSignal.Wait()
}

func (s *clusterDataStoreTestSuite) TestPostRemoveCluster_allErrors() {
	clusterID := fixtureconsts.Cluster1
	removedCluster := &storage.Cluster{
		Id: clusterID,
	}

	clusterIDSearchQuery := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, clusterID).ProtoQuery()
	matchClusterIDSearchQuery := protomock.GoMockMatcherEqualMessage(clusterIDSearchQuery)

	testError := errors.New("test error")

	// Set up expectations for postRemoveCluster calls
	// 1. Close connection
	s.sensorConnectionMgr.EXPECT().CloseConnection(clusterID).Times(1)

	// 2. Remove image integrations
	s.imageIntegrationDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(nil, testError)

	// 3. Delete cluster health
	s.clusterHealthStore.EXPECT().
		Delete(gomock.Any(), clusterID).
		Times(1).
		Return(testError)

	// 4. Remove from ranker (no mock needed, it's a real object)
	// s.clusterRanker.Remove(clusterID) - will be called

	// 5. Remove namespaces
	namespace1ID := fixtureconsts.Namespace1
	s.namespaceDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: namespace1ID},
			},
			testError,
		)
	s.namespaceDS.EXPECT().
		RemoveNamespace(gomock.Any(), namespace1ID).
		Times(1).
		Return(testError)

	// 6. Remove deployments
	deployment1ID := uuid.NewTestUUID(3).String()
	deployment2ID := uuid.NewTestUUID(4).String()
	s.deploymentDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: deployment1ID},
				{ID: deployment2ID},
			},
			testError,
		)

	s.deploymentDS.EXPECT().
		RemoveDeployment(gomock.Any(), clusterID, deployment1ID).
		Times(1).
		Return(testError)
	s.deploymentDS.EXPECT().
		RemoveDeployment(gomock.Any(), clusterID, deployment2ID).
		Times(1).
		Return(testError)

	// For each deployment, get alerts and mark them stale
	alert1ID := uuid.NewTestUUID(5).String()
	alert1 := &storage.Alert{Id: alert1ID}
	matchAlert1 := protomock.GoMockMatcherEqualMessage(alert1)
	deployment1AlertQuery := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddExactMatches(pkgSearch.DeploymentID, deployment1ID).ProtoQuery()
	matchDeployment1AlertQuery := protomock.GoMockMatcherEqualMessage(deployment1AlertQuery)
	s.alertDS.EXPECT().
		SearchRawAlerts(gomock.Any(), matchDeployment1AlertQuery, true).
		Times(1).
		Return(
			[]*storage.Alert{
				alert1,
			},
			nil,
		)
	s.alertDS.EXPECT().
		MarkAlertsResolvedBatch(gomock.Any(), alert1ID).
		Times(1).
		Return(
			[]*storage.Alert{
				alert1,
			},
			nil,
		)
	s.notifierProcessor.EXPECT().
		ProcessAlert(gomock.Any(), matchAlert1).
		Times(1)

	deployment2AlertQuery := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddExactMatches(pkgSearch.DeploymentID, deployment2ID).ProtoQuery()
	matchDeployment2AlertQuery := protomock.GoMockMatcherEqualMessage(deployment2AlertQuery)
	s.alertDS.EXPECT().
		SearchRawAlerts(gomock.Any(), matchDeployment2AlertQuery, true).
		Times(1).
		Return(nil, testError)

	// 7. Remove pods
	s.podDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(nil, testError)

	// 8. Delete all nodes for cluster
	s.nodeDS.EXPECT().
		DeleteAllNodesForCluster(gomock.Any(), clusterID).
		Times(1).
		Return(testError)

	// 9. Delete external network entities for cluster
	s.networkEntityDS.EXPECT().
		DeleteExternalNetworkEntitiesForCluster(gomock.Any(), clusterID).
		Times(1).
		Return(testError)

	// 10. Remove flow store
	s.networkFlowClusterDS.EXPECT().
		RemoveFlowStore(gomock.Any(), clusterID).
		Times(1).
		Return(testError)

	// 11. Remove compliance resources (if feature enabled)
	if features.ComplianceEnhancements.Enabled() {
		s.compliancePruner.EXPECT().
			RemoveComplianceResourcesByCluster(gomock.Any(), clusterID).
			Times(1)
	}

	// 12. Process network baseline deletion
	s.networkBaselineMgr.EXPECT().
		ProcessPostClusterDelete([]string{deployment1ID, deployment2ID}).
		Times(1).
		Return(testError)

	// 13. Remove secrets
	secretID1 := uuid.NewTestUUID(8).String()
	listSecret1 := &storage.ListSecret{Id: secretID1}
	s.secretDS.EXPECT().
		SearchListSecrets(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]*storage.ListSecret{
				listSecret1,
			},
			testError,
		)
	s.secretDS.EXPECT().
		RemoveSecret(gomock.Any(), secretID1).
		Times(1).
		Return(testError)

	// 14. Remove service accounts
	s.serviceAccountDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(nil, testError)

	// 15. Remove K8S roles
	s.k8sRoleDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(nil, testError)

	// 16. Remove role bindings
	s.k8sRoleBindingDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(nil, testError)

	// 17. Delete cluster CVEs
	s.clusterCVEDS.EXPECT().
		DeleteClusterCVEsInternal(gomock.Any(), clusterID).
		Times(1).
		Return(testError)

	ctx := sac.WithAllAccess(s.T().Context())
	doneSignal := concurrency.NewSignal()
	s.datastore.postRemoveCluster(ctx, removedCluster, &doneSignal)

	doneSignal.Wait()
}

func (s *clusterDataStoreTestSuite) TestPostRemoveCluster_searchSuccessRemovalErrors() {
	clusterID := fixtureconsts.Cluster1
	removedCluster := &storage.Cluster{
		Id: clusterID,
	}

	clusterIDSearchQuery := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, clusterID).ProtoQuery()
	matchClusterIDSearchQuery := protomock.GoMockMatcherEqualMessage(clusterIDSearchQuery)

	testError := errors.New("test error")

	// Set up expectations for postRemoveCluster calls
	// 1. Close connection
	s.sensorConnectionMgr.EXPECT().CloseConnection(clusterID).Times(1)

	// 2. Remove image integrations
	s.imageIntegrationDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(nil, testError)

	// 3. Delete cluster health
	s.clusterHealthStore.EXPECT().
		Delete(gomock.Any(), clusterID).
		Times(1).
		Return(testError)

	// 4. Remove from ranker (no mock needed, it's a real object)
	// s.clusterRanker.Remove(clusterID) - will be called

	// 5. Remove namespaces
	namespace1ID := fixtureconsts.Namespace1
	s.namespaceDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: namespace1ID},
			},
			testError,
		)
	s.namespaceDS.EXPECT().
		RemoveNamespace(gomock.Any(), namespace1ID).
		Times(1).
		Return(testError)

	// 6. Remove deployments
	deployment1ID := uuid.NewTestUUID(3).String()
	deployment2ID := uuid.NewTestUUID(4).String()
	s.deploymentDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: deployment1ID},
				{ID: deployment2ID},
			},
			testError,
		)

	s.deploymentDS.EXPECT().
		RemoveDeployment(gomock.Any(), clusterID, deployment1ID).
		Times(1).
		Return(testError)
	s.deploymentDS.EXPECT().
		RemoveDeployment(gomock.Any(), clusterID, deployment2ID).
		Times(1).
		Return(testError)

	// For each deployment, get alerts and mark them stale
	alert1ID := uuid.NewTestUUID(5).String()
	alert1 := &storage.Alert{Id: alert1ID}
	matchAlert1 := protomock.GoMockMatcherEqualMessage(alert1)
	deployment1AlertQuery := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddExactMatches(pkgSearch.DeploymentID, deployment1ID).ProtoQuery()
	matchDeployment1AlertQuery := protomock.GoMockMatcherEqualMessage(deployment1AlertQuery)
	s.alertDS.EXPECT().
		SearchRawAlerts(gomock.Any(), matchDeployment1AlertQuery, true).
		Times(1).
		Return(
			[]*storage.Alert{
				alert1,
			},
			nil,
		)
	s.alertDS.EXPECT().
		MarkAlertsResolvedBatch(gomock.Any(), alert1ID).
		Times(1).
		Return(
			[]*storage.Alert{
				alert1,
			},
			nil,
		)
	s.notifierProcessor.EXPECT().
		ProcessAlert(gomock.Any(), matchAlert1).
		Times(1)

	deployment2AlertQuery := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddExactMatches(pkgSearch.DeploymentID, deployment2ID).ProtoQuery()
	matchDeployment2AlertQuery := protomock.GoMockMatcherEqualMessage(deployment2AlertQuery)
	s.alertDS.EXPECT().
		SearchRawAlerts(gomock.Any(), matchDeployment2AlertQuery, true).
		Times(1).
		Return(nil, testError)

	// 7. Remove pods
	podID1 := uuid.NewTestUUID(7).String()
	s.podDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: podID1},
			},
			nil,
		)
	s.podDS.EXPECT().
		RemovePod(gomock.Any(), podID1).
		Times(1).
		Return(testError)

	// 8. Delete all nodes for cluster
	s.nodeDS.EXPECT().
		DeleteAllNodesForCluster(gomock.Any(), clusterID).
		Times(1).
		Return(testError)

	// 9. Delete external network entities for cluster
	s.networkEntityDS.EXPECT().
		DeleteExternalNetworkEntitiesForCluster(gomock.Any(), clusterID).
		Times(1).
		Return(testError)

	// 10. Remove flow store
	s.networkFlowClusterDS.EXPECT().
		RemoveFlowStore(gomock.Any(), clusterID).
		Times(1).
		Return(testError)

	// 11. Remove compliance resources (if feature enabled)
	if features.ComplianceEnhancements.Enabled() {
		s.compliancePruner.EXPECT().
			RemoveComplianceResourcesByCluster(gomock.Any(), clusterID).
			Times(1)
	}

	// 12. Process network baseline deletion
	s.networkBaselineMgr.EXPECT().
		ProcessPostClusterDelete([]string{deployment1ID, deployment2ID}).
		Times(1).
		Return(testError)

	// 13. Remove secrets
	secretID1 := uuid.NewTestUUID(8).String()
	listSecret1 := &storage.ListSecret{Id: secretID1}
	s.secretDS.EXPECT().
		SearchListSecrets(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]*storage.ListSecret{
				listSecret1,
			},
			testError,
		)
	s.secretDS.EXPECT().
		RemoveSecret(gomock.Any(), secretID1).
		Times(1).
		Return(testError)

	// 14. Remove service accounts
	serviceAccount1ID := uuid.NewTestUUID(9).String()
	serviceAccount2ID := uuid.NewTestUUID(10).String()
	s.serviceAccountDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: serviceAccount1ID},
				{ID: serviceAccount2ID},
			},
			nil,
		)
	s.serviceAccountDS.EXPECT().
		RemoveServiceAccount(gomock.Any(), serviceAccount1ID).
		Times(1).
		Return(testError)
	s.serviceAccountDS.EXPECT().
		RemoveServiceAccount(gomock.Any(), serviceAccount2ID).
		Times(1).
		Return(testError)

	// 15. Remove K8S roles
	k8sRole1ID := uuid.NewTestUUID(11).String()
	s.k8sRoleDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: k8sRole1ID},
			},
			nil,
		)
	s.k8sRoleDS.EXPECT().
		RemoveRole(gomock.Any(), k8sRole1ID).
		Times(1).
		Return(testError)

	// 16. Remove role bindings
	k8soleBinding1ID := uuid.NewTestUUID(12).String()
	s.k8sRoleBindingDS.EXPECT().
		Search(gomock.Any(), matchClusterIDSearchQuery).
		Times(1).
		Return(
			[]pkgSearch.Result{
				{ID: k8soleBinding1ID},
			},
			nil,
		)
	s.k8sRoleBindingDS.EXPECT().
		RemoveRoleBinding(gomock.Any(), k8soleBinding1ID).
		Times(1).
		Return(testError)

	// 17. Delete cluster CVEs
	s.clusterCVEDS.EXPECT().
		DeleteClusterCVEsInternal(gomock.Any(), clusterID).
		Times(1).
		Return(testError)

	ctx := sac.WithAllAccess(s.T().Context())
	doneSignal := concurrency.NewSignal()
	s.datastore.postRemoveCluster(ctx, removedCluster, &doneSignal)

	doneSignal.Wait()
}
