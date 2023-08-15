//go:build sql_integration

package m188tom189

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	policyPostgresStore "github.com/stackrox/rox/migrator/migrations/m_188_to_m_189_test_generic_example/policy/store"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type policyMigrationTestSuite struct {
	suite.Suite

	db          *pghelper.TestPostgres
	policyStore policyPostgresStore.Store

	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(policyMigrationTestSuite))
}

func simplePolicy(policyID string) *storage.Policy {
	return &storage.Policy{
		Id:   policyID,
		Name: fmt.Sprintf("Policy with id %s", policyID),
	}
}

func (s *policyMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), false)
	s.policyStore = policyPostgresStore.New(s.db.DB)
	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), frozenSchema.CreateTablePoliciesStmt)

	s.ctx = sac.WithAllAccess(context.Background())

	// insert other un policies that won't be migrated in the db for migration to run successfully
	policies := []*storage.Policy{
		simplePolicy("880fd131-46f0-43d2-82c9-547f5aa7e043"),
		simplePolicy("47cb9e0a-879a-417b-9a8f-de644d7c8a77"),
		simplePolicy("6226d4ad-7619-4a0b-a160-46373cfcee66"),
		simplePolicy("436811e7-892f-4da6-a0f5-8cc459f1b954"),
		simplePolicy("742e0361-bddd-4a2d-8758-f2af6197f61d"),
		simplePolicy("16c95922-08c4-41b6-a721-dc4b2a806632"),
		simplePolicy("a9b9ecf7-9707-4e32-8b62-d03018ed454f"),
		simplePolicy("32d770b9-c6ba-4398-b48a-0c3e807644ed"),
	}

	s.NoError(s.policyStore.UpsertMany(s.ctx, policies))
}

func (s *policyMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

// TestPolicyDescriptionMigration tests that at least one of the policies that needs to have its description
// updated does indeed get successfully get migrated
func (s *policyMigrationTestSuite) TestPolicyDescriptionMigration() {

	testPolicy := fixtures.GetPolicy()
	testPolicy.Id = "80267b36-2182-4fb3-8b53-e80c031f4ad8"
	testPolicy.Name = "ADD Command used instead of COPY"
	testPolicy.Description = "Alert on deployments using a ADD command"
	testPolicy.PolicySections = []*storage.PolicySection{
		{
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: "Dockerfile Line",
					Values: []*storage.PolicyValue{
						{
							Value: "ADD=.*",
						},
					},
				},
			},
		},
	}
	require.NoError(s.T(), s.policyStore.Upsert(s.ctx, testPolicy))

	s.NoError(migration.Run(&types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.db.GetGormDB(),
	}))

	expectedDescription := "Alert on deployments using an ADD command"
	policy, exists, err := s.policyStore.Get(s.ctx, testPolicy.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(expectedDescription, policy.GetDescription(), "description doesn't match after migration")
}

// TestPolicyExclusionMigration tests that at least one of the policies that needs to have an exclusion added
// does indeed get successfully get migrated
func (s *policyMigrationTestSuite) TestPolicyExclusionMigration() {
	testPolicy := fixtures.GetPolicy()
	testPolicy.Id = "6abcaa13-9ed6-4109-a1a7-be2e8280e49e"
	testPolicy.Name = "Docker CIS 5.7: Ensure privileged ports are not mapped within containers"
	testPolicy.Description = "The TCP/IP port numbers below 1024 are considered privileged ports. Normal users and processes are not allowed to use them for various security reasons. Containers are, however, allowed to map their ports to privileged ports."
	testPolicy.PolicySections = []*storage.PolicySection{
		{
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName:       "Exposed Node Port",
					BooleanOperator: storage.BooleanOperator_AND,
					Values: []*storage.PolicyValue{
						{
							Value: "<= 1024",
						},
						{
							Value: "> 0",
						},
					},
				},
			},
		},
	}
	require.NoError(s.T(), s.policyStore.Upsert(s.ctx, testPolicy))

	s.NoError(migration.Run(&types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.db.GetGormDB(),
	}))

	policy, exists, err := s.policyStore.Get(s.ctx, testPolicy.GetId())
	s.True(exists)
	s.NoError(err)
	expectedExclusions := []*storage.Exclusion{
		{
			Name: "Don't alert on the router-default deployment in namespace openshift-ingress",
			Deployment: &storage.Exclusion_Deployment{
				Name: "router-default",
				Scope: &storage.Scope{
					Namespace: "openshift-ingress",
				},
			},
		},
	}
	s.ElementsMatch(policy.Exclusions, expectedExclusions, "exclusion do not match after migration")
}
