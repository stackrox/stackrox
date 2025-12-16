//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestPolicyCategoryEdgeDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(PolicyCategoryEdgePostgresDataStoreTestSuite))
}

type PolicyCategoryEdgePostgresDataStoreTestSuite struct {
	suite.Suite

	ctx       context.Context
	db        postgres.DB
	datastore DataStore
}

func (s *PolicyCategoryEdgePostgresDataStoreTestSuite) SetupSuite() {
	s.ctx = context.Background()
}

func (s *PolicyCategoryEdgePostgresDataStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())

	policyCategoryEdgeStorage := pgStore.New(s.db)
	s.datastore = New(policyCategoryEdgeStorage)
}

func (s *PolicyCategoryEdgePostgresDataStoreTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *PolicyCategoryEdgePostgresDataStoreTestSuite) TestSearchEdges() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.WorkflowAdministration),
	))

	// Insert test policies directly to database (needed for foreign key constraints)
	_, err := s.db.Exec(ctx, `
		INSERT INTO policies (id, name, disabled, lifecyclestages, categories)
		VALUES
			('policy-1', 'Test Policy 1', false, '{}', '{}'),
			('policy-2', 'Test Policy 2',  false, '{}', '{}')
	`)
	s.NoError(err)

	// Insert test categories directly to database
	_, err = s.db.Exec(ctx, `
		INSERT INTO policy_categories (id, name)
		VALUES
			('category-1', 'Test Category 1'),
			('category-2', 'Test Category 2')
	`)
	s.NoError(err)

	// Create test edges
	edge1 := &storage.PolicyCategoryEdge{
		Id:         uuid.NewV4().String(),
		PolicyId:   "policy-1",
		CategoryId: "category-1",
	}

	edge2 := &storage.PolicyCategoryEdge{
		Id:         uuid.NewV4().String(),
		PolicyId:   "policy-2",
		CategoryId: "category-1",
	}

	edge3 := &storage.PolicyCategoryEdge{
		Id:         uuid.NewV4().String(),
		PolicyId:   "policy-1",
		CategoryId: "category-2",
	}

	// Add edges
	err = s.datastore.UpsertMany(ctx, []*storage.PolicyCategoryEdge{edge1, edge2, edge3})
	s.NoError(err)

	// Define test cases
	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
		expectedIDs   []string
		validateFunc  func(results []*v1.SearchResult)
	}{
		{
			name:          "empty query returns all edges with names populated",
			query:         pkgSearch.EmptyQuery(),
			expectedCount: 3,
			expectedIDs:   []string{edge1.GetId(), edge2.GetId(), edge3.GetId()},
			validateFunc: func(results []*v1.SearchResult) {
				for _, result := range results {
					s.Equal(v1.SearchCategory_POLICY_CATEGORY_EDGE, result.GetCategory())
					// Name should equal ID for edges
					s.Equal(result.GetId(), result.GetName())
				}
			},
		},
		{
			name:          "nil query defaults to empty query",
			query:         nil,
			expectedCount: 3,
			validateFunc: func(results []*v1.SearchResult) {
				for _, result := range results {
					s.Equal(result.GetId(), result.GetName(), "Name should equal ID")
				}
			},
		},
		{
			name:          "query by policy ID",
			query:         pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.PolicyID, "policy-1").ProtoQuery(),
			expectedCount: 2,
			expectedIDs:   []string{edge1.GetId(), edge3.GetId()},
			validateFunc: func(results []*v1.SearchResult) {
				for _, result := range results {
					s.Equal(result.GetId(), result.GetName(), "Name should equal ID")
				}
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			results, err := s.datastore.SearchEdges(ctx, tc.query)
			s.NoError(err)
			s.Len(results, tc.expectedCount, "Expected %d results, got %d", tc.expectedCount, len(results))

			// Validate expected IDs if provided
			if len(tc.expectedIDs) > 0 {
				actualIDs := make([]string, 0, len(results))
				for _, result := range results {
					actualIDs = append(actualIDs, result.GetId())
				}
				s.ElementsMatch(tc.expectedIDs, actualIDs)
			}

			// Run custom validation function if provided
			if tc.validateFunc != nil {
				tc.validateFunc(results)
			}
		})
	}

	// Clean up - delete in reverse order (edges first, then policies, then categories)
	s.NoError(s.datastore.DeleteMany(ctx, edge1.GetId(), edge2.GetId(), edge3.GetId()))

	// Delete test policies from database
	_, err = s.db.Exec(ctx, `DELETE FROM policies WHERE id IN ('policy-1', 'policy-2')`)
	s.NoError(err)

	// Delete test categories from database
	_, err = s.db.Exec(ctx, `DELETE FROM policy_categories WHERE id IN ('category-1', 'category-2')`)
	s.NoError(err)

	// Verify cleanup
	results, err := s.datastore.SearchEdges(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Empty(results)
}
