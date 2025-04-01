//go:build sql_integration

package policymigrationhelper

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	categorypostgresstore "github.com/stackrox/rox/migrator/migrations/policymigrationhelper/categorypostgresstorefortest"
	categorySchema "github.com/stackrox/rox/migrator/migrations/policymigrationhelper/categorypostgresstorefortest/schema"
	edgeypostgresstore "github.com/stackrox/rox/migrator/migrations/policymigrationhelper/edgepostgresstorefortest"
	edgeSchema "github.com/stackrox/rox/migrator/migrations/policymigrationhelper/edgepostgresstorefortest/schema"
	policypostgresstore "github.com/stackrox/rox/migrator/migrations/policymigrationhelper/policypostgresstorefortest"
	policySchema "github.com/stackrox/rox/migrator/migrations/policymigrationhelper/policypostgresstorefortest/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

// The tests in here are a subset of what exists in policy_migrator_test.go
// The ones that were copied over have been updated to work with postgres.

var (
	policyID = "0000-0000-0000-0000"
)

func TestPostgresPolicyMigrator(t *testing.T) {
	suite.Run(t, new(postgresPolicyMigratorTestSuite))
}

type postgresPolicyMigratorTestSuite struct {
	suite.Suite
	ctx           context.Context
	db            *pghelper.TestPostgres
	store         policypostgresstore.Store
	categoryStore categorypostgresstore.Store
	edgeStore     edgeypostgresstore.Store
}

func (s *postgresPolicyMigratorTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), false)
	s.ctx = sac.WithAllAccess(context.Background())
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), policySchema.CreateTablePoliciesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), categorySchema.CreateTablePolicyCategoriesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), edgeSchema.CreateTablePolicyCategoryEdgesStmt)

	s.store = policypostgresstore.New(s.db, s.T())
	s.categoryStore = categorypostgresstore.New(s.db, s.T())
	s.edgeStore = edgeypostgresstore.New(s.db, s.T())

}

func (s *postgresPolicyMigratorTestSuite) comparePolicyWithDB(policyID string, policy *storage.Policy) {
	newPolicy, exists, err := s.store.Get(s.ctx, policyID)
	s.NoError(err)
	s.True(exists)

	policy.Categories = nil
	protoassert.Equal(s.T(), policy, newPolicy)
}

// TODO: Remove once the deprecated functions are removed
func (s *postgresPolicyMigratorTestSuite) getTestCaseFunctions() map[string]func(map[string]PolicyChanges, map[string]*storage.Policy) error {
	return map[string]func(map[string]PolicyChanges, map[string]*storage.Policy) error{
		"MigratePoliciesWithStore": func(policiesToMigrate map[string]PolicyChanges, comparisonPolicies map[string]*storage.Policy) error {
			return MigratePoliciesWithStore(policiesToMigrate, comparisonPolicies,
				s.store.Exists, s.store.Get, s.store.Upsert)
		},
		"MigratePoliciesWithStoreV2": func(policiesToMigrate map[string]PolicyChanges, comparisonPolicies map[string]*storage.Policy) error {
			return MigratePoliciesWithStoreV2(policiesToMigrate, comparisonPolicies, s.store.Get, s.store.Upsert,
				s.getAllCategories,
				s.edgeStore.Upsert,
				s.deleteEdge)
		},
	}
}

// Test that unrelated policies aren't updated
func (s *postgresPolicyMigratorTestSuite) TestUnrelatedPolicyIsNotUpdated() {
	policyID := "this-is-a-random-id-that-should-not-exist"
	policy := testPolicy(policyID)

	policiesToMigrate := map[string]PolicyChanges{
		"0000-0000-0000-0000": {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("this is a new description")},
		},
	}

	comparisonPolicies := map[string]*storage.Policy{
		policyID: policy,
	}

	tests := s.getTestCaseFunctions()
	for tc, fn := range tests {
		s.T().Run(tc, func(t *testing.T) {
			s.NoError(s.store.Upsert(s.ctx, policy))
			s.NoError(fn(policiesToMigrate, comparisonPolicies))
			s.comparePolicyWithDB(policyID, policy)
		})
	}

}

// Test that an unmodified policy that matches comparison policy is updated
func (s *postgresPolicyMigratorTestSuite) TestUnmodifiedAndMatchingPolicyIsUpdated() {
	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("this is a new description")},
		},
	}

	tests := s.getTestCaseFunctions()
	for tc, fn := range tests {
		s.T().Run(tc, func(t *testing.T) {
			policy := testPolicy(policyID)

			comparisonPolicies := map[string]*storage.Policy{
				policyID: policy,
			}

			s.NoError(s.store.Upsert(s.ctx, policy))
			s.NoError(fn(policiesToMigrate, comparisonPolicies))

			// Policy should've had description changed, but nothing else
			policy.Description = *policiesToMigrate[policyID].ToChange.Description
			s.comparePolicyWithDB(policyID, policy)
		})
	}
}

// Test that all unmodified policies are updated
func (s *postgresPolicyMigratorTestSuite) TestAllUnmodifiedPoliciesGetUpdated() {
	policiesToTest := make([]*storage.Policy, 10)
	comparisonPolicies := make(map[string]*storage.Policy)
	policiesToMigrate := make(map[string]PolicyChanges)

	tests := s.getTestCaseFunctions()
	for tc, fn := range tests {
		s.T().Run(tc, func(t *testing.T) {
			// Create and insert a set of unmodified fake policies
			for i := 0; i < 10; i++ {
				policy := testPolicy(fmt.Sprintf("policy%d", i))
				policiesToTest[i] = policy
				policy.Name = fmt.Sprintf("policy-name%d", i) // name is a unique key
				policy.Description = "sfasdf"

				comparisonPolicy := policy.CloneVT()
				comparisonPolicies[policy.Id] = comparisonPolicy
				policiesToMigrate[policy.Id] = PolicyChanges{
					FieldsToCompare: []FieldComparator{PolicySectionComparator, ExclusionComparator, RemediationComparator, RationaleComparator},
					ToChange:        PolicyUpdates{Description: strPtr(fmt.Sprintf("%s new description", policy.Id))}, // give them all a new description
				}
			}

			s.NoError(s.store.UpsertMany(s.ctx, policiesToTest))
			s.NoError(fn(policiesToMigrate, comparisonPolicies))

			for _, policy := range policiesToTest {
				// All the policies should've changed
				policy.Description = fmt.Sprintf("%s new description", policy.Id)
				s.comparePolicyWithDB(policy.Id, policy)
			}
		})
	}
}

// Test that exclusions can get added and removed appropriately
func (s *postgresPolicyMigratorTestSuite) TestExclusionAreAddedAndRemovedAsNecessary() {
	policy := testPolicy(policyID)

	// Add a bunch of exclusions into the DB
	policy.Exclusions = []*storage.Exclusion{
		{Name: "exclusion0", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-0"}}},
		{Name: "exclusion1", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace 1"}}},
		{Name: "exclusion0", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-0"}}},
		{Name: "exclusion2", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-2"}}},
		{Name: "exclusion3", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-3"}}},
		{Name: "exclusion4", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-4"}}},
	}

	s.NoError(s.store.Upsert(s.ctx, policy))

	comparisonPolicies := map[string]*storage.Policy{
		policyID: policy,
	}

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{ExclusionComparator},
			ToChange: PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{
					{Name: "exclusion1-changed", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-1"}}},
					{Name: "NEW exclusion", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-NEW"}}},
					{Name: "NEW exclusion2", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-NEW2"}}},
				},
				ExclusionsToRemove: []*storage.Exclusion{
					{Name: "exclusion1", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace 1"}}},
					{Name: "exclusion4", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-4"}}},
					{Name: "exclusion-NaN", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-NaN"}}}, // this exclusion doesn't exist so it shouldn't get removed
				},
			},
		},
	}

	s.NoError(MigratePoliciesWithStoreV2(
		policiesToMigrate,
		comparisonPolicies,
		s.store.Get,
		s.store.Upsert,
		s.getAllCategories,
		s.edgeStore.Upsert,
		s.deleteEdge,
	))

	// Policy exclusions should be updated
	policy.Exclusions = []*storage.Exclusion{
		{Name: "exclusion0", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-0"}}},
		{Name: "exclusion0", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-0"}}},
		{Name: "exclusion2", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-2"}}},
		{Name: "exclusion3", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-3"}}},
		{Name: "exclusion1-changed", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-1"}}},
		{Name: "NEW exclusion", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-NEW"}}},
		{Name: "NEW exclusion2", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-NEW2"}}},
	}

	s.comparePolicyWithDB(policyID, policy)
}

func testPolicy(id string) *storage.Policy {
	return &storage.Policy{
		Id:          id,
		Name:        "name",
		Remediation: "remediation",
		Rationale:   "rationale",
		Description: "description",
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{FieldName: "Process Name", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "iptables"}}},
				},
			},
		},
		Exclusions: []*storage.Exclusion{
			{Name: "exclusion name", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace name"}}},
		},
	}
}

// Test that categories can get added and removed appropriately
func (s *postgresPolicyMigratorTestSuite) TestCategoriesAreAddedAndRemovedAsNecessary() {
	policy := testPolicy(policyID)

	// Add a bunch of exclusions into the DB
	policy.Categories = []string{"cat-1", "cat-2", "my-cat", "cat-2", "cat-4", "duped-category"}

	s.NoError(s.store.Upsert(s.ctx, policy))

	for _, c := range policy.Categories {
		s.NoError(s.categoryStore.Upsert(s.ctx, &storage.PolicyCategory{
			Id:        c,
			Name:      c,
			IsDefault: true,
		}))
	}
	s.NoError(s.categoryStore.Upsert(s.ctx, &storage.PolicyCategory{
		Id:        "category-2-changed",
		Name:      "category-2-changed",
		IsDefault: true,
	}))
	s.NoError(s.categoryStore.Upsert(s.ctx, &storage.PolicyCategory{
		Id:        "category-5",
		Name:      "category-5",
		IsDefault: true,
	}))

	comparisonPolicies := map[string]*storage.Policy{
		policyID: policy,
	}

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{PolicySectionComparator},
			ToChange: PolicyUpdates{
				CategoriesToAdd:    []string{"category-2-changed", "category-5", "duped-category"},
				CategoriesToRemove: []string{"cat-2", "cat-4", "i-dont-exist"},
			},
		},
	}

	s.NoError(MigratePoliciesWithStoreV2(
		policiesToMigrate,
		comparisonPolicies,
		s.store.Get,
		s.store.Upsert,
		s.getAllCategories,
		s.edgeStore.Upsert,
		s.deleteEdge,
	))

	// Policy categories should be updated
	policy.Categories = []string{"cat-1", "my-cat", "cat-2", "duped-category", "category-2-changed", "category-5"}
	s.comparePolicyWithDB(policyID, policy)
}

// Test that exclusions are added if the policy never had any before
func (s *postgresPolicyMigratorTestSuite) TestCategoriesAreAddedEvenIfPolicyHadNoneBefore() {
	policy := testPolicy(policyID)

	// Remove all categories to start with
	policy.Categories = nil
	s.NoError(s.store.Upsert(s.ctx, policy))
	s.NoError(s.categoryStore.Upsert(s.ctx, &storage.PolicyCategory{
		Id:        "category-added",
		Name:      "category-added",
		IsDefault: true,
	}))

	comparisonPolicies := map[string]*storage.Policy{
		policyID: policy,
	}

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{PolicySectionComparator},
			ToChange: PolicyUpdates{
				CategoriesToAdd: []string{"category-added"},
			},
		},
	}

	s.NoError(MigratePoliciesWithStoreV2(
		policiesToMigrate,
		comparisonPolicies,
		s.store.Get,
		s.store.Upsert,
		s.getAllCategories,
		s.edgeStore.Upsert,
		s.deleteEdge,
	))

	// Policy categories should be updated
	policy.Categories = []string{"category-added"}
	s.comparePolicyWithDB(policyID, policy)
}

func (s *postgresPolicyMigratorTestSuite) getAllCategories(ctx context.Context) (map[string]string, error) {
	categories, err := s.categoryStore.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	categoryMap := make(map[string]string, len(categories))
	for _, c := range categories {
		categoryMap[c.Name] = c.Id
	}
	return categoryMap, nil
}

func (s *postgresPolicyMigratorTestSuite) deleteEdge(ctx context.Context, edge *storage.PolicyCategoryEdge) error {
	_, err := s.edgeStore.DeleteByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.PolicyID, edge.GetPolicyId()).
		AddExactMatches(search.PolicyCategoryID, edge.GetCategoryId()).ProtoQuery())
	return err
}
