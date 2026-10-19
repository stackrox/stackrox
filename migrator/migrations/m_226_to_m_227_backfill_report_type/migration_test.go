//go:build sql_integration

package m226tom227

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_226_to_m_227_backfill_report_type/schema"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_226_to_m_227_backfill_report_type/test/schema"
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

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) databases() *types.Databases {
	return &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}
}

func (s *migrationTestSuite) setupPreMigrationTable() {
	_, err := s.db.DB.Exec(s.ctx, "DROP TABLE IF EXISTS report_snapshots")
	s.Require().NoError(err)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), oldSchema.CreateTableReportSnapshotsStmt)
}

func (s *migrationTestSuite) insertOldSnapshot(name string) {
	snapshot := oldSchema.ReportSnapshots{
		ReportID:              uuid.NewV4().String(),
		ReportConfigurationID: uuid.NewV4().String(),
		Name:                  name,
		Serialized:            []byte("{}"),
	}
	s.Require().NoError(s.db.GetGormDB().Create(&snapshot).Error)
}

func (s *migrationTestSuite) typeColumnExists() bool {
	var exists bool
	err := s.db.DB.QueryRow(s.ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'report_snapshots' AND column_name = 'type'
		)
	`).Scan(&exists)
	s.Require().NoError(err)
	return exists
}

func (s *migrationTestSuite) countWhere(where string) int {
	var count int
	s.Require().NoError(s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM report_snapshots WHERE "+where).Scan(&count))
	return count
}

func (s *migrationTestSuite) TestMigration() {
	s.setupPreMigrationTable()
	dbs := s.databases()

	const numReports = 3
	for i := range numReports {
		s.insertOldSnapshot(fmt.Sprintf("Test Report %d", i+1))
	}

	s.Require().False(s.typeColumnExists(), "Type column should not exist before migration")

	s.Require().NoError(migration.Run(dbs))

	s.Require().True(s.typeColumnExists(), "Type column should exist after migration")
	s.Require().Equal(numReports, s.countWhere("type = 0"), "All reports should have type=0 after migration")
	s.Require().Equal(0, s.countWhere("type IS NULL"), "No reports should have NULL type after migration")

	s.Require().NoError(migration.Run(dbs))
	s.Require().Equal(numReports, s.countWhere("type = 0"), "Count should remain the same after idempotent run")
}

func (s *migrationTestSuite) TestMigrationWithMixedData() {
	s.setupPreMigrationTable()
	dbs := s.databases()

	s.insertOldSnapshot("Pre-column Report")

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), updatedSchema.CreateTableReportSnapshotsStmt)

	vulnerabilityType := int32(storage.ReportSnapshot_VULNERABILITY)
	nodeVulnerabilityType := int32(storage.ReportSnapshot_NODE_VULNERABILITY)
	testData := []struct {
		name string
		typ  *int32
	}{
		{"Null Report 1", nil},
		{"Existing Report 1", &vulnerabilityType},
		{"Node Report 1", &nodeVulnerabilityType},
	}
	for _, td := range testData {
		if td.typ == nil {
			_, err := s.db.DB.Exec(s.ctx, `
				INSERT INTO report_snapshots (reportid, reportconfigurationid, name, serialized, type)
				VALUES ($1, $2, $3, $4, NULL)
			`, uuid.NewV4().String(), uuid.NewV4().String(), td.name, []byte("{}"))
			s.Require().NoError(err)
			continue
		}
		_, err := s.db.DB.Exec(s.ctx, `
			INSERT INTO report_snapshots (reportid, reportconfigurationid, name, serialized, type)
			VALUES ($1, $2, $3, $4, $5)
		`, uuid.NewV4().String(), uuid.NewV4().String(), td.name, []byte("{}"), *td.typ)
		s.Require().NoError(err)
	}

	s.Require().Equal(2, s.countWhere("type IS NULL"))
	s.Require().Equal(1, s.countWhere("type = 0"))
	s.Require().Equal(1, s.countWhere("type = 1"))

	s.Require().NoError(migration.Run(dbs))

	s.Require().Equal(0, s.countWhere("type IS NULL"), "No NULL values should remain")
	s.Require().Equal(3, s.countWhere("type = 0"), "NULL values should be backfilled to 0, existing 0 unchanged")
	s.Require().Equal(1, s.countWhere("type = 1"), "NODE_VULNERABILITY reports should remain unchanged")
}
