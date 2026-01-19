//go:build sql_integration

package m217tom218

import (
	"context"
	"slices"
	"testing"

	"github.com/stackrox/rox/migrator/migrations/indexhelper"
	"github.com/stackrox/rox/migrator/migrations/m_217_to_m_218_alert_to_btree/test/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
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
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, schema.CreateTableAlertsStmt)

	// Add some alerts
	var convertedAlerts []schema.Alerts
	numAlerts := 3000
	batchSize := 50
	for i := 0; i < numAlerts; i++ {
		id := uuid.NewV4().String()
		alert := fixtures.GetAlertWithID(id)
		alert.NamespaceId = uuid.NewV4().String()

		converted, err := schema.ConvertAlertFromProto(alert)
		s.Require().NoError(err)
		convertedAlerts = append(convertedAlerts, *converted)
	}

	for alertBatch := range slices.Chunk(convertedAlerts, batchSize) {
		s.Require().NoError(dbs.GormDB.CreateInBatches(alertBatch, batchSize).Error)
	}
	log.Info("Created the alerts")

	// Verify hash indexes
	exists, err := indexhelper.IndexExists(s.ctx, s.db, tableName, index, "hash")
	s.Require().NoError(err)
	s.Require().True(exists)

	s.Assert().NoError(migration.Run(dbs))

	s.verifyNewIndexes()

	// Run the migration a second time to ensure idempotentcy.
	s.Assert().NoError(migration.Run(dbs))

	s.verifyNewIndexes()
}

func (s *migrationTestSuite) verifyNewIndexes() {
	// Verify hash indexes no longer exist.
	exists, err := indexhelper.IndexExists(s.ctx, s.db, tableName, index, "hash")
	s.Assert().NoError(err)
	s.Assert().False(exists)

	exists, err = indexhelper.IndexExists(s.ctx, s.db, tableName, index, "btree")
	s.Assert().NoError(err)
	s.Assert().True(exists)
}
