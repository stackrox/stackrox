//go:build sql_integration

package m189tom190

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_189_to_m_190_plop_add_closed_time_column/schema"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_189_to_m_190_plop_add_closed_time_column/test/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/timestamp"
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
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableListeningEndpointsStmt)

	// Add some plops
	numPlop := 2000
	var convertedPlops []oldSchema.ListeningEndpoints
	for i := 0; i < numPlop; i++ {
		plop := &storage.ProcessListeningOnPortStorage{}
		s.NoError(testutils.FullInit(plop, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		plop.CloseTimestamp = timestamp.Now().GogoProtobuf()
		converted, err := oldSchema.ConvertProcessListeningOnPortStorageFromProto(plop)
		s.Require().NoError(err)
		convertedPlops = append(convertedPlops, *converted)
	}

	s.Require().NoError(dbs.GormDB.Create(convertedPlops).Error)

	// Apply the new schema to then ensure time field is empty
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, updatedSchema.CreateTableListeningEndpointsStmt)

	var n int
	err := s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM listening_endpoints WHERE closetimestamp IS NULL;").Scan(&n)
	s.NoError(err)
	s.Require().Equal(numPlop, n)

	// Now run the migration
	log.Infof("SHREWS start -- %v", time.Now())
	s.Require().NoError(migration.Run(dbs))
	log.Infof("SHREWS end -- %v", time.Now())

	// After the migration, timestamp should not be NULL
	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM listening_endpoints WHERE closetimestamp IS NULL;").Scan(&n)
	s.NoError(err)
	s.Require().Equal(0, n)

}
