package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	rocksdbStore "github.com/stackrox/rox/central/alert/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/alert/mappings"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	cleanupCtxKey = testutils.UnrestrictedReadWriteCtx
)

func TestAlertDatastoreSAC(t *testing.T) {
	suite.Run(t, new(alertDatastoreSACTestSuite))
}

type alertDatastoreSACTestSuite struct {
	suite.Suite

	engine *rocksdb.RocksDB
	index  *bleve.Index

	pool *pgxpool.Pool

	storage    store.Store
	indexer    index.Indexer
	search     search.Searcher
	optionsMap searchPkg.OptionsMap
	datastore  DataStore

	mockCtrl *gomock.Controller

	testContexts map[string]context.Context

	testAlertIDs []string
}

func (s *alertDatastoreSACTestSuite) SetupSuite() {
	var err error
	alertObj := "alertSACTest"

	s.mockCtrl = gomock.NewController(s.T())
	if features.PostgresDatastore.Enabled() {
		ctx := context.Background()
		source := pgtest.GetConnectionString(s.T())
		config, err := pgxpool.ParseConfig(source)
		s.NoError(err)
		s.pool, err = pgxpool.ConnectConfig(context.Background(), config)
		s.NoError(err)
		pgStore.Destroy(ctx, s.pool)
		gormDB := pgtest.OpenGormDB(s.T(), source)
		defer pgtest.CloseGormDB(s.T(), gormDB)
		s.storage = pgStore.CreateTableAndNewStore(ctx, s.pool, gormDB)
		s.indexer = pgStore.NewIndexWrapper(s.pool)
		s.optionsMap = schema.AlertsSchema.OptionsMap
	} else {
		s.engine, err = rocksdb.NewTemp(alertObj)
		s.NoError(err)
		var bleveindex bleve.Index
		bleveindex, err = globalindex.TempInitializeIndices(alertObj)
		s.index = &bleveindex
		s.NoError(err)

		s.storage = rocksdbStore.New(s.engine)
		s.indexer = index.New(*s.index)
		s.optionsMap = mappings.OptionsMap
	}
	s.search = search.New(s.storage, s.indexer)

	s.datastore, err = New(s.storage, s.indexer, s.search)
	s.NoError(err)

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Alert)
}

func (s *alertDatastoreSACTestSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	} else {
		err := rocksdb.CloseAndRemove(s.engine)
		s.NoError(err)
	}
}

func (s *alertDatastoreSACTestSuite) SetupTest() {
	s.testAlertIDs = make([]string, 0)

	// Inject test data set for search tests
	alerts := fixtures.GetSACTestAlertSet()
	for _, alert := range alerts {
		err := s.datastore.UpsertAlert(s.testContexts[testutils.UnrestrictedReadWriteCtx], alert)
		s.NoError(err)
		s.testAlertIDs = append(s.testAlertIDs, alert.GetId())
	}
}

func (s *alertDatastoreSACTestSuite) TearDownTest() {
	err := s.datastore.DeleteAlerts(s.testContexts[cleanupCtxKey], s.testAlertIDs...)
	s.NoError(err)
	s.testAlertIDs = nil
}

type crudTest struct {
	scopeKey      string
	expectedError error
	expectError   bool
	expectedFound bool
}

func (s *alertDatastoreSACTestSuite) cleanupAlert(ID string) {
	_ = s.datastore.DeleteAlerts(s.testContexts[cleanupCtxKey], ID)
}

func (s *alertDatastoreSACTestSuite) TestUpsertAlert() {
	alert1 := fixtures.GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	alert2 := fixtures.GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	s.testAlertIDs = append(s.testAlertIDs, alert1.Id)
	s.testAlertIDs = append(s.testAlertIDs, alert2.Id)

	cases := map[string]crudTest{
		"(full) read-only cannot upsert": {
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"full read-write can upsert": {
			scopeKey:      testutils.UnrestrictedReadWriteCtx,
			expectError:   false,
			expectedError: nil,
		},
		"full read-write on wrong cluster cannot upsert": {
			scopeKey:      testutils.Cluster1ReadWriteCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and wrong namespace name cannot upsert": {
			scopeKey:      testutils.Cluster1NamespaceAReadWriteCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and matching namespace name cannot upsert": {
			scopeKey:      testutils.Cluster1NamespaceBReadWriteCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on right cluster but wrong namespaces cannot upsert": {
			scopeKey:      testutils.Cluster2NamespacesACReadWriteCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"full read-write on right cluster can upsert": {
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectError:   false,
			expectedError: nil,
		},
		"read-write on the right cluster and namespace can upsert": {
			scopeKey:      testutils.Cluster2NamespaceBReadWriteCtx,
			expectError:   false,
			expectedError: nil,
		},
		"read-write on the right cluster and at least the right namespace can upsert": {
			scopeKey:      testutils.Cluster2NamespacesABReadWriteCtx,
			expectError:   false,
			expectedError: nil,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.scopeKey]
			var err error
			err = s.datastore.UpsertAlert(ctx, alert1)
			defer s.cleanupAlert(alert1.Id)
			if !c.expectError {
				s.NoError(err)
			} else {
				s.Equal(c.expectedError, err)
			}
			err = s.datastore.UpsertAlert(ctx, alert2)
			defer s.cleanupAlert(alert2.Id)
			if !c.expectError {
				s.NoError(err)
			} else {
				s.Equal(c.expectedError, err)
			}
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestMarkAlertStale() {
	cases := map[string]crudTest{
		"(full) read-only cannot mark alert stale": {
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectError:   true,
			expectedFound: true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"full read-write can mark alert stale": {
			scopeKey:    testutils.UnrestrictedReadWriteCtx,
			expectError: false,
		},
		"full read-write on wrong cluster cannot mark alert stale": {
			scopeKey:      testutils.Cluster1ReadWriteCtx,
			expectError:   true,
			expectedFound: false,
		},
		"read-write on wrong cluster and wrong namespace name cannot mark alert stale": {
			scopeKey:      testutils.Cluster1NamespaceAReadWriteCtx,
			expectError:   true,
			expectedFound: false,
		},
		"read-write on wrong cluster and matching namespace name cannot mark alert stale": {
			scopeKey:      testutils.Cluster1NamespaceBReadWriteCtx,
			expectError:   true,
			expectedFound: false,
		},
		"read-write on right cluster but wrong namespaces cannot mark alert stale": {
			scopeKey:      testutils.Cluster2NamespacesACReadWriteCtx,
			expectError:   true,
			expectedFound: false,
		},
		"full read-write on right cluster can mark alert stale": {
			scopeKey:    testutils.Cluster2ReadWriteCtx,
			expectError: false,
		},
		"read-write on the right cluster and namespace can mark alert stale": {
			scopeKey:    testutils.Cluster2NamespaceBReadWriteCtx,
			expectError: false,
		},
		"read-write on the right cluster and at least the right namespace can mark alert stale": {
			scopeKey:    testutils.Cluster2NamespacesABReadWriteCtx,
			expectError: false,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			var err error
			alert1 := fixtures.GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			s.testAlertIDs = append(s.testAlertIDs, alert1.Id)
			err = s.datastore.UpsertAlert(s.testContexts[testutils.UnrestrictedReadWriteCtx], alert1)
			defer s.cleanupAlert(alert1.Id)
			s.NoError(err)
			alert2 := fixtures.GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			s.testAlertIDs = append(s.testAlertIDs, alert2.Id)
			err = s.datastore.UpsertAlert(s.testContexts[testutils.UnrestrictedReadWriteCtx], alert2)
			defer s.cleanupAlert(alert2.Id)
			s.NoError(err)

			ctx := s.testContexts[c.scopeKey]
			err = s.datastore.MarkAlertStale(ctx, alert1.GetId())
			if !c.expectError {
				s.NoError(err)
			} else if !c.expectedFound {
				s.Equal(fmt.Errorf("alert with id '%s' does not exist", alert1.GetId()), err)
			} else {
				s.Equal(c.expectedError, err)
			}
			err = s.datastore.MarkAlertStale(ctx, alert2.GetId())
			if !c.expectError {
				s.NoError(err)
			} else if !c.expectedFound {
				s.Equal(fmt.Errorf("alert with id '%s' does not exist", alert2.GetId()), err)
			} else {
				s.Equal(c.expectedError, err)
			}
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestGetAlert() {
	// Inject two scoped alerts to the storage
	// The test will validate, depending on the scope present in the operation context,
	// whether the data should be seen by the requester or not
	var err error
	alert1 := fixtures.GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err = s.datastore.UpsertAlert(s.testContexts[testutils.UnrestrictedReadWriteCtx], alert1)
	s.testAlertIDs = append(s.testAlertIDs, alert1.Id)
	s.NoError(err)
	alert2 := fixtures.GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err = s.datastore.UpsertAlert(s.testContexts[testutils.UnrestrictedReadWriteCtx], alert2)
	s.testAlertIDs = append(s.testAlertIDs, alert2.Id)
	s.NoError(err)

	cases := map[string]crudTest{
		"(full) read-only can read": {
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedFound: true,
		},
		"full read-write can read": {
			scopeKey:      testutils.UnrestrictedReadWriteCtx,
			expectedFound: true,
		},
		"full read-write on wrong cluster cannot read": {
			scopeKey:      testutils.Cluster1ReadWriteCtx,
			expectedFound: false,
		},
		"read-write on wrong cluster and wrong namespace name cannot read": {
			scopeKey:      testutils.Cluster1NamespaceAReadWriteCtx,
			expectedFound: false,
		},
		"read-write on wrong cluster and matching namespace name cannot read": {
			scopeKey:      testutils.Cluster1NamespaceBReadWriteCtx,
			expectedFound: false,
		},
		"read-write on right cluster but wrong namespaces cannot read": {
			scopeKey:      testutils.Cluster2NamespacesACReadWriteCtx,
			expectedFound: false,
		},
		"full read-write on right cluster can read": {
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedFound: true,
		},
		"read-write on the right cluster and namespace can read": {
			scopeKey:      testutils.Cluster2NamespaceBReadWriteCtx,
			expectedFound: true,
		},
		"read-write on the right cluster and at least the right namespace can read": {
			scopeKey:      testutils.Cluster2NamespacesABReadWriteCtx,
			expectedFound: true,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.scopeKey]
			readAlert1, found1, err1 := s.datastore.GetAlert(ctx, alert1.GetId())
			s.NoError(err1)
			if c.expectedFound {
				s.True(found1)
				s.Equal(*alert1, *readAlert1)
			} else {
				s.False(found1)
				s.Nil(readAlert1)
			}
			readAlert2, found2, err2 := s.datastore.GetAlert(ctx, alert2.GetId())
			s.NoError(err2)
			if c.expectedFound {
				s.True(found2)
				s.Equal(*alert2, *readAlert2)
			} else {
				s.False(found2)
				s.Nil(readAlert2)
			}
		})
	}
}

// Note: UpsertAlerts does not enforce Scoped access control checks, these are performed
// one level up in the caller code

// Note: DeleteAlerts has a slightly different scope management behaviour: only users with
// full access scope on the alert resource are allowed to delete alerts

func (s *alertDatastoreSACTestSuite) TestDeleteAlert() {

	cases := map[string]crudTest{
		"(full) read-only cannot delete": {
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"full read-write can delete": {
			scopeKey: testutils.UnrestrictedReadWriteCtx,
		},
		"full read-write on wrong cluster cannot delete": {
			scopeKey:      testutils.Cluster1ReadWriteCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and wrong namespace name cannot delete": {
			scopeKey:      testutils.Cluster1NamespaceAReadWriteCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and matching namespace name cannot delete": {
			scopeKey:      testutils.Cluster1NamespaceBReadWriteCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on right cluster but wrong namespaces cannot delete": {
			scopeKey:      testutils.Cluster2NamespacesACReadWriteCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"full read-write on right cluster cannot delete": {
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on the right cluster and namespace cannot delete": {
			scopeKey:      testutils.Cluster2NamespaceBReadWriteCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on the right cluster and at least the right namespace cannot delete": {
			scopeKey:      testutils.Cluster2NamespacesABReadWriteCtx,
			expectError:   true,
			expectedError: sac.ErrResourceAccessDenied,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			// Inject two scoped alerts to the storage
			// The test will validate, depending on the scope present in the operation context,
			// whether the data should be seen by the requester or not
			var err error
			alert1 := fixtures.GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			err = s.datastore.UpsertAlert(s.testContexts[testutils.UnrestrictedReadWriteCtx], alert1)
			s.testAlertIDs = append(s.testAlertIDs, alert1.Id)
			s.NoError(err)
			alert2 := fixtures.GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			err = s.datastore.UpsertAlert(s.testContexts[testutils.UnrestrictedReadWriteCtx], alert2)
			s.testAlertIDs = append(s.testAlertIDs, alert2.Id)
			s.NoError(err)
			ctx := s.testContexts[c.scopeKey]
			err1 := s.datastore.DeleteAlerts(ctx, alert1.GetId())
			if c.expectError {
				s.Error(err1)
				s.ErrorIs(c.expectedError, err1)
			} else {
				s.NoError(err1)
			}
			err2 := s.datastore.DeleteAlerts(ctx, alert1.GetId(), alert2.GetId())
			if c.expectError {
				s.Error(err2)
				s.ErrorIs(c.expectedError, err2)
			} else {
				s.NoError(err2)
			}
		})
	}
}

type alertSACSearchResult struct {
	scopeKey     string
	resultCounts map[string]map[string]int // Top level key is the cluster ID, then namespace
}

// The SAC alert test dataset defined in pkg/fixtures/alert.go has the following alert distribution
// Global: 1
// Cluster1::NamespaceA: 8 alerts
// Cluster1::NamespaceB: 5 alerts
// Cluster2::NamespaceB: 3 alerts
// Cluster2::NamespaceC: 2 alerts
var alertScopedSACSearchTestCases = map[string]alertSACSearchResult{
	"Cluster1 read-write access should only see Cluster1 alerts": {
		scopeKey: testutils.Cluster1ReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceA: 7,
				testconsts.NamespaceB: 5,
			},
		},
	},
	"Cluster1 and NamespaceA read-write access should only see Cluster1 and NamespaceA alerts": {
		scopeKey: testutils.Cluster1NamespaceAReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceA: 8,
			},
		},
	},
	"Cluster1 and NamespaceB read-write access should only see Cluster1 and NamespaceB alerts": {
		scopeKey: testutils.Cluster1NamespaceBReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceB: 5,
			},
		},
	},
	"Cluster1 and NamespaceC read-write access should only no alert": {
		scopeKey:     testutils.Cluster1NamespaceCReadWriteCtx,
		resultCounts: map[string]map[string]int{},
	},
	"Cluster1 and Namespaces A and B read-write access should only appropriate cluster/namespace alerts": {
		scopeKey: testutils.Cluster1NamespacesABReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceA: 8,
				testconsts.NamespaceB: 5,
			},
		},
	},
	"Cluster1 and Namespaces A and C read-write access should only appropriate cluster/namespace alerts": {
		scopeKey: testutils.Cluster1NamespacesACReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceA: 8,
			},
		},
	},
	"Cluster1 and Namespaces B and C read-write access should only appropriate cluster/namespace alerts": {
		scopeKey: testutils.Cluster1NamespacesBCReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceB: 5,
			},
		},
	},
	"Cluster2 read-write access should only see Cluster2 alerts": {
		scopeKey: testutils.Cluster2ReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceB: 3,
				testconsts.NamespaceC: 2,
			},
		},
	},
	"Cluster2 and NamespaceA read-write access should see no alert": {
		scopeKey:     testutils.Cluster2NamespaceAReadWriteCtx,
		resultCounts: map[string]map[string]int{},
	},
	"Cluster2 and NamespaceB read-write access should only see Cluster2 and NamespaceB alerts": {
		scopeKey: testutils.Cluster2NamespaceBReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceB: 3,
			},
		},
	},
	"Cluster2 and NamespaceC read-write access should only see Cluster2 and NamespaceC alert": {
		scopeKey: testutils.Cluster2NamespaceCReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceC: 2,
			},
		},
	},
	"Cluster2 and Namespaces A and B read-write access should only appropriate cluster/namespace alerts": {
		scopeKey: testutils.Cluster2NamespacesABReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceB: 3,
			},
		},
	},
	"Cluster2 and Namespaces A and C read-write access should only appropriate cluster/namespace alerts": {
		scopeKey: testutils.Cluster2NamespacesACReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceC: 2,
			},
		},
	},
	"Cluster2 and Namespaces B and C read-write access should only appropriate cluster/namespace alerts": {
		scopeKey: testutils.Cluster2NamespacesBCReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceB: 3,
				testconsts.NamespaceC: 2,
			},
		},
	},
	"Cluster3 read-write access should see no alert": {
		scopeKey:     testutils.Cluster3ReadWriteCtx,
		resultCounts: map[string]map[string]int{},
	},
}

var alertUnrestrictedSACSearchTestCases = map[string]alertSACSearchResult{
	"full read access should see all alerts": {
		// SAC search fields are not injected in query when running unscoped search
		// Therefore results cannot be dispatched per cluster and namespace
		scopeKey: testutils.UnrestrictedReadCtx,
		resultCounts: map[string]map[string]int{
			"": {"": 19},
		},
	},
	"full read-write access should see all alerts": {
		// SAC search fields are not injected in query when running unscoped search
		// Therefore results cannot be dispatched per cluster and namespace
		scopeKey: testutils.UnrestrictedReadWriteCtx,
		resultCounts: map[string]map[string]int{
			"": {"": 19},
		},
	},
}

var alertUnrestrictedSACObjectSearchTestCases = map[string]alertSACSearchResult{
	"full read access should see all alerts": {
		// SAC search fields are not injected in query when running unscoped search
		// Therefore results cannot be dispatched per cluster and namespace
		scopeKey: testutils.UnrestrictedReadCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceA: 8,
				testconsts.NamespaceB: 5,
			},
			testconsts.Cluster2: {
				testconsts.NamespaceB: 3,
				testconsts.NamespaceC: 2,
			},
			"": {"": 1},
		},
	},
	"full read-write access should see all alerts": {
		// SAC search fields are not injected in query when running unscoped search
		// Therefore results cannot be dispatched per cluster and namespace
		scopeKey: testutils.UnrestrictedReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceA: 8,
				testconsts.NamespaceB: 5,
			},
			testconsts.Cluster2: {
				testconsts.NamespaceB: 3,
				testconsts.NamespaceC: 2,
			},
			"": {"": 1},
		},
	},
}

func (s *alertDatastoreSACTestSuite) runSearchTest(testparams alertSACSearchResult) {
	ctx := s.testContexts[testparams.scopeKey]
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	resultCounts := testutils.CountResultsPerClusterAndNamespace(s.T(), searchResults, s.optionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testparams.resultCounts, resultCounts)
}

func (s *alertDatastoreSACTestSuite) TestAlertScopedSearch() {
	for name, c := range alertScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestAlertUnrestrictedSearch() {
	for name, c := range alertUnrestrictedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *alertDatastoreSACTestSuite) runCountTest(testparams alertSACSearchResult) {
	ctx := s.testContexts[testparams.scopeKey]
	resultCount, err := s.datastore.Count(ctx, nil)
	s.NoError(err)
	expectedResultCount := testutils.AggregateCounts(s.T(), testparams.resultCounts)
	s.Equal(expectedResultCount, resultCount)
}

func (s *alertDatastoreSACTestSuite) TestAlertScopedCount() {
	for name, c := range alertScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runCountTest(c)
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestAlertUnrestrictedCount() {
	for name, c := range alertUnrestrictedSACSearchTestCases {
		s.Run(name, func() {
			s.runCountTest(c)
		})
	}
}

func (s *alertDatastoreSACTestSuite) runCountAlertsTest(testparams alertSACSearchResult) {
	ctx := s.testContexts[testparams.scopeKey]
	resultCount, err := s.datastore.CountAlerts(ctx)
	s.NoError(err)
	expectedResultCount := testutils.AggregateCounts(s.T(), testparams.resultCounts)
	s.Equal(expectedResultCount, resultCount)
}

func (s *alertDatastoreSACTestSuite) TestAlertScopedCountAlerts() {
	for name, c := range alertScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runCountAlertsTest(c)
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestAlertUnrestrictedCountAlerts() {
	for name, c := range alertUnrestrictedSACSearchTestCases {
		s.Run(name, func() {
			s.runCountAlertsTest(c)
		})
	}
}

func (s *alertDatastoreSACTestSuite) runSearchAlertsTest(testparams alertSACSearchResult) {
	ctx := s.testContexts[testparams.scopeKey]
	searchResults, err := s.datastore.SearchAlerts(ctx, nil)
	s.NoError(err)
	resultsDistribution := testutils.CountSearchResultsPerClusterAndNamespace(s.T(), searchResults, s.optionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testparams.resultCounts, resultsDistribution)
}

func (s *alertDatastoreSACTestSuite) TestAlertScopedSearchAlerts() {
	for name, c := range alertScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchAlertsTest(c)
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestAlertUnrestrictedSearchAlerts() {
	for name, c := range alertUnrestrictedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchAlertsTest(c)
		})
	}
}

func countListAlertsResultsPerClusterAndNamespace(results []*storage.ListAlert) map[string]map[string]int {
	resultDistribution := make(map[string]map[string]int, 0)
	for _, result := range results {
		var clusterID string
		var namespace string
		entityData := result.GetCommonEntityInfo()
		clusterID = entityData.GetClusterId()
		namespace = entityData.GetNamespace()
		if _, clusterIDExists := resultDistribution[clusterID]; !clusterIDExists {
			resultDistribution[clusterID] = make(map[string]int, 0)
		}
		if _, namespaceExists := resultDistribution[clusterID][namespace]; !namespaceExists {
			resultDistribution[clusterID][namespace] = 0
		}
		resultDistribution[clusterID][namespace]++
	}
	return resultDistribution
}

func (s *alertDatastoreSACTestSuite) runSearchListAlertsTest(testparams alertSACSearchResult) {
	ctx := s.testContexts[testparams.scopeKey]
	searchResults, err := s.datastore.SearchListAlerts(ctx, nil)
	s.NoError(err)
	resultsDistribution := countListAlertsResultsPerClusterAndNamespace(searchResults)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testparams.resultCounts, resultsDistribution)
}

func (s *alertDatastoreSACTestSuite) TestAlertScopedSearchListAlerts() {
	for name, c := range alertScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchListAlertsTest(c)
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestAlertUnrestrictedSearchListAlerts() {
	for name, c := range alertUnrestrictedSACObjectSearchTestCases {
		s.Run(name, func() {
			s.runSearchListAlertsTest(c)
		})
	}
}

func (s *alertDatastoreSACTestSuite) runListAlertsTest(testparams alertSACSearchResult) {
	ctx := s.testContexts[testparams.scopeKey]
	searchResults, err := s.datastore.ListAlerts(ctx, nil)
	s.NoError(err)
	resultsDistribution := countListAlertsResultsPerClusterAndNamespace(searchResults)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testparams.resultCounts, resultsDistribution)
}

func (s *alertDatastoreSACTestSuite) TestAlertScopedListAlerts() {
	for name, c := range alertScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runListAlertsTest(c)
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestAlertUnrestrictedListAlerts() {
	for name, c := range alertUnrestrictedSACObjectSearchTestCases {
		s.Run(name, func() {
			s.runListAlertsTest(c)
		})
	}
}

func countSearchRawAlertsResultsPerClusterAndNamespace(results []*storage.Alert) map[string]map[string]int {
	resultDistribution := make(map[string]map[string]int, 0)
	for _, result := range results {
		var clusterID string
		var namespace string
		switch entity := result.Entity.(type) {
		case *storage.Alert_Deployment_:
			if entity.Deployment != nil {
				clusterID = entity.Deployment.GetClusterId()
				namespace = entity.Deployment.GetNamespace()
			}
		case *storage.Alert_Resource_:
			if entity.Resource != nil {
				clusterID = entity.Resource.GetClusterId()
				namespace = entity.Resource.GetNamespace()
			}
		}
		if _, clusterIDExists := resultDistribution[clusterID]; !clusterIDExists {
			resultDistribution[clusterID] = make(map[string]int, 0)
		}
		if _, namespaceExists := resultDistribution[clusterID][namespace]; !namespaceExists {
			resultDistribution[clusterID][namespace] = 0
		}
		resultDistribution[clusterID][namespace]++
	}
	return resultDistribution
}

func (s *alertDatastoreSACTestSuite) runSearchRawAlertsTest(testparams alertSACSearchResult) {
	ctx := s.testContexts[testparams.scopeKey]
	searchResults, err := s.datastore.SearchRawAlerts(ctx, nil)
	s.NoError(err)
	resultsDistribution := countSearchRawAlertsResultsPerClusterAndNamespace(searchResults)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testparams.resultCounts, resultsDistribution)
}

func (s *alertDatastoreSACTestSuite) TestAlertScopedSearchRawAlerts() {
	for name, c := range alertScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchRawAlertsTest(c)
		})
	}
}

func (s *alertDatastoreSACTestSuite) TestAlertUnrestrictedSearchRawAlerts() {
	for name, c := range alertUnrestrictedSACObjectSearchTestCases {
		s.Run(name, func() {
			s.runSearchRawAlertsTest(c)
		})
	}
}
