//go:build sql_integration

package n57ton58

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/n_57_to_n_58_postgres_remove_clustercve_permission/postgres"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stretchr/testify/suite"
)

var (
	unmigratedPSs = []*storage.PermissionSet{
		{
			Id:   "id0",
			Name: "ps0",
			ResourceToAccess: map[string]storage.Access{
				"ClusterCVE": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:   "id1",
			Name: "ps1",
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	unmigratedPSsAfterMigration = []*storage.PermissionSet{
		{
			Id:   "id0",
			Name: "ps0",
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:   "id1",
			Name: "ps1",
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	alreadyMigratedPSs = []*storage.PermissionSet{
		{
			Id:               "id2",
			Name:             "ps2",
			ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_WRITE_ACCESS},
		},
		{
			Id:               "id3",
			Name:             "ps3",
			ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_WRITE_ACCESS},
		},
	}
)

type psMigrationTestSuite struct {
	suite.Suite

	db    *pghelper.TestPostgres
	store postgres.Store
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(psMigrationTestSuite))
}

func (s *psMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), true)
	s.store = postgres.New(s.db.Pool)
	schema.ApplySchemaForTable(context.Background(), s.db.GetGormDB(), schema.PermissionSetsTableName)
}

func (s *psMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *psMigrationTestSuite) TestMigration() {
	ctx := context.Background()
	var psToUpsert []*storage.PermissionSet
	psToUpsert = append(psToUpsert, unmigratedPSs...)
	psToUpsert = append(psToUpsert, alreadyMigratedPSs...)

	for _, initial := range psToUpsert {
		s.NoError(s.store.Upsert(ctx, initial))
	}

	dbs := &types.Databases{
		PostgresDB: s.db.Pool,
	}

	s.NoError(migration.Run(dbs))

	var allPSsAfterMigration []*storage.PermissionSet

	s.store.Walk(ctx, func(obj *storage.PermissionSet) error {
		allPSsAfterMigration = append(allPSsAfterMigration, obj)
		return nil
	})

	var expectedPSsAfterMigration []*storage.PermissionSet
	expectedPSsAfterMigration = append(expectedPSsAfterMigration, unmigratedPSsAfterMigration...)
	expectedPSsAfterMigration = append(expectedPSsAfterMigration, alreadyMigratedPSs...)

	s.ElementsMatch(expectedPSsAfterMigration, allPSsAfterMigration)
}

func (s *psMigrationTestSuite) TestMigrationOnCleanDB() {
	dbs := &types.Databases{
		PostgresDB: s.db.Pool,
	}
	s.NoError(migration.Run(dbs))
}
