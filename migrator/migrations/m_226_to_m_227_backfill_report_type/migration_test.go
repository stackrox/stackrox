//go:build sql_integration

package m226tom227

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
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

	// Create the table manually (simulating pre-migration state without type column)
	// The migration will add the type column
	_, err := s.db.DB.Exec(s.ctx, `
		CREATE TABLE IF NOT EXISTS report_snapshots (
			reportid UUID PRIMARY KEY,
			reportconfigurationid VARCHAR,
			name VARCHAR,
			serialized BYTEA
		)
	`)
	s.Require().NoError(err)
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

	// Insert test data without type column (simulating pre-migration state)
	testReportIDs := []string{
		uuid.NewV4().String(),
		uuid.NewV4().String(),
		uuid.NewV4().String(),
	}

	for i, reportID := range testReportIDs {
		_, err := s.db.DB.Exec(s.ctx, `
			INSERT INTO report_snapshots (reportid, reportconfigurationid, name, serialized)
			VALUES ($1, $2, $3, $4)
		`, reportID, uuid.NewV4().String(), "Test Report "+string(rune(i+1)), []byte("{}"))
		s.Require().NoError(err)
	}

	// Verify table has no type column yet
	var columnExists bool
	err := s.db.DB.QueryRow(s.ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'report_snapshots' AND column_name = 'type'
		)
	`).Scan(&columnExists)
	s.Require().NoError(err)
	s.Require().False(columnExists, "Type column should not exist before migration")

	// Run the migration
	s.Require().NoError(migration.Run(dbs))

	// Verify type column was added
	err = s.db.DB.QueryRow(s.ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'report_snapshots' AND column_name = 'type'
		)
	`).Scan(&columnExists)
	s.Require().NoError(err)
	s.Require().True(columnExists, "Type column should exist after migration")

	// Verify all existing reports have type=0 (VULNERABILITY)
	var backfilledCount int
	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM report_snapshots WHERE type = 0").Scan(&backfilledCount)
	s.Require().NoError(err)
	s.Require().Equal(len(testReportIDs), backfilledCount, "All reports should have type=0 after migration")

	// Verify no NULL values remain
	var nullCount int
	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM report_snapshots WHERE type IS NULL").Scan(&nullCount)
	s.Require().NoError(err)
	s.Require().Equal(0, nullCount, "No reports should have NULL type after migration")

	// Test idempotency: running again should not error and should not change anything
	s.Require().NoError(migration.Run(dbs))

	// Verify count is still correct after second run
	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM report_snapshots WHERE type = 0").Scan(&backfilledCount)
	s.Require().NoError(err)
	s.Require().Equal(len(testReportIDs), backfilledCount, "Count should remain the same after idempotent run")
}

func (s *migrationTestSuite) TestMigrationWithMixedData() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	// Clear previous test data
	_, err := s.db.DB.Exec(s.ctx, "DELETE FROM report_snapshots")
	s.Require().NoError(err)

	// Drop type column to simulate pre-migration state
	_, err = s.db.DB.Exec(s.ctx, "ALTER TABLE report_snapshots DROP COLUMN IF EXISTS type")
	s.Require().NoError(err)

	// Re-run migration to add type column
	s.Require().NoError(migration.Run(dbs))

	// Insert mixed data: some with NULL, some with existing values
	vulnerabilityType := int32(storage.ReportSnapshot_VULNERABILITY)
	nodeVulnerabilityType := int32(storage.ReportSnapshot_NODE_VULNERABILITY)

	testData := []struct {
		id   string
		name string
		typ  *int32
	}{
		{uuid.NewV4().String(), "Null Report 1", nil}, // NULL - should be backfilled
		{uuid.NewV4().String(), "Existing Report 1", &vulnerabilityType},
		{uuid.NewV4().String(), "Node Report 1", &nodeVulnerabilityType},
		{uuid.NewV4().String(), "Null Report 2", nil}, // NULL - should be backfilled
	}

	for _, td := range testData {
		if td.typ == nil {
			_, err := s.db.DB.Exec(s.ctx, `
				INSERT INTO report_snapshots (reportid, reportconfigurationid, name, serialized, type)
				VALUES ($1, $2, $3, $4, NULL)
			`, td.id, uuid.NewV4().String(), td.name, []byte("{}"))
			s.Require().NoError(err)
		} else {
			_, err := s.db.DB.Exec(s.ctx, `
				INSERT INTO report_snapshots (reportid, reportconfigurationid, name, serialized, type)
				VALUES ($1, $2, $3, $4, $5)
			`, td.id, uuid.NewV4().String(), td.name, []byte("{}"), *td.typ)
			s.Require().NoError(err)
		}
	}

	// Verify initial state
	var nullCount, vulnerabilityCount, nodeVulnerabilityCount int
	s.Require().NoError(s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM report_snapshots WHERE type IS NULL").Scan(&nullCount))
	s.Require().NoError(s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM report_snapshots WHERE type = 0").Scan(&vulnerabilityCount))
	s.Require().NoError(s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM report_snapshots WHERE type = 1").Scan(&nodeVulnerabilityCount))

	s.Require().Equal(2, nullCount)
	s.Require().Equal(1, vulnerabilityCount)
	s.Require().Equal(1, nodeVulnerabilityCount)

	// Run migration
	s.Require().NoError(migration.Run(dbs))

	// Verify post-migration state
	s.Require().NoError(s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM report_snapshots WHERE type IS NULL").Scan(&nullCount))
	s.Require().NoError(s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM report_snapshots WHERE type = 0").Scan(&vulnerabilityCount))
	s.Require().NoError(s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM report_snapshots WHERE type = 1").Scan(&nodeVulnerabilityCount))

	s.Require().Equal(0, nullCount, "No NULL values should remain")
	s.Require().Equal(3, vulnerabilityCount, "NULL values should be backfilled to 0, existing 0 unchanged")
	s.Require().Equal(1, nodeVulnerabilityCount, "NODE_VULNERABILITY reports should remain unchanged")
}
