//go:build sql_integration

package m212tom213

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_212_to_m_213_add_container_start_column_to_indicators/schema"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_212_to_m_213_add_container_start_column_to_indicators/test/schema"
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

func (s *migrationTestSuite) TestMigration() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	// Create the old schema for testing
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableProcessIndicatorsStmt)

	// Add some process indicators
	numIndicators := 3000
	numNilContainerTime := 10
	var convertedProcessIndicators []oldSchema.ProcessIndicators
	for i := 0; i < numIndicators; i++ {
		processIndicator := &storage.ProcessIndicator{}
		s.NoError(testutils.FullInit(processIndicator, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

		// Since we are skipping records that have a nil time we need to create some to ensure that code executes properly
		if i < numNilContainerTime {
			processIndicator.ContainerStartTime = nil
		}

		converted, err := oldSchema.ConvertProcessIndicatorFromProto(processIndicator)
		s.Require().NoError(err)
		convertedProcessIndicators = append(convertedProcessIndicators, *converted)
	}

	s.Require().NoError(dbs.GormDB.Create(convertedProcessIndicators).Error)

	// Apply the new schema to then ensure time field is empty
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, updatedSchema.CreateTableProcessIndicatorsStmt)

	var n int
	err := s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM process_indicators WHERE containerstarttime IS NULL;").Scan(&n)
	s.NoError(err)
	s.Require().Equal(numIndicators, n)

	// Now run the migration
	s.Require().NoError(migration.Run(dbs))

	// After the migration, timestamp should only be NULL for indicators that had a null container time in the serialized object.
	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM process_indicators WHERE containerstarttime IS NULL;").Scan(&n)
	s.NoError(err)
	s.Require().Equal(numNilContainerTime, n)
}
