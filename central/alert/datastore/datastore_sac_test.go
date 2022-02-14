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
	cluster3   = "cluster3"
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
	alertCluster3ReadWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert),
			sac.ClusterScopeKeys(cluster3)))
	alertMixedClusterNamespaceReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.OneStepSCC{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS): sac.OneStepSCC{
				sac.ResourceScopeKey(resources.Alert.Resource): sac.OneStepSCC{
					sac.ClusterScopeKey(cluster1): sac.AllowFixedScopes(sac.NamespaceScopeKeys(namespaceA)),
					sac.ClusterScopeKey(cluster2): sac.AllowFixedScopes(sac.NamespaceScopeKeys(namespaceB)),
					sac.ClusterScopeKey(cluster3): sac.AllowFixedScopes(sac.NamespaceScopeKeys(namespaceC)),
				},
			},
		})
)

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

type searchTestCase struct {
	ctx                 context.Context
	expectedResultCount int
	expectedResultMap   map[string]map[string]int
}

var (
	searchTestCases = map[string]searchTestCase{
		"Scope:[cluster1]": {
			ctx:                 alertCluster1ReadWriteCtx,
			expectedResultCount: 13,
			expectedResultMap: map[string]map[string]int{
				cluster1: {
					namespaceA: 8,
					namespaceB: 5,
				},
			},
		},
		"Scope:[cluster1::namespaceA]": {
			ctx:                 alertCluster1NamespaceAReadWriteCtx,
			expectedResultCount: 8,
			expectedResultMap: map[string]map[string]int{
				cluster1: {
					namespaceA: 8,
				},
			},
		},
		"Scope:[cluster1::namespaceB]": {
			ctx:                 alertCluster1NamespaceBReadWriteCtx,
			expectedResultCount: 5,
			expectedResultMap: map[string]map[string]int{
				cluster1: {
					namespaceB: 5,
				},
			},
		},
		"Scope:[cluster1::namespaceC]": {
			ctx:                 alertCluster1NamespaceCReadWriteCtx,
			expectedResultCount: 0,
			expectedResultMap:   map[string]map[string]int{},
		},
		"Scope:[cluster1::namespaceA,cluster1::namespaceB]": {
			ctx:                 alertCluster1NamespacesABReadWriteCtx,
			expectedResultCount: 13,
			expectedResultMap: map[string]map[string]int{
				cluster1: {
					namespaceA: 8,
					namespaceB: 5,
				},
			},
		},
		"Scope:[cluster1::namespaceA,cluster1::namespaceC]": {
			ctx:                 alertCluster1NamespacesACReadWriteCtx,
			expectedResultCount: 8,
			expectedResultMap: map[string]map[string]int{
				cluster1: {
					namespaceA: 8,
				},
			},
		},
		"Scope:[cluster1::namespaceB,cluster1::namespaceC]": {
			ctx:                 alertCluster1NamespacesBCReadWriteCtx,
			expectedResultCount: 5,
			expectedResultMap: map[string]map[string]int{
				cluster1: {
					namespaceB: 5,
				},
			},
		},
		"Scope:[cluster2]": {
			ctx:                 alertCluster2ReadWriteCtx,
			expectedResultCount: 5,
			expectedResultMap: map[string]map[string]int{
				cluster2: {
					namespaceB: 3,
					namespaceC: 2,
				},
			},
		},
		"Scope:[cluster2:namespaceA]": {
			ctx:                 alertCluster2NamespaceAReadWriteCtx,
			expectedResultCount: 0,
			expectedResultMap:   map[string]map[string]int{},
		},
		"Scope:[cluster2:namespaceB]": {
			ctx:                 alertCluster2NamespaceBReadWriteCtx,
			expectedResultCount: 3,
			expectedResultMap: map[string]map[string]int{
				cluster2: {
					namespaceB: 3,
				},
			},
		},
		"Scope:[cluster2:namespaceC]": {
			ctx:                 alertCluster2NamespaceCReadWriteCtx,
			expectedResultCount: 2,
			expectedResultMap: map[string]map[string]int{
				cluster2: {
					namespaceC: 2,
				},
			},
		},
		"Scope:[cluster2::namespaceA,cluster2::namespaceB]": {
			ctx:                 alertCluster2NamespacesABReadWriteCtx,
			expectedResultCount: 3,
			expectedResultMap: map[string]map[string]int{
				cluster2: {
					namespaceB: 3,
				},
			},
		},
		"Scope:[cluster2::namespaceA,cluster2::namespaceC]": {
			ctx:                 alertCluster2NamespacesACReadWriteCtx,
			expectedResultCount: 2,
			expectedResultMap: map[string]map[string]int{
				cluster2: {
					namespaceC: 2,
				},
			},
		},
		"Scope:[cluster2::namespaceB,cluster2::namespaceC]": {
			ctx:                 alertCluster2NamespacesBCReadWriteCtx,
			expectedResultCount: 5,
			expectedResultMap: map[string]map[string]int{
				cluster2: {
					namespaceB: 3,
					namespaceC: 2,
				},
			},
		},
		"Scope:[cluster3]": {
			ctx:                 alertCluster3ReadWriteCtx,
			expectedResultCount: 0,
			expectedResultMap:   map[string]map[string]int{},
		},
		"Scope:[cluster1::namespaceA,cluster2::namespaceB,cluster3::namespaceC]": {
			ctx:                 alertMixedClusterNamespaceReadCtx,
			expectedResultCount: 11,
			expectedResultMap: map[string]map[string]int{
				cluster1: {namespaceA: 8},
				cluster2: {namespaceB: 3},
			},
		},
	}
)

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
	ctx := alertUnrestrictedReadCtx
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	searchResults, err := s.datastore.Search(ctx, nil)
	s.Equal(18, len(searchResults))
	s.NoError(err)
}

func (s *alertDatastoreSACTestSuite) TestScopedSearch() {
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	for name, testcase := range searchTestCases {
		s.Run(name, func() {
			searchResult, err := s.datastore.Search(testcase.ctx, nil)
			s.NoError(err)
			s.Equal(testcase.expectedResultCount, len(searchResult))
			resultCounts := countResultsPerClusterAndNamespace(searchResult)
			// Check the cluster/namespace distribution of results is the expected one.
			s.Equal(len(testcase.expectedResultMap), len(resultCounts))
			if len(testcase.expectedResultMap) > 0 {
				for clusterID, subMap := range testcase.expectedResultMap {
					_, clusterFound := resultCounts[clusterID]
					s.True(clusterFound)
					if clusterFound {
						for namespace, count := range subMap {
							_, namespaceFound := resultCounts[clusterID][namespace]
							s.True(namespaceFound)
							s.Equal(count, resultCounts[clusterID][namespace])
						}
					}
				}
			}
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchAlerts() {
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	for name, testcase := range searchTestCases {
		s.Run(name, func() {
			searchResult, err := s.datastore.SearchAlerts(testcase.ctx, nil)
			s.NoError(err)
			s.Equal(testcase.expectedResultCount, len(searchResult))
			resultCounts := countSearchAlertsResultsPerClusterAndNamespace(searchResult)
			// Check the cluster/namespace distribution of results is the expected one.
			s.Equal(len(testcase.expectedResultMap), len(resultCounts))
			if len(testcase.expectedResultMap) > 0 {
				for clusterID, subMap := range testcase.expectedResultMap {
					_, clusterFound := resultCounts[clusterID]
					s.True(clusterFound)
					if clusterFound {
						for namespace, count := range subMap {
							_, namespaceFound := resultCounts[clusterID][namespace]
							s.True(namespaceFound)
							s.Equal(count, resultCounts[clusterID][namespace])
						}
					}
				}
			}
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchRawAlerts() {
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	for name, testcase := range searchTestCases {
		s.Run(name, func() {
			searchResult, err := s.datastore.SearchRawAlerts(testcase.ctx, nil)
			s.NoError(err)
			s.Equal(testcase.expectedResultCount, len(searchResult))
			resultCounts := countSearchRawAlertsResultsPerClusterAndNamespace(searchResult)
			// Check the cluster/namespace distribution of results is the expected one.
			s.Equal(len(testcase.expectedResultMap), len(resultCounts))
			if len(testcase.expectedResultMap) > 0 {
				for clusterID, subMap := range testcase.expectedResultMap {
					_, clusterFound := resultCounts[clusterID]
					s.True(clusterFound)
					if clusterFound {
						for namespace, count := range subMap {
							_, namespaceFound := resultCounts[clusterID][namespace]
							s.True(namespaceFound)
							s.Equal(count, resultCounts[clusterID][namespace])
						}
					}
				}
			}
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestScopedSearchListAlerts() {
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	for name, testcase := range searchTestCases {
		s.Run(name, func() {
			searchResult, err := s.datastore.SearchListAlerts(testcase.ctx, nil)
			s.NoError(err)
			s.Equal(testcase.expectedResultCount, len(searchResult))
			resultCounts := countListAlertsResultsPerClusterAndNamespace(searchResult)
			// Check the cluster/namespace distribution of results is the expected one.
			s.Equal(len(testcase.expectedResultMap), len(resultCounts))
			if len(testcase.expectedResultMap) > 0 {
				for clusterID, subMap := range testcase.expectedResultMap {
					_, clusterFound := resultCounts[clusterID]
					s.True(clusterFound)
					if clusterFound {
						for namespace, count := range subMap {
							_, namespaceFound := resultCounts[clusterID][namespace]
							s.True(namespaceFound)
							s.Equal(count, resultCounts[clusterID][namespace])
						}
					}
				}
			}
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestScopedListAlerts() {
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	for name, testcase := range searchTestCases {
		s.Run(name, func() {
			searchResult, err := s.datastore.ListAlerts(testcase.ctx, nil)
			s.NoError(err)
			s.Equal(testcase.expectedResultCount, len(searchResult))
			resultCounts := countListAlertsResultsPerClusterAndNamespace(searchResult)
			// Check the cluster/namespace distribution of results is the expected one.
			s.Equal(len(testcase.expectedResultMap), len(resultCounts))
			if len(testcase.expectedResultMap) > 0 {
				for clusterID, subMap := range testcase.expectedResultMap {
					_, clusterFound := resultCounts[clusterID]
					s.True(clusterFound)
					if clusterFound {
						for namespace, count := range subMap {
							_, namespaceFound := resultCounts[clusterID][namespace]
							s.True(namespaceFound)
							s.Equal(count, resultCounts[clusterID][namespace])
						}
					}
				}
			}
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestScopedCount() {
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	for name, testcase := range searchTestCases {
		s.Run(name, func() {
			searchResult, err := s.datastore.Count(testcase.ctx, nil)
			s.NoError(err)
			s.Equal(testcase.expectedResultCount, searchResult)
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestScopedCountAlerts() {
	alertIDs := s.injectTestDataset()
	defer s.cleanAlerts(alertIDs)
	for name, testcase := range searchTestCases {
		s.Run(name, func() {
			searchResult, err := s.datastore.CountAlerts(testcase.ctx)
			s.NoError(err)
			s.Equal(testcase.expectedResultCount, searchResult)
		})
	}
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

func countSearchAlertsResultsPerClusterAndNamespace(results []*v1.SearchResult) map[string]map[string]int {
	resultCounts := make(map[string]map[string]int, 0)
	clusterIDField, _ := mappings.OptionsMap.Get(searchPkg.ClusterID.String())
	namespaceField, _ := mappings.OptionsMap.Get(searchPkg.Namespace.String())
	for _, result := range results {
		var clusterID string
		var namespace string
		for k, v := range result.GetFieldToMatches() {
			if k == clusterIDField.GetFieldPath() {
				if v != nil && len(v.Values) > 0 {
					clusterID = v.Values[0]
				}
			}
			if k == namespaceField.GetFieldPath() {
				if v != nil && len(v.Values) > 0 {
					namespace = v.Values[0]
				}
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

func countSearchRawAlertsResultsPerClusterAndNamespace(results []*storage.Alert) map[string]map[string]int {
	resultCounts := make(map[string]map[string]int, 0)
	for _, result := range results {
		var clusterID string
		var namespace string
		switch entity := result.Entity.(type) {
		case *storage.Alert_Deployment_:
			if entity.Deployment != nil {
				clusterID = entity.Deployment.GetClusterId()
				namespace = entity.Deployment.GetNamespace()
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

func countListAlertsResultsPerClusterAndNamespace(results []*storage.ListAlert) map[string]map[string]int {
	resultCounts := make(map[string]map[string]int, 0)
	for _, result := range results {
		var clusterID string
		var namespace string
		switch entity := result.Entity.(type) {
		case *storage.ListAlert_Deployment:
			if entity.Deployment != nil {
				clusterID = entity.Deployment.GetClusterId()
				namespace = entity.Deployment.GetNamespace()
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

func createTestAlert(alertID string, clusterID string, namespace string) *storage.Alert {
	alert := storage.Alert{
		Id:             alertID,
		LifecycleStage: storage.LifecycleStage_DEPLOY,
		State:          storage.ViolationState_ACTIVE,
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
