//go:build sql_integration

package m214tom215

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/indexhelper"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_214_to_m_215_process_indicator_to_btree/test/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
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

	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableProcessIndicatorsStmt)

	// Add some process indicators
	var convertedProcessIndicators []oldSchema.ProcessIndicators
	numIndicators := 3000
	for i := 0; i < numIndicators; i++ {
		processIndicator := &storage.ProcessIndicator{}
		s.Require().NoError(testutils.FullInit(processIndicator, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		processIndicator.Id = uuid.NewV4().String()

		converted, err := oldSchema.ConvertProcessIndicatorFromProto(processIndicator)
		s.Require().NoError(err)
		convertedProcessIndicators = append(convertedProcessIndicators, *converted)
	}

	if len(convertedProcessIndicators) > 0 {
		s.Require().NoError(dbs.GormDB.CreateInBatches(convertedProcessIndicators, numIndicators).Error)
	}
	log.Info("Created the indicators")

	// Verify hash indexes
	exists, err := indexhelper.IndexExists(s.ctx, s.db, tableName, deploymentIndex, "hash")
	s.Require().NoError(err)
	s.Require().True(exists)

	exists, err = indexhelper.IndexExists(s.ctx, s.db, tableName, podIndex, "hash")
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
	exists, err := indexhelper.IndexExists(s.ctx, s.db, tableName, deploymentIndex, "hash")
	s.Assert().NoError(err)
	s.Assert().False(exists)

	exists, err = indexhelper.IndexExists(s.ctx, s.db, tableName, podIndex, "hash")
	s.Assert().NoError(err)
	s.Assert().False(exists)

	exists, err = indexhelper.IndexExists(s.ctx, s.db, tableName, deploymentIndex, "btree")
	s.Assert().NoError(err)
	s.Assert().True(exists)

	exists, err = indexhelper.IndexExists(s.ctx, s.db, tableName, podIndex, "btree")
	s.Assert().NoError(err)
	s.Assert().True(exists)
}
