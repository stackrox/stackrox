//go:build sql_integration

package m215tom216

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/indexhelper"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_215_to_m_216_process_baseline_to_btree/test/schema"
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

	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableProcessBaselinesStmt)

	// Add some process baselines
	var convertedProcessBaselines []oldSchema.ProcessBaselines
	numBaselines := 3000
	for i := 0; i < numBaselines; i++ {
		processBaseline := &storage.ProcessBaseline{}
		s.Require().NoError(testutils.FullInit(processBaseline, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		processBaseline.Id = uuid.NewV4().String()

		converted, err := oldSchema.ConvertProcessBaselineFromProto(processBaseline)
		s.Require().NoError(err)
		convertedProcessBaselines = append(convertedProcessBaselines, *converted)
	}

	if len(convertedProcessBaselines) > 0 {
		s.Require().NoError(dbs.GormDB.CreateInBatches(convertedProcessBaselines, numBaselines).Error)
	}
	log.Info("Created the baselines")

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
