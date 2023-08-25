//go:build sql_integration

package m179tom180

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	policyPostgresStore "github.com/stackrox/rox/migrator/migrations/m_179_to_m_180_openshift_policy_exclusions/postgres"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type categoriesMigrationTestSuite struct {
	suite.Suite

	db          *pghelper.TestPostgres
	policyStore policyPostgresStore.Store
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(categoriesMigrationTestSuite))
}

func (s *categoriesMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), false)
	s.policyStore = policyPostgresStore.New(s.db.DB)
	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), frozenSchema.CreateTablePoliciesStmt)

}

func (s *categoriesMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *categoriesMigrationTestSuite) TestMigration() {
	ctx := sac.WithAllAccess(context.Background())
	testPolicy := fixtures.GetPolicy()
	testPolicy.Id = "ed8c7957-14de-40bc-aeab-d27ceeecfa7b"
	testPolicy.Name = "Iptables Executed in Privileged Container"
	testPolicy.Description = "Alert on privileged pods that execute iptables"
	testPolicy.PolicySections = []*storage.PolicySection{
		{
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: "Privileged Container",
					Values: []*storage.PolicyValue{
						{
							Value: "true",
						},
					},
				},
				{
					FieldName: "Process Name",
					Values: []*storage.PolicyValue{
						{
							Value: "iptables",
						},
					},
				},
				{
					FieldName: "Process UID",
					Values: []*storage.PolicyValue{
						{
							Value: "0",
						},
					},
				},
			},
		},
	}
	require.NoError(s.T(), s.policyStore.Upsert(ctx, testPolicy))
	// insert other policies in db for migration to run successfully
	policies := []string{
		"fb8f8732-c31d-496b-8fb1-d5abe6056e27",
		"880fd131-46f0-43d2-82c9-547f5aa7e043",
		"47cb9e0a-879a-417b-9a8f-de644d7c8a77",
		"6226d4ad-7619-4a0b-a160-46373cfcee66",
		"436811e7-892f-4da6-a0f5-8cc459f1b954",
		"742e0361-bddd-4a2d-8758-f2af6197f61d",
		"16c95922-08c4-41b6-a721-dc4b2a806632",
		"fe9de18b-86db-44d5-a7c4-74173ccffe2e",
		"dce17697-1b72-49d2-b18a-05d893cd9368",
		"f4996314-c3d7-4553-803b-b24ce7febe48",
		"a9b9ecf7-9707-4e32-8b62-d03018ed454f",
		"32d770b9-c6ba-4398-b48a-0c3e807644ed",
		"f95ff08d-130a-465a-a27e-32ed1fb05555",
	}

	policyName := "policy description %d"
	for i := 0; i < len(policies); i++ {
		require.NoError(s.T(), s.policyStore.Upsert(ctx, &storage.Policy{
			Id:   policies[i],
			Name: fmt.Sprintf(policyName, i),
		}))
	}
	dbs := &types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.db.GetGormDB(),
	}

	q := search.NewQueryBuilder().AddExactMatches(search.PolicyID, testPolicy.GetId()).ProtoQuery()
	policyPremigration, err := s.policyStore.GetByQuery(ctx, q)
	s.NoError(err)
	s.Empty(policyPremigration[0].Exclusions)
	s.NoError(migration.Run(dbs))
	expectedExclusions := []string{"Don't alert on ovnkube-node deployment in openshift-ovn-kubernetes Namespace",
		"Don't alert on haproxy-* deployment in openshift-vsphere-infra namespace",
		"Don't alert on keepalived-* deployment in openshift-vsphere-infra namespace",
		"Don't alert on coredns-* deployment in openshift-vsphere-infra namespace"}
	query := search.NewQueryBuilder().AddExactMatches(search.PolicyID, testPolicy.GetId()).ProtoQuery()
	policy, err := s.policyStore.GetByQuery(ctx, query)
	s.NoError(err)
	var actualExclusions []string
	for _, excl := range policy[0].Exclusions {
		actualExclusions = append(actualExclusions, excl.Name)
	}
	s.ElementsMatch(actualExclusions, expectedExclusions, "exclusion do not match after migration")

}
