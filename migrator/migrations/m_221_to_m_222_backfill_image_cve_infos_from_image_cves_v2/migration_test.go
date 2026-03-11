//go:build sql_integration

package m221tom222

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_221_to_m_222_backfill_image_cve_infos_from_image_cves_v2/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
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

func (s *migrationTestSuite) SetupTest() {
	// Create tables for each test
	dbs := s.getDatabases()
	pgutils.CreateTableFromModel(s.ctx, dbs.GormDB, schema.CreateTableImageComponentV2Stmt)
	pgutils.CreateTableFromModel(s.ctx, dbs.GormDB, schema.CreateTableImageCvesV2Stmt)
	pgutils.CreateTableFromModel(s.ctx, dbs.GormDB, schema.CreateTableImageCveInfosStmt)
}

func (s *migrationTestSuite) TearDownTest() {
	// Clean up tables after each test
	_, _ = s.db.Exec(s.ctx, "DROP TABLE IF EXISTS "+schema.ImageCveInfosTableName)
	_, _ = s.db.Exec(s.ctx, "DROP TABLE IF EXISTS "+schema.ImageCvesV2TableName)
	_, _ = s.db.Exec(s.ctx, "DROP TABLE IF EXISTS "+schema.ImageComponentV2TableName)
}

func (s *migrationTestSuite) getDatabases() *types.Databases {
	return &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}
}

// TestMigration_BackfillsImageCVEInfos tests the basic happy path
func (s *migrationTestSuite) TestMigration_BackfillsImageCVEInfos() {
	dbs := s.getDatabases()

	// Create test data
	componentID := uuid.NewV4().String()
	component := s.createComponent(componentID, "curl", "7.68.0")

	cveName := "CVE-2021-44228"
	cveTime := time.Date(2021, 12, 9, 10, 0, 0, 0, time.UTC)
	cveRecord := s.createImageCVE(cveName, componentID, "ubuntu-updater::ubuntu:20.04", &cveTime)

	// Insert test data
	s.Require().NoError(dbs.GormDB.Create(&component).Error)
	s.Require().NoError(dbs.GormDB.Create(&cveRecord).Error)

	// Run migration
	s.Require().NoError(migration.Run(dbs))

	// Verify image_cve_infos populated
	var count int64
	s.Require().NoError(dbs.GormDB.Table(schema.ImageCveInfosTableName).Count(&count).Error)
	s.Assert().Equal(int64(1), count, "should have one image_cve_infos record")

	// Verify timestamp is correct
	var info schema.ImageCveInfos
	expectedID := cve.ImageCVEInfoID(cveName, "curl", "ubuntu-updater::ubuntu:20.04")
	s.Require().NoError(dbs.GormDB.Table(schema.ImageCveInfosTableName).Where("id = ?", expectedID).First(&info).Error)
	s.Assert().Equal(expectedID, info.ID)
	s.Assert().Equal(cveName, info.Cve)
	s.Assert().Equal(cveTime, *info.FirstSystemOccurrence)
	s.Assert().Nil(info.FixAvailableTimestamp, "fix_available_timestamp should be nil per user decision")
}

// TestMigration_AggregatesMultipleCVEs tests MIN aggregation for same CVE across multiple images
func (s *migrationTestSuite) TestMigration_AggregatesMultipleCVEs() {
	dbs := s.getDatabases()

	componentID := uuid.NewV4().String()
	component := s.createComponent(componentID, "openssl", "1.1.1")

	// Create 3 image_cves_v2 records with same CVE+package+datasource but different timestamps
	time1 := time.Date(2021, 10, 1, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2021, 11, 1, 10, 0, 0, 0, time.UTC)
	time3 := time.Date(2021, 9, 1, 10, 0, 0, 0, time.UTC) // Earliest

	cve1 := s.createImageCVE("CVE-2021-3711", componentID, "rhel-updater::rhel:8", &time1)
	cve2 := s.createImageCVE("CVE-2021-3711", componentID, "rhel-updater::rhel:8", &time2)
	cve3 := s.createImageCVE("CVE-2021-3711", componentID, "rhel-updater::rhel:8", &time3)

	s.Require().NoError(dbs.GormDB.Create(&component).Error)
	s.Require().NoError(dbs.GormDB.Create(&cve1).Error)
	s.Require().NoError(dbs.GormDB.Create(&cve2).Error)
	s.Require().NoError(dbs.GormDB.Create(&cve3).Error)

	// Run migration
	s.Require().NoError(migration.Run(dbs))

	// Verify only one image_cve_infos record created
	var count int64
	s.Require().NoError(dbs.GormDB.Table(schema.ImageCveInfosTableName).Count(&count).Error)
	s.Assert().Equal(int64(1), count, "should aggregate to one record")

	// Verify it has the MIN timestamp
	var info schema.ImageCveInfos
	expectedID := cve.ImageCVEInfoID("CVE-2021-3711", "openssl", "rhel-updater::rhel:8")
	s.Require().NoError(dbs.GormDB.Table(schema.ImageCveInfosTableName).Where("id = ?", expectedID).First(&info).Error)
	s.Assert().NotNil(info.FirstSystemOccurrence)
	s.Assert().Equal(time3, *info.FirstSystemOccurrence, "should use earliest timestamp")
}

// TestMigration_LargeDatasetWithPagination tests pagination
func (s *migrationTestSuite) TestMigration_LargeDatasetWithPagination() {
	dbs := s.getDatabases()

	// Override batch sizes for this test to verify pagination
	readBatchSize = 25
	upsertBatchSize = 10

	// Create components
	numComponents := 25
	components := make([]schema.ImageComponentV2, numComponents)
	for i := 0; i < numComponents; i++ {
		componentID := uuid.NewV4().String()
		components[i] = s.createComponent(componentID, fmt.Sprintf("package-%d", i), "1.0.0")
		s.Require().NoError(dbs.GormDB.Create(&components[i]).Error)
	}

	// Create ~200 CVEs that aggregate to 75 unique CVE infos
	// Each unique (CVE, package, datasource) combination will have 2-3 duplicate CVEs with different timestamps
	numUniqueCVEInfos := 75
	numCVEsPerInfo := []int{2, 3, 2} // Cycle through 2, 3, 2 duplicates
	totalCVEs := 0
	baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	datasources := []string{"ubuntu-updater::ubuntu:20.04", "debian-updater::debian:11", "rhel-updater::rhel:8"}

	// Track expected minimum timestamps for verification
	expectedMinTimestamps := make(map[string]time.Time)

	for i := 0; i < numUniqueCVEInfos; i++ {
		cveName := fmt.Sprintf("CVE-2023-%05d", 10000+i)
		component := components[i%numComponents]
		datasource := datasources[i%len(datasources)]

		// Determine how many CVEs to create for this unique combo
		numDuplicates := numCVEsPerInfo[i%len(numCVEsPerInfo)]

		// Create multiple CVEs with different timestamps for the same (CVE, package, datasource)
		id := cve.ImageCVEInfoID(cveName, component.Name, datasource)
		for j := 0; j < numDuplicates; j++ {
			// Each duplicate has a different timestamp (earlier timestamps first)
			timestamp := baseTime.Add(time.Duration(i*numDuplicates+j) * time.Hour)
			cveRecord := s.createImageCVE(cveName, component.ID, datasource, &timestamp)
			s.Require().NoError(dbs.GormDB.Create(&cveRecord).Error)
			totalCVEs++
		}
		expectedMinTimestamps[id] = baseTime.Add(time.Duration(i*numDuplicates) * time.Hour)
	}

	s.T().Logf("Created %d CVEs that should aggregate to %d unique CVE infos", totalCVEs, numUniqueCVEInfos)

	// Run migration
	s.Require().NoError(migration.Run(dbs))

	// Verify migration results
	// Verify correct number of aggregated records
	var infos []schema.ImageCveInfos
	s.Require().NoError(dbs.GormDB.Table(schema.ImageCveInfosTableName).Find(&infos).Error)
	s.Require().Equal(numUniqueCVEInfos, len(infos), "should retrieve all aggregated records")

	// Verify each record has the minimum timestamp
	for _, info := range infos {
		s.Assert().NotNil(info.FirstSystemOccurrence, "FirstSystemOccurrence should not be nil for ID %s", info.ID)
		expectedMin, ok := expectedMinTimestamps[info.ID]
		s.Require().True(ok, "expected minimum timestamp should exist for ID %s", info.ID)
		s.Assert().Equal(expectedMin, *info.FirstSystemOccurrence,
			"MIN timestamp should match for ID %s", info.ID)
		s.Assert().Nil(info.FixAvailableTimestamp, "FixAvailableTimestamp should be nil for ID %s", info.ID)
	}
}

// Helper functions

func (s *migrationTestSuite) createComponent(id, name, version string) schema.ImageComponentV2 {
	proto := &storage.ImageComponentV2{
		Id:      id,
		Name:    name,
		Version: version,
		Source:  storage.SourceType_OS,
	}

	serialized, err := proto.MarshalVT()
	s.Require().NoError(err)

	return schema.ImageComponentV2{
		ID:              id,
		Name:            name,
		Version:         version,
		Source:          storage.SourceType_OS,
		OperatingSystem: "linux",
		Serialized:      serialized,
	}
}

func (s *migrationTestSuite) createImageCVE(cveName, componentID, datasource string, createdAt *time.Time) schema.ImageCvesV2 {
	proto := &storage.ImageCVEV2{
		// Fake unique ID.
		Id:          uuid.NewV4().String(),
		Datasource:  datasource,
		ComponentId: componentID,
		CveBaseInfo: &storage.CVEInfo{
			Cve:       cveName,
			CreatedAt: protocompat.ConvertTimeToTimestampOrNil(createdAt),
		},
	}

	serialized, err := proto.MarshalVT()
	s.Require().NoError(err)

	return schema.ImageCvesV2{
		ID:                   proto.GetId(),
		CveBaseInfoCve:       cveName,
		CveBaseInfoCreatedAt: createdAt,
		ComponentID:          componentID,
		Serialized:           serialized,
	}
}
