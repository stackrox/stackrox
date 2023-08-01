//go:build sql_integration

package m168tom169

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v74"
	permissionSetPostgresStore "github.com/stackrox/rox/migrator/migrations/m_168_to_m_169_postgres_remove_clustercve_permission/permissionsetpostgresstore"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stretchr/testify/suite"
)

const (
	id0 = "A161527B-D34F-42B8-A783-23E39B4DE15A"
	id1 = "DC04A5F8-6018-46E5-B590-87325FBF1945"
	id2 = "9C91FA2B-AE95-4C74-98A7-17AF76CC8209"
	id3 = "ABBA7029-EAED-4FFD-8FB0-02CA9F2B6A21"
)

var (
	unmigratedPSs = []*storage.PermissionSet{
		{
			Id:   id0,
			Name: "ps0",
			ResourceToAccess: map[string]storage.Access{
				"ClusterCVE": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:   id1,
			Name: "ps1",
			ResourceToAccess: map[string]storage.Access{
				"Cluster":    storage.Access_READ_WRITE_ACCESS,
				"ClusterCVE": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	unmigratedPSsAfterMigration = []*storage.PermissionSet{
		{
			Id:   id0,
			Name: "ps0",
			ResourceToAccess: map[string]storage.Access{
				"Cluster": storage.Access_READ_ACCESS,
				"Image":   storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:   id1,
			Name: "ps1",
			ResourceToAccess: map[string]storage.Access{
				"Cluster": storage.Access_READ_ACCESS,
				"Image":   storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	alreadyMigratedPSs = []*storage.PermissionSet{
		{
			Id:               id2,
			Name:             "ps2",
			ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_WRITE_ACCESS},
		},
		{
			Id:               id3,
			Name:             "ps3",
			ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_WRITE_ACCESS},
		},
	}
)

type psMigrationTestSuite struct {
	suite.Suite

	db    *pghelper.TestPostgres
	store permissionSetPostgresStore.Store
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(psMigrationTestSuite))
}

func (s *psMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), false)
	s.store = permissionSetPostgresStore.New(s.db.DB)
	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), frozenSchema.CreateTablePermissionSetsStmt)
}

func (s *psMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *psMigrationTestSuite) TestMigration() {
	ctx := context.Background()
	var psToUpsert []*storage.PermissionSet
	psToUpsert = append(psToUpsert, unmigratedPSs...)
	psToUpsert = append(psToUpsert, alreadyMigratedPSs...)

	s.NoError(s.store.UpsertMany(ctx, psToUpsert))

	dbs := &types.Databases{
		PostgresDB: s.db.DB,
	}

	s.NoError(migration.Run(dbs))

	var allPSsAfterMigration []*storage.PermissionSet

	checkErr := s.store.Walk(ctx, func(obj *storage.PermissionSet) error {
		allPSsAfterMigration = append(allPSsAfterMigration, obj)
		return nil
	})
	s.NoError(checkErr)

	var expectedPSsAfterMigration []*storage.PermissionSet
	expectedPSsAfterMigration = append(expectedPSsAfterMigration, unmigratedPSsAfterMigration...)
	expectedPSsAfterMigration = append(expectedPSsAfterMigration, alreadyMigratedPSs...)

	s.ElementsMatch(expectedPSsAfterMigration, allPSsAfterMigration)
}

func (s *psMigrationTestSuite) TestMigrationOnCleanDB() {
	dbs := &types.Databases{
		PostgresDB: s.db.DB,
	}
	s.NoError(migration.Run(dbs))
}
