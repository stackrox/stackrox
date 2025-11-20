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
	edgeDS := policyCategoryEdgeDS.New(edgeStorage)

	s.categoryDS = policyCategoryDS.New(categoryStorage, edgeDS)

	policyStorage := policyStore.New(s.db)
	s.datastore = New(policyStorage, s.mockClusterDS, s.mockNotifierDS, s.categoryDS, edgeDS)

	s.mockCategoryDS = policyCategoryMocks.NewMockDataStore(gomock.NewController(s.T()))
	s.datastoreWithMockCategoryDS = New(policyStorage, s.mockClusterDS, s.mockNotifierDS, s.mockCategoryDS, edgeDS)
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

func (s *PolicyPostgresDataStoreTestSuite) TestGetAllPoliciesReturnsFilledCategories() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.WorkflowAdministration, resources.Cluster),
	))

	testCases := []struct {
		name               string
		categories         []string
		expectedCategories []string // Expected after title case normalization
	}{
		{
			name:               "Policy With Multiple Categories",
			categories:         []string{"DevOps Best Practices", "Security Best Practices"},
			expectedCategories: []string{"DevOps Best Practices", "Security Best Practices"},
		},
		{
			name:               "Policy With Single Category",
			categories:         []string{"Anomalous Activity"},
			expectedCategories: []string{"Anomalous Activity"},
		},
		{
			name:               "Test Policy With No Categories",
			categories:         []string{},
			expectedCategories: []string{},
		},
	}

	// Create input polcies from testCases
	var inputPolcies []*storage.Policy
	for _, tc := range testCases {
		policy := fixtures.GetPolicy()
		policy.Id = "" // Clear ID so AddPolicy generates a new one
		policy.Name = tc.name
		policy.Categories = tc.categories

		id, err := s.datastore.AddPolicy(ctx, policy)
		s.NoError(err)
		policy.Id = id
		inputPolcies = append(inputPolcies, policy)
	}

	outputPolicies, err := s.datastore.GetAllPolicies(ctx)
	s.NoError(err)
	s.Require().Len(outputPolicies, len(testCases))

	// Build a map by policy ID for easier verification
	outputPoliciesMap := make(map[string]*storage.Policy)
	for _, p := range outputPolicies {
		outputPoliciesMap[p.GetId()] = p
	}

	// Verify each policy has the expected categories filled
	for i, tc := range testCases {
		policy := inputPolcies[i]
		retrievedPolicy := outputPoliciesMap[policy.GetId()]
		s.Require().NotNil(retrievedPolicy, "policy %q should be in GetAllPolicies result", tc.name)
		s.Require().Len(retrievedPolicy.GetCategories(), len(tc.expectedCategories), "policy %q should have %d categories", tc.name, len(tc.expectedCategories))

		for _, expectedCategory := range tc.expectedCategories {
			s.Contains(retrievedPolicy.GetCategories(), expectedCategory, "policy %q should contain category %q", tc.name, expectedCategory)
		}
	}

	// Cleanup
	for _, policy := range inputPolcies {
		s.NoError(s.datastore.RemovePolicy(ctx, policy))
	}
}
