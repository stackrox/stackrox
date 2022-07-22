package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/secret/mappings"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	cleanupCtxKey = testutils.UnrestrictedReadWriteCtx
)

func TestSecretDatastoreSAC(t *testing.T) {
	suite.Run(t, new(secretDatastoreSACTestSuite))
}

type secretDatastoreSACTestSuite struct {
	suite.Suite

	engine *rocksdb.RocksDB
	index  bleve.Index

	pool *pgxpool.Pool

	datastore DataStore

	testContexts map[string]context.Context

	testSecretIDs []string
}

func (s *secretDatastoreSACTestSuite) SetupSuite() {
	var err error
	secretObj := "secretSACTest"

	if features.PostgresDatastore.Enabled() {
		pgtestbase := pgtest.ForT(s.T())
		s.Require().NotNil(pgtestbase)
		s.pool = pgtestbase.Pool
		s.datastore, err = GetTestPostgresDataStore(s.T(), s.pool)
		s.Require().NoError(err)
	} else {
		s.engine, err = rocksdb.NewTemp(secretObj)
		s.NoError(err)
		s.index, err = globalindex.TempInitializeIndices(secretObj)
		s.NoError(err)

		s.datastore, err = GetTestRocksBleveDataStore(s.T(), s.engine, s.index)
		s.Require().NoError(err)
	}

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Secret)
}

func (s *secretDatastoreSACTestSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	} else {
		s.Require().NoError(rocksdb.CloseAndRemove(s.engine))
		s.Require().NoError(s.index.Close())
	}
}

func (s *secretDatastoreSACTestSuite) SetupTest() {
	s.testSecretIDs = make([]string, 0)

	// Inject test data set for search tests
	secrets := fixtures.GetSACTestSecretSet()
	var err error
	for _, secret := range secrets {
		err = s.datastore.UpsertSecret(s.testContexts[testutils.UnrestrictedReadWriteCtx], secret)
		s.testSecretIDs = append(s.testSecretIDs, secret.GetId())
		s.NoError(err)
	}
}

func (s *secretDatastoreSACTestSuite) TearDownTest() {
	for _, id := range s.testSecretIDs {
		s.cleanupSecret(id)
	}
}

func (s *secretDatastoreSACTestSuite) cleanupSecret(ID string) {
	err := s.datastore.RemoveSecret(s.testContexts[cleanupCtxKey], ID)
	s.NoError(err)
}

func (s *secretDatastoreSACTestSuite) TestUpsertSecret() {
	testedVerb := "upsert"
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), testedVerb)

	for name, c := range cases {
		s.Run(name, func() {
			testSecret := fixtures.GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			s.testSecretIDs = append(s.testSecretIDs, testSecret.GetId())
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertSecret(ctx, testSecret)
			defer s.cleanupSecret(testSecret.GetId())
			if !c.ExpectError {
				s.NoError(err)
			} else {
				s.Equal(c.ExpectedError, err)
			}
		})
	}
}

func (s *secretDatastoreSACTestSuite) TestGetSecret() {
	var err error
	testSecret := fixtures.GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err = s.datastore.UpsertSecret(s.testContexts[testutils.UnrestrictedReadWriteCtx], testSecret)
	s.testSecretIDs = append(s.testSecretIDs, testSecret.GetId())
	s.NoError(err)

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			readSecret, found, getErr := s.datastore.GetSecret(ctx, testSecret.GetId())
			s.NoError(getErr)
			if c.ExpectedFound {
				s.True(found)
				s.Equal(*testSecret, *readSecret)
			} else {
				s.False(found)
				s.Nil(readSecret)
			}
		})
	}
}

func (s *secretDatastoreSACTestSuite) TestRemoveSecret() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			testSecret := fixtures.GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			s.testSecretIDs = append(s.testSecretIDs, testSecret.GetId())
			ctx := s.testContexts[c.ScopeKey]
			var err error
			err = s.datastore.UpsertSecret(s.testContexts[testutils.UnrestrictedReadWriteCtx], testSecret)
			defer s.cleanupSecret(testSecret.GetId())
			s.NoError(err)
			err = s.datastore.RemoveSecret(ctx, testSecret.GetId())
			if !c.ExpectError {
				s.NoError(err)
			} else {
				s.Equal(c.ExpectedError, err)
			}
		})
	}
}

type secretSACSearchResult struct {
	scopeKey     string
	resultCounts map[string]map[string]int // Top level is the cluster ID, then namespace
}

// The SAC secret test dataset defined in pkg/fixtures/secret.go has the following secret distribution
// Cluster1::NamespaceA: 8 secrets
// Cluster1::NamespaceB: 5 secrets
// Cluster2::NamespaceB: 3 secrets
// Cluster2::NamespaceC: 2 secrets
var secretScopedSACSearchTestCases = map[string]secretSACSearchResult{
	"Cluster1 read-write access should only see Cluster1 secrets": {
		scopeKey: testutils.Cluster1ReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceA: 8,
				testconsts.NamespaceB: 5,
			},
		},
	},
	"Cluster1 and NamespaceA read-write access should only see Cluster1 and NamespaceA secrets": {
		scopeKey: testutils.Cluster1NamespaceAReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceA: 8,
			},
		},
	},
	"Cluster1 and NamespaceB read-write access should only see Cluster1 and NamespaceB secrets": {
		scopeKey: testutils.Cluster1NamespaceBReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceB: 5,
			},
		},
	},
	"Cluster1 and NamespaceC read-write access should see no secret": {
		scopeKey:     testutils.Cluster1NamespaceCReadWriteCtx,
		resultCounts: map[string]map[string]int{},
	},
	"Cluster1 and Namespaces A and B read-write access should only see appropriate cluster/namespace secrets": {
		scopeKey: testutils.Cluster1NamespacesABReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceA: 8,
				testconsts.NamespaceB: 5,
			},
		},
	},
	"Cluster1 and Namespaces A and C read-write access should only see appropriate cluster/namespace secrets": {
		scopeKey: testutils.Cluster1NamespacesACReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceA: 8,
			},
		},
	},
	"Cluster1 and Namespaces B and C read-write access should only see appropriate cluster/namespace secrets": {
		scopeKey: testutils.Cluster1NamespacesBCReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster1: {
				testconsts.NamespaceB: 5,
			},
		},
	},
	"Cluster2 read-write access should only see Cluster2 secrets": {
		scopeKey: testutils.Cluster2ReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceB: 3,
				testconsts.NamespaceC: 2,
			},
		},
	},
	"Cluster2 and NamespaceA read-write access should see no secret": {
		scopeKey:     testutils.Cluster2NamespaceAReadWriteCtx,
		resultCounts: map[string]map[string]int{},
	},
	"Cluster2 and NamespaceB read-write access should only see Cluster2 and NamespaceB secrets": {
		scopeKey: testutils.Cluster2NamespaceBReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceB: 3,
			},
		},
	},
	"Cluster2 and NamespaceC read-write access should only see Cluster2 and NamespaceC secrets": {
		scopeKey: testutils.Cluster2NamespaceCReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceC: 2,
			},
		},
	},
	"Cluster2 and Namespaces A and B read-write access should only see appropriate cluster/namespace secrets": {
		scopeKey: testutils.Cluster2NamespacesABReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceB: 3,
			},
		},
	},
	"Cluster2 and Namespaces A and C read-write access should only see appropriate cluster/namespace secrets": {
		scopeKey: testutils.Cluster2NamespacesACReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceC: 2,
			},
		},
	},
	"Cluster2 and Namespaces B and C read-write access should only see appropriate cluster/namespace secrets": {
		scopeKey: testutils.Cluster2NamespacesBCReadWriteCtx,
		resultCounts: map[string]map[string]int{
			testconsts.Cluster2: {
				testconsts.NamespaceB: 3,
				testconsts.NamespaceC: 2,
			},
		},
	},
}

var secretUnrestrictedSACSearchTestCases = map[string]secretSACSearchResult{
	"full read access should see all secrets": {
		// SAC search fields are not injected in query when running unscoped search
		// Therefore results cannot be dispatched per cluster and namespace
		scopeKey: testutils.UnrestrictedReadCtx,
		resultCounts: map[string]map[string]int{
			"": {"": 18},
		},
	},
	"full read-write access should see all secrets": {
		// SAC search fields are not injected in query when running unscoped search
		// Therefore results cannot be dispatched per cluster and namespace
		scopeKey: testutils.UnrestrictedReadWriteCtx,
		resultCounts: map[string]map[string]int{
			"": {"": 18},
		},
	},
}

var secretUnrestrictedSACObjectSearchTestCases = map[string]secretSACSearchResult{
	"full read access should see all secrets": {
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
		},
	},
	"full read-write access should see all secrets": {
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
		},
	},
}

func (s *secretDatastoreSACTestSuite) runCountTest(testParams secretSACSearchResult) {
	ctx := s.testContexts[testParams.scopeKey]
	resultCount, err := s.datastore.Count(ctx, nil)
	s.NoError(err)
	expectedResultCount := testutils.AggregateCounts(s.T(), testParams.resultCounts)
	s.Equal(expectedResultCount, resultCount)
}

func (s *secretDatastoreSACTestSuite) TestSecretScopedCount() {
	for name, c := range secretScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runCountTest(c)
		})
	}
}

func (s *secretDatastoreSACTestSuite) TestSecretUnrestrictedCount() {
	for name, c := range secretUnrestrictedSACSearchTestCases {
		s.Run(name, func() {
			s.runCountTest(c)
		})
	}
}

func (s *secretDatastoreSACTestSuite) runCountSecretsTest(testParams secretSACSearchResult) {
	ctx := s.testContexts[testParams.scopeKey]
	resultCount, err := s.datastore.CountSecrets(ctx)
	s.NoError(err)
	expectedResultCount := testutils.AggregateCounts(s.T(), testParams.resultCounts)
	s.Equal(expectedResultCount, resultCount)
}

func (s *secretDatastoreSACTestSuite) TestSecretScopedCountSecrets() {
	for name, c := range secretScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runCountSecretsTest(c)
		})
	}
}

func (s *secretDatastoreSACTestSuite) TestSecretUnrestrictedCountSecrets() {
	for name, c := range secretUnrestrictedSACSearchTestCases {
		s.Run(name, func() {
			s.runCountSecretsTest(c)
		})
	}
}

func (s *secretDatastoreSACTestSuite) runSearchTest(testParams secretSACSearchResult) {
	ctx := s.testContexts[testParams.scopeKey]
	searchResults, err := s.datastore.Search(ctx, nil)
	s.NoError(err)
	resultCounts := testutils.CountResultsPerClusterAndNamespace(s.T(), searchResults, mappings.OptionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testParams.resultCounts, resultCounts)
}

func (s *secretDatastoreSACTestSuite) TestSecretScopedSearch() {
	for name, c := range secretScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *secretDatastoreSACTestSuite) TestSecretUnrestrictedSearch() {
	for name, c := range secretUnrestrictedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *secretDatastoreSACTestSuite) runSearchSecretsTest(testParams secretSACSearchResult) {
	ctx := s.testContexts[testParams.scopeKey]
	searchResults, err := s.datastore.SearchSecrets(ctx, nil)
	s.NoError(err)
	resultDistribution := testutils.CountSearchResultsPerClusterAndNamespace(s.T(), searchResults, mappings.OptionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testParams.resultCounts, resultDistribution)
}

func (s *secretDatastoreSACTestSuite) TestSecretScopedSearchSecrets() {
	for name, c := range secretScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchSecretsTest(c)
		})
	}
}

func (s *secretDatastoreSACTestSuite) TestSecretUnrestrictedSearchSecrets() {
	for name, c := range secretUnrestrictedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchSecretsTest(c)
		})
	}
}

func (s *secretDatastoreSACTestSuite) runSearchListSecretsTest(testParams secretSACSearchResult) {
	ctx := s.testContexts[testParams.scopeKey]
	searchResults, err := s.datastore.SearchListSecrets(ctx, nil)
	s.NoError(err)
	resultObjects := make([]sac.NamespaceScopedObject, 0, len(searchResults))
	for ix := range searchResults {
		resultObjects = append(resultObjects, searchResults[ix])
	}
	resultCount := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjects)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testParams.resultCounts, resultCount)
}

func (s *secretDatastoreSACTestSuite) TestSecretScopedSearchListSecrets() {
	for name, c := range secretScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchListSecretsTest(c)
		})
	}
}

func (s *secretDatastoreSACTestSuite) TestSecretUnrestrictedSearchListSecrets() {
	for name, c := range secretUnrestrictedSACObjectSearchTestCases {
		s.Run(name, func() {
			s.runSearchListSecretsTest(c)
		})
	}
}

func (s *secretDatastoreSACTestSuite) runSearchRawSecretsTest(testParams secretSACSearchResult) {
	ctx := s.testContexts[testParams.scopeKey]
	searchResults, err := s.datastore.SearchRawSecrets(ctx, nil)
	s.NoError(err)
	resultObjects := make([]sac.NamespaceScopedObject, 0, len(searchResults))
	for ix := range searchResults {
		resultObjects = append(resultObjects, searchResults[ix])
	}
	resultCount := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjects)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testParams.resultCounts, resultCount)
}

func (s *secretDatastoreSACTestSuite) TestSecretScopedSearchRawSecrets() {
	for name, c := range secretScopedSACSearchTestCases {
		s.Run(name, func() {
			s.runSearchRawSecretsTest(c)
		})
	}
}

func (s *secretDatastoreSACTestSuite) TestSecretUnrestrictedSearchRawSecrets() {
	for name, c := range secretUnrestrictedSACObjectSearchTestCases {
		s.Run(name, func() {
			s.runSearchRawSecretsTest(c)
		})
	}
}
