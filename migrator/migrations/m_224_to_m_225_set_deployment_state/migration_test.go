//go:build sql_integration

package m223tom224

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_224_to_m_225_set_deployment_state/test/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
}

func (s *migrationTestSuite) TestMigration() {
	db := s.db.DB
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: db,
		DBCtx:      s.ctx,
	}

	// Create the old schema (without deleted and state columns).
	pgutils.CreateTableFromModel(s.ctx, dbs.GormDB, oldSchema.CreateTableDeploymentsStmt)

	// Insert test deployments using the old schema (without deleted and state columns).
	numDeployments := 5
	for range numDeployments {
		id := uuid.NewV4().String()

		dep := &storage.Deployment{Id: id, Name: "test-deployment"}
		serialized, err := dep.MarshalVT()
		s.Require().NoError(err)

		_, err = db.Exec(s.ctx,
			"INSERT INTO deployments (id, name, hash, type, namespace, namespaceid, orchestratorcomponent, created, clusterid, clustername, priority, serviceaccount, serviceaccountpermissionlevel, riskscore, platformcomponent, serialized) VALUES ($1, $2, 0, 'Deployment', 'default', $3, false, now(), $4, 'test-cluster', 0, 'default', 0, 0, false, $5)",
			id, dep.GetName(), uuid.NewV4().String(), uuid.NewV4().String(), serialized,
		)
		s.Require().NoError(err)
	}

	// Verify the old schema has no state column, so existing rows cannot
	// have a state value yet. After GORM adds the column, these rows will
	// have state = NULL until the migration backfills them.
	s.Require().NoError(migration.Run(dbs))

	// Verify the backfill set state = 0 (DEPLOYMENT_STATE_ACTIVE) for all
	// existing rows that previously had NULL.
	var activeCount int
	err := db.QueryRow(s.ctx, "SELECT COUNT(*) FROM deployments WHERE state = 0").Scan(&activeCount)
	s.Require().NoError(err)
	s.Equal(numDeployments, activeCount)

	// Verify no rows still have NULL state after the migration.
	var nullStateCount int
	err = db.QueryRow(s.ctx, "SELECT COUNT(*) FROM deployments WHERE state IS NULL").Scan(&nullStateCount)
	s.Require().NoError(err)
	s.Equal(0, nullStateCount)

	// Verify the deleted column exists and is NULL for all rows.
	var deletedNullCount int
	err = db.QueryRow(s.ctx, "SELECT COUNT(*) FROM deployments WHERE deleted IS NULL").Scan(&deletedNullCount)
	s.Require().NoError(err)
	s.Equal(numDeployments, deletedNullCount)

	// Run migration again to verify idempotency.
	s.Require().NoError(migration.Run(dbs))
}
