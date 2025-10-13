//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestNamespaceDataStoreComprehensive(t *testing.T) {
	suite.Run(t, new(namespaceDatastoreComprehensiveSuite))
}

type namespaceDatastoreComprehensiveSuite struct {
	suite.Suite

	pgTestBase       *pgtest.TestPostgres
	datastore        DataStore
	optionsMap       searchPkg.OptionsMap
	testContexts     map[string]context.Context
	testNamespaceIDs []string
}

func (s *namespaceDatastoreComprehensiveSuite) SetupSuite() {
	var err error
	s.pgTestBase = pgtest.ForT(s.T())
	s.Require().NotNil(s.pgTestBase)
	s.datastore, err = GetTestPostgresDataStore(s.T(), s.pgTestBase.DB)
	s.Require().NoError(err)
	s.optionsMap = schema.NamespacesSchema.OptionsMap

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.Namespace)
}

func (s *namespaceDatastoreComprehensiveSuite) TearDownSuite() {
	s.pgTestBase.DB.Close()
}

func (s *namespaceDatastoreComprehensiveSuite) SetupTest() {
	s.testNamespaceIDs = make([]string, 0)
}

func (s *namespaceDatastoreComprehensiveSuite) TearDownTest() {
	for _, id := range s.testNamespaceIDs {
		s.deleteNamespace(id)
	}
}

func (s *namespaceDatastoreComprehensiveSuite) deleteNamespace(id string) {
	s.Require().NoError(s.datastore.RemoveNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], id))
}

// Test GetManyNamespaces functionality
func (s *namespaceDatastoreComprehensiveSuite) TestGetManyNamespaces() {
	// Create test namespaces
	ns1 := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA)
	ns2 := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	ns3 := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster3, testconsts.NamespaceC)

	testNamespaces := []*storage.NamespaceMetadata{ns1, ns2, ns3}
	for _, ns := range testNamespaces {
		err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns)
		s.Require().NoError(err)
		s.testNamespaceIDs = append(s.testNamespaceIDs, ns.GetId())
	}

	testCases := []struct {
		name        string
		ctx         string
		ids         []string
		expectedLen int
	}{
		{
			name:        "Get all namespaces with unrestricted access",
			ctx:         testutils.UnrestrictedReadCtx,
			ids:         []string{ns1.GetId(), ns2.GetId(), ns3.GetId()},
			expectedLen: 3,
		},
		{
			name:        "Get namespaces with cluster1 access",
			ctx:         testutils.Cluster1ReadWriteCtx,
			ids:         []string{ns1.GetId(), ns2.GetId(), ns3.GetId()},
			expectedLen: 1, // Only ns1 should be accessible
		},
		{
			name:        "Get single namespace",
			ctx:         testutils.UnrestrictedReadCtx,
			ids:         []string{ns1.GetId()},
			expectedLen: 1,
		},
		{
			name:        "Get non-existent namespace",
			ctx:         testutils.UnrestrictedReadCtx,
			ids:         []string{uuid.NewV4().String()},
			expectedLen: 0,
		},
		{
			name:        "Empty IDs slice",
			ctx:         testutils.UnrestrictedReadCtx,
			ids:         []string{},
			expectedLen: 0,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ctx := s.testContexts[tc.ctx]
			namespaces, err := s.datastore.GetManyNamespaces(ctx, tc.ids)
			s.NoError(err)
			s.Equal(tc.expectedLen, len(namespaces))

			// Verify all returned namespaces have priority set
			for _, ns := range namespaces {
				s.GreaterOrEqual(ns.GetPriority(), int64(0))
			}
		})
	}
}

// Test priority/ranking functionality
func (s *namespaceDatastoreComprehensiveSuite) TestPriorityAndRanking() {
	// Create test namespaces with different risks
	ns1 := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA)
	ns2 := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)

	// Add namespaces
	for _, ns := range []*storage.NamespaceMetadata{ns1, ns2} {
		err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns)
		s.Require().NoError(err)
		s.testNamespaceIDs = append(s.testNamespaceIDs, ns.GetId())
	}

	// Test that priorities are set when retrieving namespaces
	s.Run("GetNamespace sets priority", func() {
		ns, found, err := s.datastore.GetNamespace(s.testContexts[testutils.UnrestrictedReadCtx], ns1.GetId())
		s.NoError(err)
		s.True(found)
		s.GreaterOrEqual(ns.GetPriority(), int64(0))
	})

	s.Run("GetAllNamespaces sets priorities", func() {
		namespaces, err := s.datastore.GetAllNamespaces(s.testContexts[testutils.UnrestrictedReadCtx])
		s.NoError(err)
		for _, ns := range namespaces {
			s.GreaterOrEqual(ns.GetPriority(), int64(0))
		}
	})

	s.Run("SearchNamespaces sets priorities", func() {
		namespaces, err := s.datastore.SearchNamespaces(s.testContexts[testutils.UnrestrictedReadCtx], searchPkg.EmptyQuery())
		s.NoError(err)
		for _, ns := range namespaces {
			s.GreaterOrEqual(ns.GetPriority(), int64(0))
		}
	})
}

// Test priority sorting in Search method
func (s *namespaceDatastoreComprehensiveSuite) TestPriorityBasedSorting() {
	// Create test namespaces
	ns1 := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA)
	ns2 := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)

	for _, ns := range []*storage.NamespaceMetadata{ns1, ns2} {
		err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns)
		s.Require().NoError(err)
		s.testNamespaceIDs = append(s.testNamespaceIDs, ns.GetId())
	}

	testCases := []struct {
		name      string
		sortField string
		reversed  bool
	}{
		{
			name:      "Search with priority sorting",
			sortField: searchPkg.NamespacePriority.String(),
			reversed:  false,
		},
		{
			name:      "Search with reversed priority sorting",
			sortField: searchPkg.NamespacePriority.String(),
			reversed:  true,
		},
		{
			name:      "Search without priority sorting",
			sortField: searchPkg.Namespace.String(),
			reversed:  false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			query := &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{
							Field:    tc.sortField,
							Reversed: tc.reversed,
						},
					},
				},
			}

			results, err := s.datastore.Search(s.testContexts[testutils.UnrestrictedReadCtx], query)
			s.NoError(err)
			s.GreaterOrEqual(len(results), 2)
		})
	}
}

// Test error handling and edge cases
func (s *namespaceDatastoreComprehensiveSuite) TestErrorHandling() {
	testCases := []struct {
		name        string
		testFunc    func() error
		expectError bool
	}{
		{
			name: "AddNamespace with nil namespace",
			testFunc: func() error {
				return s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], nil)
			},
			expectError: true,
		},
		{
			name: "UpdateNamespace with nil namespace",
			testFunc: func() error {
				return s.datastore.UpdateNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], nil)
			},
			expectError: true,
		},
		{
			name: "RemoveNamespace with empty ID",
			testFunc: func() error {
				return s.datastore.RemoveNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], "")
			},
			expectError: false,
		},
		{
			name: "RemoveNamespace with non-existent ID",
			testFunc: func() error {
				return s.datastore.RemoveNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], uuid.NewV4().String())
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.testFunc()
			if tc.expectError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}

	// Test GetNamespace edge cases
	getNamespaceTests := []struct {
		name        string
		id          string
		expectErr   bool
		expectFound bool
	}{
		{
			name:        "GetNamespace with empty ID",
			id:          "",
			expectErr:   false,
			expectFound: false,
		},
		{
			name:        "GetNamespace with non-existent ID",
			id:          uuid.NewV4().String(),
			expectErr:   false,
			expectFound: false,
		},
	}

	for _, tc := range getNamespaceTests {
		s.Run(tc.name, func() {
			ns, found, err := s.datastore.GetNamespace(s.testContexts[testutils.UnrestrictedReadCtx], tc.id)
			if tc.expectErr {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(tc.expectFound, found)
				if !found {
					s.Nil(ns)
				}
			}
		})
	}

	// Test search/count with nil query (less valuable tests)
	s.Run("Search with nil query", func() {
		results, err := s.datastore.Search(s.testContexts[testutils.UnrestrictedReadCtx], nil)
		s.NoError(err)
		s.Empty(results) // No namespaces in database for this test
	})

	s.Run("Count with nil query", func() {
		count, err := s.datastore.Count(s.testContexts[testutils.UnrestrictedReadCtx], nil)
		s.NoError(err)
		s.Equal(0, count) // No namespaces in database for this test
	})
}

// Test ranking removal when namespace is deleted
func (s *namespaceDatastoreComprehensiveSuite) TestRankingRemovalOnDelete() {
	ns := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA)

	// Add namespace
	err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns)
	s.Require().NoError(err)
	s.testNamespaceIDs = append(s.testNamespaceIDs, ns.GetId())

	// Verify namespace exists and has priority
	retrievedNs, found, err := s.datastore.GetNamespace(s.testContexts[testutils.UnrestrictedReadCtx], ns.GetId())
	s.NoError(err)
	s.True(found)
	s.GreaterOrEqual(retrievedNs.GetPriority(), int64(0))

	// Remove namespace
	err = s.datastore.RemoveNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns.GetId())
	s.NoError(err)

	// Verify namespace is gone
	_, found, err = s.datastore.GetNamespace(s.testContexts[testutils.UnrestrictedReadCtx], ns.GetId())
	s.NoError(err)
	s.False(found)

	// Remove from test cleanup list since we already deleted it
	for i, id := range s.testNamespaceIDs {
		if id == ns.GetId() {
			s.testNamespaceIDs = append(s.testNamespaceIDs[:i], s.testNamespaceIDs[i+1:]...)
			break
		}
	}
}

// Test race conditions in searchNamespaces method
func (s *namespaceDatastoreComprehensiveSuite) TestRaceConditionHandling() {
	ns := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA)

	// Add namespace
	err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns)
	s.Require().NoError(err)
	s.testNamespaceIDs = append(s.testNamespaceIDs, ns.GetId())

	// Test concurrent operations
	var wg sync.WaitGroup
	var searchErr, deleteErr error

	wg.Add(2)

	// Concurrent search
	go func() {
		defer wg.Done()
		_, searchErr = s.datastore.SearchNamespaces(s.testContexts[testutils.UnrestrictedReadCtx], searchPkg.EmptyQuery())
	}()

	// Concurrent delete
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Small delay to increase chance of race
		deleteErr = s.datastore.RemoveNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns.GetId())
	}()

	wg.Wait()

	// Both operations should succeed (or search should handle missing namespace gracefully)
	s.NoError(searchErr)
	s.NoError(deleteErr)

	// Remove from test cleanup list since we deleted it
	for i, id := range s.testNamespaceIDs {
		if id == ns.GetId() {
			s.testNamespaceIDs = append(s.testNamespaceIDs[:i], s.testNamespaceIDs[i+1:]...)
			break
		}
	}
}

// Test performance with larger datasets
func (s *namespaceDatastoreComprehensiveSuite) TestPerformanceWithLargeDataset() {
	const numNamespaces = 100
	namespaces := make([]*storage.NamespaceMetadata, 0, numNamespaces)

	// Create many namespaces
	for i := 0; i < numNamespaces; i++ {
		ns := fixtures.GetScopedNamespace(
			uuid.NewV4().String(),
			testconsts.Cluster1,
			fmt.Sprintf("test-namespace-%d", i),
		)
		namespaces = append(namespaces, ns)

		err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns)
		s.Require().NoError(err)
		s.testNamespaceIDs = append(s.testNamespaceIDs, ns.GetId())
	}

	// Test performance of various operations
	start := time.Now()
	allNs, err := s.datastore.GetAllNamespaces(s.testContexts[testutils.UnrestrictedReadCtx])
	elapsed := time.Since(start)

	s.NoError(err)
	s.GreaterOrEqual(len(allNs), numNamespaces)
	s.Less(elapsed, 5*time.Second, "GetAllNamespaces should complete within 5 seconds")

	// Test GetManyNamespaces performance
	ids := make([]string, 0, numNamespaces)
	for _, ns := range namespaces {
		ids = append(ids, ns.GetId())
	}

	start = time.Now()
	manyNs, err := s.datastore.GetManyNamespaces(s.testContexts[testutils.UnrestrictedReadCtx], ids)
	elapsed = time.Since(start)

	s.NoError(err)
	s.Equal(numNamespaces, len(manyNs))
	s.Less(elapsed, 5*time.Second, "GetManyNamespaces should complete within 5 seconds")
}

// Test SearchResults functionality more thoroughly
func (s *namespaceDatastoreComprehensiveSuite) TestSearchResultsComprehensive() {
	ns1 := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA)
	ns2 := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)

	for _, ns := range []*storage.NamespaceMetadata{ns1, ns2} {
		err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns)
		s.Require().NoError(err)
		s.testNamespaceIDs = append(s.testNamespaceIDs, ns.GetId())
	}

	testCases := []struct {
		name          string
		query         *v1.Query
		expectResults bool
		validateFunc  func([]*v1.SearchResult)
	}{
		{
			name:          "SearchResults returns proper format",
			query:         searchPkg.EmptyQuery(),
			expectResults: true,
			validateFunc: func(results []*v1.SearchResult) {
				s.GreaterOrEqual(len(results), 2)
				for _, result := range results {
					s.NotEmpty(result.GetId())
					s.NotEmpty(result.GetName())
					s.Equal(v1.SearchCategory_NAMESPACES, result.GetCategory())
					s.NotEmpty(result.GetLocation())
					s.GreaterOrEqual(result.GetScore(), float64(0))
				}
			},
		},
		{
			name:          "SearchResults with specific query",
			query:         searchPkg.NewQueryBuilder().AddStrings(searchPkg.Namespace, testconsts.NamespaceA).ProtoQuery(),
			expectResults: true,
			validateFunc: func(results []*v1.SearchResult) {
				// Should find at least the namespace we're searching for
				found := false
				for _, result := range results {
					if result.GetName() == testconsts.NamespaceA {
						found = true
						break
					}
				}
				s.True(found, "Should find the searched namespace")
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			results, err := s.datastore.SearchResults(s.testContexts[testutils.UnrestrictedReadCtx], tc.query)
			s.NoError(err)
			if tc.validateFunc != nil {
				tc.validateFunc(results)
			}
		})
	}
}

// Test concurrent operations
func (s *namespaceDatastoreComprehensiveSuite) TestConcurrentOperations() {
	const numGoroutines = 10
	const numOperationsPerGoroutine = 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperationsPerGoroutine; j++ {
				ns := fixtures.GetScopedNamespace(
					uuid.NewV4().String(),
					testconsts.Cluster1,
					fmt.Sprintf("concurrent-ns-%d-%d", goroutineID, j),
				)

				// Add namespace
				if err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns); err != nil {
					errors <- err
					continue
				}

				// Get namespace
				if _, _, err := s.datastore.GetNamespace(s.testContexts[testutils.UnrestrictedReadCtx], ns.GetId()); err != nil {
					errors <- err
				}

				// Update namespace
				ns.Priority = int64(j)
				if err := s.datastore.UpdateNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns); err != nil {
					errors <- err
				}

				// Delete namespace
				if err := s.datastore.RemoveNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns.GetId()); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		s.NoError(err)
	}
}

// Test GetNamespacesForSAC edge cases
func (s *namespaceDatastoreComprehensiveSuite) TestGetNamespacesForSACEdgeCases() {
	ns := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA)
	err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], ns)
	s.Require().NoError(err)
	s.testNamespaceIDs = append(s.testNamespaceIDs, ns.GetId())

	testCases := []struct {
		name              string
		contextKey        string
		expectMinResults  int
		expectPrioritySet bool
		validateNamespace bool
	}{
		{
			name:              "GetNamespacesForSAC with full access",
			contextKey:        testutils.UnrestrictedReadCtx,
			expectMinResults:  1,
			expectPrioritySet: false, // Full access path doesn't set priorities
			validateNamespace: false,
		},
		{
			name:              "GetNamespacesForSAC with limited access",
			contextKey:        testutils.Cluster1NamespaceAReadWriteCtx,
			expectMinResults:  1,
			expectPrioritySet: true, // SearchNamespaces path sets priorities
			validateNamespace: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			namespaces, err := s.datastore.GetNamespacesForSAC(s.testContexts[tc.contextKey])
			s.NoError(err)
			s.GreaterOrEqual(len(namespaces), tc.expectMinResults)

			if tc.validateNamespace {
				// Should find our test namespace
				found := false
				for _, retrievedNs := range namespaces {
					if retrievedNs.GetId() == ns.GetId() {
						found = true
						if tc.expectPrioritySet {
							s.GreaterOrEqual(retrievedNs.GetPriority(), int64(0))
						}
						break
					}
				}
				s.True(found)
			}
		})
	}
}
