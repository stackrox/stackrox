//go:build sql_integration

package m218tom219

import (
	"context"
	"slices"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/indexhelper"
	"github.com/stackrox/rox/migrator/migrations/m_218_to_m_219_plop_index_to_btree/test/schema"
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

	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, schema.CreateTableListeningEndpointsStmt)

	// Add some PLOPs
	var convertedPLOPs []schema.ListeningEndpoints
	numPLOPs := 3000
	batchSize := 50
	for i := 0; i < numPLOPs; i++ {
		plop := &storage.ProcessListeningOnPortStorage{}
		s.Require().NoError(testutils.FullInit(plop, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		plop.Id = uuid.NewV4().String()

		converted, err := schema.ConvertProcessListeningOnPortStorageFromProto(plop)
		s.Require().NoError(err)
		convertedPLOPs = append(convertedPLOPs, *converted)
	}

	for plopBatch := range slices.Chunk(convertedPLOPs, batchSize) {
		s.Require().NoError(dbs.GormDB.CreateInBatches(plopBatch, batchSize).Error)
	}
	log.Info("Created the PLOPs")

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
