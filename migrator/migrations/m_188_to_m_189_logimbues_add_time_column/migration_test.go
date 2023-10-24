//go:build sql_integration

package m188tom189

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_188_to_m_189_logimbues_add_time_column/schema"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_188_to_m_189_logimbues_add_time_column/test/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
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

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) TestMigration() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	// Create the old schema for testing
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableLogImbuesStmt)

	// Add some log imbues
	numImbues := 2000
	var convertedLogImbues []oldSchema.LogImbues
	for i := 0; i < numImbues; i++ {
		logImbue := &storage.LogImbue{}
		s.NoError(testutils.FullInit(logImbue, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		converted, err := oldSchema.ConvertLogImbueFromProto(logImbue)
		s.Require().NoError(err)
		convertedLogImbues = append(convertedLogImbues, *converted)
	}

	s.Require().NoError(dbs.GormDB.Create(convertedLogImbues).Error)

	// Apply the new schema to then ensure time field is empty
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, updatedSchema.CreateTableLogImbuesStmt)

	var n int
	err := s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM log_imbues WHERE timestamp IS NULL;").Scan(&n)
	s.NoError(err)
	s.Require().Equal(numImbues, n)

	// Now run the migration
	s.Require().NoError(migration.Run(dbs))

	// After the migration, timestamp should not be NULL
	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM log_imbues WHERE timestamp IS NULL;").Scan(&n)
	s.NoError(err)
	s.Require().Equal(0, n)
}
