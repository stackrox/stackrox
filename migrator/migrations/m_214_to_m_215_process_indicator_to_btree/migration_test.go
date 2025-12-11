//go:build sql_integration

package m214tom215

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_212_to_m_213_add_container_start_column_to_indicators/test/schema"
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
	var indicators []*storage.ProcessIndicator
	numIndicators := 3000
	for i := 0; i < numIndicators; i++ {
		processIndicator := &storage.ProcessIndicator{}
		s.NoError(testutils.FullInit(processIndicator, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		indicators = append(indicators, processIndicator)
	}

	var convertedProcessIndicators []oldSchema.ProcessIndicators
	for _, processIndicator := range indicators {
		// spreading these across some deployments to set up search test
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
	row := s.db.QueryRow(s.ctx, hashIndexQuery, tableName, deploymentIndex)
	var exists bool
	s.Require().NoError(row.Scan(&exists))
	s.Require().True(exists)

	row = s.db.QueryRow(s.ctx, hashIndexQuery, tableName, deploymentIndex)
	s.Require().NoError(row.Scan(&exists))
	s.Require().True(exists)

	s.Require().NoError(migration.Run(dbs))

	// Verfiy hash indexes no longer exist.
	row = s.db.QueryRow(s.ctx, hashIndexQuery, tableName, deploymentIndex)
	s.Require().NoError(row.Scan(&exists))
	s.Require().False(exists)

	row = s.db.QueryRow(s.ctx, hashIndexQuery, tableName, deploymentIndex)
	s.Require().NoError(row.Scan(&exists))
	s.Require().False(exists)

	// Verify btree indexes
	btreeIndexQuery := `SELECT EXISTS(
	SELECT tab.relname, cls.relname, am.amname
	FROM pg_index idx
	JOIN pg_class cls ON cls.oid=idx.indexrelid
	JOIN pg_class tab ON tab.oid=idx.indrelid
	JOIN pg_am am ON am.oid=cls.relam
	where tab.relname = $1 AND
	am.amname = 'btree' AND cls.relname = $2
	)`
	row = s.db.QueryRow(s.ctx, btreeIndexQuery, tableName, deploymentIndex)
	s.Require().NoError(row.Scan(&exists))
	s.Require().True(exists)

	row = s.db.QueryRow(s.ctx, btreeIndexQuery, tableName, deploymentIndex)
	s.Require().NoError(row.Scan(&exists))
	s.Require().True(exists)

}
