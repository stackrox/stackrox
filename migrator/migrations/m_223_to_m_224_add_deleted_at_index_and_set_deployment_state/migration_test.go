//go:build sql_integration

package m223tom224

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_223_to_m_224_add_deleted_at_index_and_set_deployment_state/test/schema"
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
	deploymentIDs := make([]string, numDeployments)
	for i := range numDeployments {
		id := uuid.NewV4().String()
		deploymentIDs[i] = id

		dep := &storage.Deployment{Id: id, Name: "test-deployment"}
		serialized, err := dep.MarshalVT()
		s.Require().NoError(err)

		_, err = db.Exec(s.ctx,
			"INSERT INTO deployments (id, name, hash, type, namespace, namespaceid, orchestratorcomponent, created, clusterid, clustername, priority, serviceaccount, serviceaccountpermissionlevel, riskscore, platformcomponent, serialized) VALUES ($1, $2, 0, 'Deployment', 'default', $3, false, now(), $4, 'test-cluster', 0, 'default', 0, 0, false, $5)",
			id, dep.GetName(), uuid.NewV4().String(), uuid.NewV4().String(), serialized,
		)
		s.Require().NoError(err)
	}

	// Run migration to add deleted and state columns and backfill state.
	s.Require().NoError(migration.Run(dbs))

	// Verify all deployments now have state = 1 (STATE_ACTIVE).
	var activeCount int
	err := db.QueryRow(s.ctx, "SELECT COUNT(*) FROM deployments WHERE state = 1").Scan(&activeCount)
	s.Require().NoError(err)
	s.Equal(numDeployments, activeCount)

	// Verify no deployments have NULL state or state = 0 (STATE_UNSPECIFIED) after migration.
	var unspecifiedCount int
	err = db.QueryRow(s.ctx, "SELECT COUNT(*) FROM deployments WHERE state IS NULL OR state = 0").Scan(&unspecifiedCount)
	s.Require().NoError(err)
	s.Equal(0, unspecifiedCount)

	// Verify index exists on deleted.
	var indexExists bool
	err = db.QueryRow(s.ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE tablename = 'deployments' AND indexname = 'deployments_deleted')").Scan(&indexExists)
	s.Require().NoError(err)
	s.True(indexExists, "index deployments_deleted should exist")

	// Run migration again to verify idempotency.
	s.Require().NoError(migration.Run(dbs))

	// Verify state is still STATE_ACTIVE after second run.
	err = db.QueryRow(s.ctx, "SELECT COUNT(*) FROM deployments WHERE state = 1").Scan(&activeCount)
	s.Require().NoError(err)
	s.Equal(numDeployments, activeCount)
}
