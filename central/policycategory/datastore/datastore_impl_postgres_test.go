//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/policycategory/store/postgres"
	edgeDataStore "github.com/stackrox/rox/central/policycategoryedge/datastore"
	policyCategoryEdgePostgres "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestPolicyCategoryDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(PolicyCategoryPostgresDataStoreTestSuite))
}

type PolicyCategoryPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx           context.Context
	db            postgres.DB
	datastore     DataStore
	edgeDatastore edgeDataStore.DataStore
}

func (s *PolicyCategoryPostgresDataStoreTestSuite) SetupSuite() {

	s.ctx = context.Background()
}

func (s *PolicyCategoryPostgresDataStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())

	policyCategoryEdgeStorage := policyCategoryEdgePostgres.New(s.db)
	s.edgeDatastore = edgeDataStore.New(policyCategoryEdgeStorage)

	policyCategoryStore := pgStore.New(s.db)
	s.datastore = New(policyCategoryStore, s.edgeDatastore)
}

func (s *PolicyCategoryPostgresDataStoreTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *PolicyCategoryPostgresDataStoreTestSuite) TestSearchWithPostgres() {
	category := &storage.PolicyCategory{
		Id:        "id-1",
		Name:      "Boo's Category",
		IsDefault: false,
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.WorkflowAdministration),
	))

	// Add category.
	_, err := s.datastore.AddPolicyCategory(ctx, category)
	s.NoError(err)

	// Basic unscoped search.
	results, err := s.datastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, category.GetName()).ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)

	// Add new category.
	anotherCategory := &storage.PolicyCategory{
		Id:        "id-2",
		Name:      "Boo's Other Category",
		IsDefault: false,
	}
	_, err = s.datastore.AddPolicyCategory(ctx, anotherCategory)
	s.NoError(err)

	// Search multiple images.
	categories, err := s.datastore.SearchRawPolicyCategories(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(categories, 2)

	// Search for just one category.
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, category.GetName()).ProtoQuery()
	categories, err = s.datastore.SearchRawPolicyCategories(ctx, q)
	s.NoError(err)
	s.Len(categories, 1)
	s.Equal("id-1", categories[0].GetId())

}

func (s *PolicyCategoryPostgresDataStoreTestSuite) TestSearchPolicyCategories() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.WorkflowAdministration),
	))

	// Create test policy categories
	category1 := &storage.PolicyCategory{
		Id:        "test-cat-1",
		Name:      "Security Best Practices",
		IsDefault: true,
	}

	category2 := &storage.PolicyCategory{
		Id:        "test-cat-2",
		Name:      "Package Management",
		IsDefault: true,
	}

	category3 := &storage.PolicyCategory{
		Id:        "test-cat-3",
		Name:      "Privileges",
		IsDefault: false,
	}

	// Add categories
	_, err := s.datastore.AddPolicyCategory(ctx, category1)
	s.NoError(err)

	_, err = s.datastore.AddPolicyCategory(ctx, category2)
	s.NoError(err)

	_, err = s.datastore.AddPolicyCategory(ctx, category3)
	s.NoError(err)

	// Define test cases
	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
		expectedIDs   []string
		expectedNames []string
		validateFunc  func(results []*v1.SearchResult)
	}{
		{
			name:          "empty query returns all categories with names populated",
			query:         pkgSearch.EmptyQuery(),
			expectedCount: 3,
			expectedNames: []string{"Security Best Practices", "Package Management", "Privileges"},
		},
		{
			name:          "nil query defaults to empty query",
			query:         nil,
			expectedCount: 3,
			expectedNames: []string{"Security Best Practices", "Package Management", "Privileges"},
		},
		{
			name:          "query by exact category name",
			query:         pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.PolicyCategoryName, "Package Management").ProtoQuery(),
			expectedCount: 1,
			expectedIDs:   []string{"test-cat-2"},
			expectedNames: []string{"Package Management"},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			results, err := s.datastore.SearchPolicyCategories(ctx, tc.query)
			s.NoError(err)
			s.Len(results, tc.expectedCount, "Expected %d results, got %d", tc.expectedCount, len(results))

			actualIDs := make([]string, 0, len(results))
			actualNames := make([]string, 0, len(results))
			for _, result := range results {
				actualIDs = append(actualIDs, result.GetId())
				actualNames = append(actualNames, result.GetName())
				s.Equal(result.GetCategory(), v1.SearchCategory_POLICY_CATEGORIES)
			}

			if len(tc.expectedNames) > 0 {
				s.ElementsMatch(tc.expectedNames, actualNames)
			}

			if len(tc.expectedIDs) > 0 {
				s.ElementsMatch(tc.expectedIDs, actualIDs)
			}
		})
	}

	// Clean up
	s.NoError(s.datastore.DeletePolicyCategory(ctx, category1.GetId()))
	s.NoError(s.datastore.DeletePolicyCategory(ctx, category2.GetId()))
	s.NoError(s.datastore.DeletePolicyCategory(ctx, category3.GetId()))

	// Verify cleanup
	results, err := s.datastore.SearchPolicyCategories(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Empty(results)
}
