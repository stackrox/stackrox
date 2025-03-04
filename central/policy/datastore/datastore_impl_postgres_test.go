//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/central/policy/search"
	policyStore "github.com/stackrox/rox/central/policy/store"
	pgStore "github.com/stackrox/rox/central/policy/store/postgres"
	policyCategoryDS "github.com/stackrox/rox/central/policycategory/datastore"
	categorySearch "github.com/stackrox/rox/central/policycategory/search"
	categoryPostgres "github.com/stackrox/rox/central/policycategory/store/postgres"
	policyCategoryEdgeDS "github.com/stackrox/rox/central/policycategoryedge/datastore"
	edgeSearch "github.com/stackrox/rox/central/policycategoryedge/search"
	edgePostgres "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
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
	"gorm.io/gorm"
)

func TestPolicyDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(PolicyPostgresDataStoreTestSuite))
}

type PolicyPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx            context.Context
	db             postgres.DB
	gormDB         *gorm.DB
	mockClusterDS  *clusterDSMocks.MockDataStore
	mockNotifierDS *notifierDSMocks.MockDataStore

	datastore  DataStore
	categoryDS policyCategoryDS.DataStore
}

func (s *PolicyPostgresDataStoreTestSuite) SetupSuite() {

	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)

	pool, err := postgres.New(s.ctx, config)
	s.NoError(err)
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	s.db = pool
}

func (s *PolicyPostgresDataStoreTestSuite) SetupTest() {
	pgStore.Destroy(s.ctx, s.db)
	categoryPostgres.Destroy(s.ctx, s.db)
	edgePostgres.Destroy(s.ctx, s.db)

	s.mockClusterDS = clusterDSMocks.NewMockDataStore(gomock.NewController(s.T()))
	s.mockNotifierDS = notifierDSMocks.NewMockDataStore(gomock.NewController(s.T()))

	categoryStorage := categoryPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	categorySearcher := categorySearch.New(categoryStorage)

	edgeStorage := edgePostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	edgeSearcher := edgeSearch.New(edgeStorage)

	s.categoryDS = policyCategoryDS.New(categoryStorage, categorySearcher, policyCategoryEdgeDS.New(edgeStorage, edgeSearcher))

	policyDS := policyStore.New(s.db)
	s.datastore = New(policyDS, search.New(policyDS), s.mockClusterDS, s.mockNotifierDS, s.categoryDS)

}

func (s *PolicyPostgresDataStoreTestSuite) TearDownSuite() {
	s.db.Close()
	pgtest.CloseGormDB(s.T(), s.gormDB)
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
	s.NoError(s.datastore.RemovePolicy(ctx, policy.GetId()))
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
	s.NoError(s.datastore.RemovePolicy(ctx, policy.GetId()))
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
			s.Require().Len(responses[0].Errors, 1)
			s.Require().Equal(responses[0].Errors[0].Type, c.expectedImportError)

			// Now try to import with overwrite true
			responses, allSucceeded, err = s.datastore.ImportPolicies(ctx, []*storage.Policy{c.newPolicy}, true)

			if c.failOnOverwrite {
				s.Require().NoError(err) // It's not an error just a failure?
				s.Require().False(allSucceeded)
				s.Require().Len(responses, 1)
				s.Require().Len(responses[0].Errors, 1)
				s.Require().Equal(responses[0].Errors[0].Type, c.expectedImportError) // ... should the error be different?

				// Find the existing policy and validate the name and id
				result, _, err := s.datastore.GetPolicy(ctx, c.existingPolicy.GetId())
				s.NoError(err)
				s.Equal(c.existingPolicy.GetName(), result.GetName())

				// Delete the policy
				s.NoError(s.datastore.RemovePolicy(ctx, c.existingPolicy.GetId()))
			} else {
				s.NoError(err) // It's not an error just a failure?
				s.True(allSucceeded)
				s.Require().Len(responses, 1)
				s.Empty(responses[0].Errors)

				// Find the new policy and validate the name and id
				result, _, err := s.datastore.GetPolicy(ctx, c.newPolicy.GetId())
				s.NoError(err)
				s.Equal(c.newPolicy.GetName(), result.GetName())

				// Delete the policy
				s.NoError(s.datastore.RemovePolicy(ctx, c.newPolicy.GetId()))
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
	s.Len(policies[0].Categories, 3)
}
