//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	policyStore "github.com/stackrox/rox/central/policy/store"
	policyCategoryDS "github.com/stackrox/rox/central/policycategory/datastore"
	policyCategoryMocks "github.com/stackrox/rox/central/policycategory/datastore/mocks"
	categoryPostgres "github.com/stackrox/rox/central/policycategory/store/postgres"
	policyCategoryEdgeDS "github.com/stackrox/rox/central/policycategoryedge/datastore"
	edgePostgres "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	policiesPkg "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPolicyDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(PolicyPostgresDataStoreTestSuite))
}

type PolicyPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx            context.Context
	db             postgres.DB
	mockClusterDS  *clusterDSMocks.MockDataStore
	mockNotifierDS *notifierDSMocks.MockDataStore

	datastore  DataStore
	categoryDS policyCategoryDS.DataStore

	mockCategoryDS              *policyCategoryMocks.MockDataStore
	datastoreWithMockCategoryDS DataStore
}

func (s *PolicyPostgresDataStoreTestSuite) SetupSuite() {
	s.ctx = context.Background()
}

func (s *PolicyPostgresDataStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())

	s.mockClusterDS = clusterDSMocks.NewMockDataStore(gomock.NewController(s.T()))
	s.mockNotifierDS = notifierDSMocks.NewMockDataStore(gomock.NewController(s.T()))

	categoryStorage := categoryPostgres.New(s.db)

	edgeStorage := edgePostgres.New(s.db)

	s.categoryDS = policyCategoryDS.New(categoryStorage, policyCategoryEdgeDS.New(edgeStorage))

	policyStorage := policyStore.New(s.db)
	s.datastore = New(policyStorage, s.mockClusterDS, s.mockNotifierDS, s.categoryDS)

	s.mockCategoryDS = policyCategoryMocks.NewMockDataStore(gomock.NewController(s.T()))
	s.datastoreWithMockCategoryDS = New(policyStorage, s.mockClusterDS, s.mockNotifierDS, s.mockCategoryDS)
}

func (s *PolicyPostgresDataStoreTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *PolicyPostgresDataStoreTestSuite) TestInsertUpdatePolicy() {
	policy := fixtures.GetPolicy()

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.WorkflowAdministration, resources.Cluster),
	))

	// Add policy.
	_, err := s.datastore.AddPolicy(ctx, policy)
	s.NoError(err)

	// Basic unscoped search.
	results, err := s.datastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Image Assurance").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(policy.GetId(), results[0].ID)

	policy.Categories = []string{"Image Assurance", "Boo Category 1", "Boo Category 2"}
	// Update policy
	s.NoError(s.datastore.UpdatePolicy(ctx, policy))

	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Image Assurance").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(policy.GetId(), results[0].ID)

	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Container Configuration").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 0)

	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Boo Category 1").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(policy.GetId(), results[0].ID)

	// Delete policy
	s.NoError(s.datastore.RemovePolicy(ctx, policy))
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Boo Category 1").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 0)
}

func (s *PolicyPostgresDataStoreTestSuite) TestImportPolicy() {

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.WorkflowAdministration, resources.Cluster),
	))
	s.mockClusterDS.EXPECT().GetClusters(ctx).Return([]*storage.Cluster{fixtures.GetCluster("cluster-1")}, nil)

	policy := fixtures.GetPolicy()
	policy.Id = ""

	// Import policy.
	_, allSucceeded, err := s.datastore.ImportPolicies(ctx, []*storage.Policy{policy}, true)
	s.NoError(err)
	s.True(allSucceeded)

	// Basic unscoped search.
	results, err := s.datastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Image Assurance").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(policy.GetId(), results[0].ID)

	// Delete policy
	s.NoError(s.datastore.RemovePolicy(ctx, policy))
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Image Assurance").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 0)
}

func (s *PolicyPostgresDataStoreTestSuite) TestImportOverwriteDefaultPolicy() {

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.WorkflowAdministration, resources.Cluster),
	))
	s.mockClusterDS.EXPECT().GetClusters(ctx).Return([]*storage.Cluster{fixtures.GetCluster("cluster-1")}, nil).AnyTimes()
	basePolicy := fixtures.GetPolicy()
	basePolicy.Scope = []*storage.Scope{} // clear out scope to avoid unrelated "removed_clusters_or_notifiers" errors

	sameIDExistingPolicy := basePolicy.CloneVT()
	sameIDExistingPolicy.Id = "ID1"
	sameIDExistingPolicy.Name = "existing name 1"

	sameNameExistingPolicy := basePolicy.CloneVT()
	sameNameExistingPolicy.Id = "existing ID 2"
	sameNameExistingPolicy.Name = "A very good name"

	// Same ID as sameIDExistingPolicy, unique name
	sameIDNewPolicy := sameIDExistingPolicy.CloneVT()
	sameIDNewPolicy.Name = "New Name"

	// Same name as sameNameExistingPolicy, unique ID
	sameNameNewPolicy := sameNameExistingPolicy.CloneVT()
	sameNameNewPolicy.Id = "new ID 2"

	cases := []struct {
		name                    string
		existingPolicy          *storage.Policy
		newPolicy               *storage.Policy
		existingPolicyIsDefault bool
		expectedImportError     string
		failOnOverwrite         bool
	}{
		{
			"same id as existing default policy, fail even with overwrite",
			sameIDExistingPolicy,
			sameIDNewPolicy,
			true,
			policiesPkg.ErrImportDuplicateSystemPolicyID,
			true,
		},
		{
			"same name as existing default policy, fail even with overwrite",
			sameNameExistingPolicy,
			sameNameNewPolicy,
			true,
			policiesPkg.ErrImportDuplicateSystemPolicyName,
			true,
		},
		{
			"same id as existing custom policy, succeed on overwrite",
			sameIDExistingPolicy,
			sameIDNewPolicy,
			false,
			policiesPkg.ErrImportDuplicateID,
			false,
		},
		{
			"same name as existing custom policy, succeed on overwrite",
			sameNameExistingPolicy,
			sameNameNewPolicy,
			false,
			policiesPkg.ErrImportDuplicateName,
			false,
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			_, err := s.datastore.AddPolicy(ctx, c.existingPolicy)
			s.NoError(err)

			// Use update to set the policy to default. Cannot set it to default with Add since AddPolicy wipes the setting
			if c.existingPolicyIsDefault {
				c.existingPolicy.IsDefault = true
				err = s.datastore.UpdatePolicy(ctx, c.existingPolicy)
				s.NoError(err)
			}

			// Try to import the new policies with overwrite false
			responses, allSucceeded, err := s.datastore.ImportPolicies(ctx, []*storage.Policy{c.newPolicy}, false)

			// Should fail to import due to duplicate name/id
			s.Require().NoError(err) // It's not an error just a failure?
			s.Require().False(allSucceeded)
			s.Require().Len(responses, 1)
			s.Require().Len(responses[0].GetErrors(), 1)
			s.Require().Equal(responses[0].GetErrors()[0].GetType(), c.expectedImportError)

			// Now try to import with overwrite true
			responses, allSucceeded, err = s.datastore.ImportPolicies(ctx, []*storage.Policy{c.newPolicy}, true)

			if c.failOnOverwrite {
				s.Require().NoError(err) // It's not an error just a failure?
				s.Require().False(allSucceeded)
				s.Require().Len(responses, 1)
				s.Require().Len(responses[0].GetErrors(), 1)
				s.Require().Equal(responses[0].GetErrors()[0].GetType(), c.expectedImportError) // ... should the error be different?

				// Find the existing policy and validate the name and id
				result, _, err := s.datastore.GetPolicy(ctx, c.existingPolicy.GetId())
				s.NoError(err)
				s.Equal(c.existingPolicy.GetName(), result.GetName())

				// Delete the policy
				s.NoError(s.datastore.RemovePolicy(ctx, c.existingPolicy))
			} else {
				s.NoError(err) // It's not an error just a failure?
				s.True(allSucceeded)
				s.Require().Len(responses, 1)
				s.Empty(responses[0].GetErrors())

				// Find the new policy and validate the name and id
				result, _, err := s.datastore.GetPolicy(ctx, c.newPolicy.GetId())
				s.NoError(err)
				s.Equal(c.newPolicy.GetName(), result.GetName())

				// Delete the policy
				s.NoError(s.datastore.RemovePolicy(ctx, c.newPolicy))
			}
		})
	}
}

func (s *PolicyPostgresDataStoreTestSuite) TestSearchPolicyCategoryFeatureDisabled() {
	// Policy should get upserted with category names stored inside the policy storage proto object
	// no edges, no separate category objects)
	policy := fixtures.GetPolicy()

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.WorkflowAdministration, resources.Cluster),
	))

	// Add policy.
	_, err := s.datastore.AddPolicy(ctx, policy)
	s.NoError(err)

	// Basic unscoped search.
	results, err := s.datastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Image Assurance").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(policy.GetId(), results[0].ID)
}

func (s *PolicyPostgresDataStoreTestSuite) TestSearchRawPolicies() {
	policy := fixtures.GetPolicy()

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.WorkflowAdministration, resources.Cluster),
	))

	// Add policy.
	_, err := s.datastore.AddPolicy(ctx, policy)
	s.NoError(err)

	policies, err := s.datastore.SearchRawPolicies(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(policies, 1)
	s.Len(policies[0].GetCategories(), 3)
}

func (s *PolicyPostgresDataStoreTestSuite) TestSearchPolicies() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.WorkflowAdministration, resources.Cluster),
	))

	// Create test policies
	policy1 := fixtures.GetPolicy()
	policy1.Name = "Test Policy 60-Day Image Age"
	policy1.Categories = []string{"DevOps Best Practices"}

	policy2 := fixtures.GetPolicy()
	policy2.Id = ""
	policy2.Name = "Test Policy CVSS Score>7"
	policy2.Categories = []string{"Security Best Practices"}

	policy3 := fixtures.GetPolicy()
	policy3.Id = ""
	policy3.Name = "Test Policy NVD CVSS Score>7"
	policy3.Categories = []string{"Security Best Practices", "Network Security"}

	// Add policies
	id1, err := s.datastore.AddPolicy(ctx, policy1)
	s.NoError(err)
	s.NotEmpty(id1)

	id2, err := s.datastore.AddPolicy(ctx, policy2)
	s.NoError(err)
	s.NotEmpty(id2)

	id3, err := s.datastore.AddPolicy(ctx, policy3)
	s.NoError(err)
	s.NotEmpty(id3)

	// Define test cases
	testCases := []struct {
		name              string
		query             *v1.Query
		expectedCount     int
		expectedPolicyIDs []string
		expectedNames     []string
		validateFunc      func(results []*v1.SearchResult)
	}{
		{
			name:          "empty query returns all policies with names populated",
			query:         pkgSearch.EmptyQuery(),
			expectedCount: 3,
			validateFunc: func(results []*v1.SearchResult) {
				nameMap := make(map[string]string) // id -> name
				for _, result := range results {
					s.Equal(v1.SearchCategory_POLICIES, result.GetCategory())
					nameMap[result.GetId()] = result.GetName()
				}
				s.Equal("Test Policy 60-Day Image Age", nameMap[id1])
				s.Equal("Test Policy CVSS Score>7", nameMap[id2])
				s.Equal("Test Policy NVD CVSS Score>7", nameMap[id3])
			},
		},
		{
			name:          "nil query defaults to empty query",
			query:         nil,
			expectedCount: 3,
			validateFunc: func(results []*v1.SearchResult) {
				nameMap := make(map[string]string) // id -> name
				for _, result := range results {
					s.Equal(v1.SearchCategory_POLICIES, result.GetCategory())
					nameMap[result.GetId()] = result.GetName()
				}
				s.Equal("Test Policy 60-Day Image Age", nameMap[id1])
				s.Equal("Test Policy CVSS Score>7", nameMap[id2])
				s.Equal("Test Policy NVD CVSS Score>7", nameMap[id3])
			},
		},
		{
			name:          "query by category filters correctly",
			query:         pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "DevOps Best Practices").ProtoQuery(),
			expectedCount: 1,
			expectedNames: []string{"Test Policy 60-Day Image Age"},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			results, err := s.datastore.SearchPolicies(ctx, tc.query)
			s.NoError(err)
			s.Len(results, tc.expectedCount, "Expected %d results, got %d", tc.expectedCount, len(results))

			// Validate expected policy IDs if provided
			if len(tc.expectedPolicyIDs) > 0 {
				actualIDs := make([]string, 0, len(results))
				for _, result := range results {
					actualIDs = append(actualIDs, result.GetId())
				}
				s.ElementsMatch(tc.expectedPolicyIDs, actualIDs)
			}

			// Validate expected names if provided
			if len(tc.expectedNames) > 0 {
				actualNames := make([]string, 0, len(results))
				for _, result := range results {
					actualNames = append(actualNames, result.GetName())
				}
				s.ElementsMatch(tc.expectedNames, actualNames)
			}

			// Run custom validation function if provided
			if tc.validateFunc != nil {
				tc.validateFunc(results)
			}
		})
	}

	s.NoError(s.datastore.RemovePolicy(ctx, policy1))
	s.NoError(s.datastore.RemovePolicy(ctx, policy2))
	s.NoError(s.datastore.RemovePolicy(ctx, policy3))

	// Verify cleanup
	results, err := s.datastore.SearchPolicies(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Empty(results)
}

func (s *PolicyPostgresDataStoreTestSuite) TestTransactionRollbacks() {
	policy := fixtures.GetPolicy()
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.WorkflowAdministration, resources.Cluster),
	))

	expected := errors.New("boom")
	s.mockCategoryDS.EXPECT().SetPolicyCategoriesForPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(expected).Times(1)

	_, err := s.datastoreWithMockCategoryDS.AddPolicy(ctx, policy)
	s.Equal(expected, err)

	// Verify that policy creation was rolled back since an error was encountered
	count, _ := s.datastoreWithMockCategoryDS.Count(ctx, pkgSearch.EmptyQuery())
	s.Equal(0, count)

	s.mockCategoryDS.EXPECT().SetPolicyCategoriesForPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	_, err = s.datastoreWithMockCategoryDS.AddPolicy(ctx, policy)
	s.NoError(err)

	// Verify that policy was successfully created
	count, _ = s.datastoreWithMockCategoryDS.Count(ctx, pkgSearch.EmptyQuery())
	s.Equal(1, count)

	// Clean up policy
	_ = s.datastoreWithMockCategoryDS.RemovePolicy(ctx, policy)
}

func (s *PolicyPostgresDataStoreTestSuite) TestAddDefaultsDeduplicatesCategoryNames() {
	ctx := sac.WithAllAccess(context.Background())

	// Create a policy with incorrect category names that need to be deduplicated
	policy := fixtures.GetPolicy()
	policy.Id = "test-policy-dedup"
	policy.Name = "Test Policy for Deduplication"

	// Add the policy first
	_, err := s.datastore.AddPolicy(ctx, policy)
	s.NoError(err)

	// Clear existing categories from the policy
	err = s.categoryDS.SetPolicyCategoriesForPolicy(ctx, policy.GetId(), []string{})
	s.NoError(err)

	// Create categories with incorrect names directly using the store to bypass normalization
	// These are the incorrect names: "Docker Cis" and "Devops Best Practices"
	categoryStorage := categoryPostgres.New(s.db)
	edgeStorage := edgePostgres.New(s.db)
	edgeDS := policyCategoryEdgeDS.New(edgeStorage)

	dockerCisCategory := &storage.PolicyCategory{
		Id:        uuid.NewV4().String(),
		Name:      "Docker Cis",
		IsDefault: false,
	}
	devopsCategory := &storage.PolicyCategory{
		Id:        uuid.NewV4().String(),
		Name:      "Devops Best Practices",
		IsDefault: false,
	}

	// Upsert the incorrect categories directly to the store
	err = categoryStorage.Upsert(ctx, dockerCisCategory)
	s.NoError(err)
	err = categoryStorage.Upsert(ctx, devopsCategory)
	s.NoError(err)

	// Create edges linking the policy to the incorrect categories
	dockerCisEdge := &storage.PolicyCategoryEdge{
		Id:         uuid.NewV4().String(),
		PolicyId:   policy.GetId(),
		CategoryId: dockerCisCategory.GetId(),
	}
	devopsEdge := &storage.PolicyCategoryEdge{
		Id:         uuid.NewV4().String(),
		PolicyId:   policy.GetId(),
		CategoryId: devopsCategory.GetId(),
	}
	err = edgeDS.UpsertMany(ctx, []*storage.PolicyCategoryEdge{dockerCisEdge, devopsEdge})
	s.NoError(err)

	// Verify the policy has the incorrect category names
	categories, err := s.categoryDS.GetPolicyCategoriesForPolicy(ctx, policy.GetId())
	s.NoError(err)
	s.Len(categories, 2)
	categoryNames := make([]string, len(categories))
	for i, c := range categories {
		categoryNames[i] = c.GetName()
	}
	s.Contains(categoryNames, "Docker Cis")
	s.Contains(categoryNames, "Devops Best Practices")

	// Verify the incorrect category objects exist
	searchQuery := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.PolicyCategoryName, "Docker Cis", "Devops Best Practices").ProtoQuery()
	results, err := s.categoryDS.Search(ctx, searchQuery)
	s.NoError(err)
	s.Len(results, 2) // Both incorrect categories should exist

	// Now call addDefaults which should fix the category names
	policyStorage := policyStore.New(s.db)
	addDefaults(policyStorage, s.categoryDS, s.datastore)

	// Verify the policy now has the correct category names
	categories, err = s.categoryDS.GetPolicyCategoriesForPolicy(ctx, policy.GetId())
	s.NoError(err)
	s.Len(categories, 2)
	categoryNames = make([]string, len(categories))
	for i, c := range categories {
		categoryNames[i] = c.GetName()
	}
	s.Contains(categoryNames, "Docker CIS")
	s.Contains(categoryNames, "DevOps Best Practices")
	s.NotContains(categoryNames, "Docker Cis")
	s.NotContains(categoryNames, "Devops Best Practices")

	// Verify the incorrect category objects have been deleted
	searchQuery = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.PolicyCategoryName, "Docker Cis", "Devops Best Practices").ProtoQuery()
	results, err = s.categoryDS.Search(ctx, searchQuery)
	s.NoError(err)
	s.Len(results, 0) // Both incorrect categories should be deleted

	// Verify the correct category objects exist
	searchQuery = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.PolicyCategoryName, "Docker CIS", "DevOps Best Practices").ProtoQuery()
	results, err = s.categoryDS.Search(ctx, searchQuery)
	s.NoError(err)
	s.Len(results, 2) // Both correct categories should exist

	// Clean up
	s.NoError(s.datastore.RemovePolicy(ctx, policy))
}
