package datastore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	commentsstoreMocks "github.com/stackrox/rox/central/alert/datastore/internal/commentsstore/mocks"
	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	"github.com/stackrox/rox/central/alert/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/alert/mappings"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	rocksdbPkg "github.com/stackrox/rox/pkg/rocksdb"
	rocksdbMetrics "github.com/stackrox/rox/pkg/rocksdb/metrics"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	cluster1   = "cluster1"
	cluster2   = "cluster2"
	namespaceA = "namespaceA"
	namespaceB = "namespaceB"
	namespaceC = "namespaceC"
)

var (
	alertUnrestrictedReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))
	alertUnrestrictedReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))
	alertCluster1ReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster1)))
	alertCluster1NamespaceAReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster1),
			sac.NamespaceScopeKeys(namespaceA)))
	alertCluster1NamespaceBReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster1),
			sac.NamespaceScopeKeys(namespaceB)))
	alertCluster1NamespaceCReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster1),
			sac.NamespaceScopeKeys(namespaceC)))
	alertCluster1NamespacesABReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster1),
			sac.NamespaceScopeKeys(namespaceA, namespaceB)))
	alertCluster1NamespacesACReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster1),
			sac.NamespaceScopeKeys(namespaceA, namespaceC)))
	alertCluster1NamespacesBCReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster1),
			sac.NamespaceScopeKeys(namespaceB, namespaceC)))
	alertCluster2ReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster2)))
	alertCluster2NamespaceAReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster2),
			sac.NamespaceScopeKeys(namespaceA)))
	alertCluster2NamespaceBReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster2),
			sac.NamespaceScopeKeys(namespaceB)))
	alertCluster2NamespaceCReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster2),
			sac.NamespaceScopeKeys(namespaceC)))
	alertCluster2NamespacesABReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster2),
			sac.NamespaceScopeKeys(namespaceA, namespaceB)))
	alertCluster2NamespacesACReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster2),
			sac.NamespaceScopeKeys(namespaceA, namespaceC)))
	alertCluster2NamespacesBCReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster2),
			sac.NamespaceScopeKeys(namespaceB, namespaceC)))
	alertMixedClusterNamespaceReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.OneStepSCC{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS): sac.OneStepSCC{
				sac.ResourceScopeKey(resources.Alert.Resource): sac.OneStepSCC{
					sac.ClusterScopeKey(cluster1): sac.AllowFixedScopes(sac.NamespaceScopeKeys(namespaceA)),
					sac.ClusterScopeKey(cluster2): sac.AllowFixedScopes(sac.NamespaceScopeKeys(namespaceB)),
				},
			},
		})
)

func createTestAlert(alertID string, clusterID string, namespace string) *storage.Alert {
	alert := storage.Alert{
		Id:             alertID,
		LifecycleStage: storage.LifecycleStage_DEPLOY,
		State:          storage.ViolationState_ATTEMPTED,
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Id:          strings.Join([]string{clusterID, namespace}, "::"),
				Name:        strings.Join([]string{clusterID, namespace}, "::"),
				Type:        "TestDeployment",
				ClusterId:   clusterID,
				ClusterName: clusterID,
				Namespace:   namespace,
				NamespaceId: namespace,
			},
		},
	}
	return &alert
}

func TestAlertDatastoreSAC(t *testing.T) {
	suite.Run(t, new(alertDatastoreSACTestSuite))
}

type alertDatastoreSACTestSuite struct {
	suite.Suite

	dbdir string

	comments  *commentsstoreMocks.MockStore
	storage   store.Store
	indexer   index.Indexer
	search    search.Searcher
	datastore DataStore

	mockCtrl *gomock.Controller
}

func (s *alertDatastoreSACTestSuite) SetupSuite() {
	var err error
	// Begin test DB framework creation
	alertObj := "alert"
	s.dbdir, err = os.MkdirTemp("", "alerttests")
	s.NoError(err)
	db, dberr := rocksdbPkg.New(rocksdbMetrics.GetRocksDBPath(s.dbdir))
	s.NoError(dberr)
	indexPath := filepath.Join(s.dbdir, "index", alertObj)
	alertIndex, alertIndexErr := globalindex.InitializeIndices(
		alertObj, indexPath, globalindex.EphemeralIndex, v1.SearchCategory_ALERTS.String())
	s.NoError(alertIndexErr)
	// End test DB framework creation
	s.mockCtrl = gomock.NewController(s.T())
	s.comments = commentsstoreMocks.NewMockStore(s.mockCtrl)
	s.storage = rocksdb.NewFullStore(db)
	s.indexer = index.New(alertIndex)
	s.search = search.New(s.storage, s.indexer)
	s.datastore, err = New(s.storage, s.comments, s.indexer, s.search)
	s.Require().NoError(err)
}

func (s *alertDatastoreSACTestSuite) TearDownSuite() {
	err := os.RemoveAll(s.dbdir)
	s.NoError(err)
}

func (s *alertDatastoreSACTestSuite) SetupTest() {
}

func (s *alertDatastoreSACTestSuite) TestUpsertDeleteAlertAllowed() {
	ctx := alertUnrestrictedReadWriteCtx
	testAlertID := "TestUpsertDeleteAllowed"
	s.comments.EXPECT().RemoveAlertComments(testAlertID).Return(nil)
	updateErr := s.datastore.UpsertAlert(ctx, createTestAlert(testAlertID, cluster1, namespaceA))
	deleterErr := s.datastore.DeleteAlerts(ctx, testAlertID)
	s.Require().NoError(updateErr)
	s.Require().NoError(deleterErr)
}

func (s *alertDatastoreSACTestSuite) TestUpsertDeleteAlertDenied() {
	ctx := alertCluster2ReadWriteCtx
	testAlertID := "TestUpsertDeleteDenied"
	// Here, the context does not allow cluster1. Operations should be denied
	updateErr := s.datastore.UpsertAlert(ctx, createTestAlert(testAlertID, cluster1, namespaceA))
	deleteErr := s.datastore.DeleteAlerts(ctx, testAlertID)
	s.Equal(sac.ErrResourceAccessDenied, updateErr, "alert update should have been denied")
	s.Equal(sac.ErrResourceAccessDenied, deleteErr, "alert removal should have been denied")
}

func (s *alertDatastoreSACTestSuite) TestUnrestrictedSearch() {
	ctx := alertUnrestrictedReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.Equal(18, len(searchResults))
	s.NoError(err)
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl1() {
	ctx := alertCluster1ReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(13, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(2, len(resultCounts[cluster1]))
	s.Equal(8, resultCounts[cluster1][namespaceA])
	s.Equal(5, resultCounts[cluster1][namespaceB])
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl1NsA() {
	ctx := alertCluster1NamespaceAReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(8, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(1, len(resultCounts[cluster1]))
	s.Equal(8, resultCounts[cluster1][namespaceA])
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl1NsB() {
	ctx := alertCluster1NamespaceBReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(5, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(1, len(resultCounts[cluster1]))
	s.Equal(5, resultCounts[cluster1][namespaceB])
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl1NsC() {
	ctx := alertCluster1NamespaceCReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(0, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(0, len(resultCounts))
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl1NsAB() {
	ctx := alertCluster1NamespacesABReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(13, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(2, len(resultCounts[cluster1]))
	s.Equal(8, resultCounts[cluster1][namespaceA])
	s.Equal(5, resultCounts[cluster1][namespaceB])
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl1NsAC() {
	ctx := alertCluster1NamespacesACReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(8, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(1, len(resultCounts[cluster1]))
	s.Equal(8, resultCounts[cluster1][namespaceA])
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl1NsBC() {
	ctx := alertCluster1NamespacesBCReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(5, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(1, len(resultCounts[cluster1]))
	s.Equal(5, resultCounts[cluster1][namespaceB])
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl2() {
	ctx := alertCluster2ReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(5, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(2, len(resultCounts[cluster2]))
	s.Equal(3, resultCounts[cluster2][namespaceB])
	s.Equal(2, resultCounts[cluster2][namespaceC])
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl2NsA() {
	ctx := alertCluster2NamespaceAReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(0, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(0, len(resultCounts))
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl2NsB() {
	ctx := alertCluster2NamespaceBReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(3, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(1, len(resultCounts[cluster2]))
	s.Equal(3, resultCounts[cluster2][namespaceB])
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl2NsC() {
	ctx := alertCluster2NamespaceCReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(2, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(1, len(resultCounts[cluster2]))
	s.Equal(2, resultCounts[cluster2][namespaceC])
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl2NsAB() {
	ctx := alertCluster2NamespacesABReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(3, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(1, len(resultCounts[cluster2]))
	s.Equal(3, resultCounts[cluster2][namespaceB])
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl2NsAC() {
	ctx := alertCluster2NamespacesACReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(2, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(1, len(resultCounts[cluster2]))
	s.Equal(2, resultCounts[cluster2][namespaceC])
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchCl2NsBC() {
	ctx := alertCluster2NamespacesBCReadWriteCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	s.Equal(5, len(searchResults))
	resultCounts := countResultsPerClusterAndNamespace(searchResults)
	s.Equal(1, len(resultCounts))
	s.Equal(2, len(resultCounts[cluster2]))
	s.Equal(3, resultCounts[cluster2][namespaceB])
	s.Equal(2, resultCounts[cluster2][namespaceC])
}

func countResultsPerClusterAndNamespace(results []searchPkg.Result) map[string]map[string]int {
	resultCounts := make(map[string]map[string]int, 0)
	clusterIDField, _ := mappings.OptionsMap.Get(searchPkg.ClusterID.String())
	namespaceField, _ := mappings.OptionsMap.Get(searchPkg.Namespace.String())
	for _, result := range results {
		var clusterID string
		var namespace string
		for k, v := range result.Fields {
			if k == clusterIDField.GetFieldPath() {
				clusterID = fmt.Sprintf("%v", v)
			}
			if k == namespaceField.GetFieldPath() {
				namespace = fmt.Sprintf("%v", v)
			}
		}
		if _, clusterExists := resultCounts[clusterID]; !clusterExists {
			resultCounts[clusterID] = make(map[string]int, 0)
		}
		if _, namespaceExists := resultCounts[clusterID][namespace]; !namespaceExists {
			resultCounts[clusterID][namespace] = 0
		}
		resultCounts[clusterID][namespace]++
	}
	return resultCounts
}

func (s *alertDatastoreSACTestSuite) cleanAlerts(alertIDs []string) {
	err := s.datastore.DeleteAlerts(alertUnrestrictedReadWriteCtx, alertIDs...)
	s.NoError(err)
}

func (s *alertDatastoreSACTestSuite) injectAlert(clusterID string, namespace string) string {
	alert := createTestAlert(uuid.NewV4().String(), clusterID, namespace)
	s.comments.EXPECT().RemoveAlertComments(alert.Id).Return(nil)
	upsertErr := s.datastore.UpsertAlert(alertUnrestrictedReadWriteCtx, alert)
	s.NoError(upsertErr, "test preparation failed on upsert alert %s", alert.Id)
	return alert.Id
}

func (s *alertDatastoreSACTestSuite) injectTestDataset() []string {
	alertIDs := make([]string, 0, 18)
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceA))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceA))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceA))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceA))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceA))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceA))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceA))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceA))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceB))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceB))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceB))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceB))
	alertIDs = append(alertIDs, s.injectAlert(cluster1, namespaceB))
	alertIDs = append(alertIDs, s.injectAlert(cluster2, namespaceB))
	alertIDs = append(alertIDs, s.injectAlert(cluster2, namespaceB))
	alertIDs = append(alertIDs, s.injectAlert(cluster2, namespaceB))
	alertIDs = append(alertIDs, s.injectAlert(cluster2, namespaceC))
	alertIDs = append(alertIDs, s.injectAlert(cluster2, namespaceC))
	return alertIDs
}
