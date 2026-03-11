//go:build sql_integration

package m221tom222

import (
	"context"
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
	_, _ = s.db.Exec(s.ctx, "DROP TABLE IF EXISTS image_cve_infos")
	_, _ = s.db.Exec(s.ctx, "DROP TABLE IF EXISTS image_cves_v2")
	_, _ = s.db.Exec(s.ctx, "DROP TABLE IF EXISTS image_component_v2")
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
	cveTime := time.Date(2021, 12, 9, 10, 0, 0, 0, time.UTC)
	cveRecord := s.createImageCVE("CVE-2021-44228", componentID, "ubuntu-updater::ubuntu:20.04", &cveTime)

	// Insert test data
	s.Require().NoError(dbs.GormDB.Create(&component).Error)
	s.Require().NoError(dbs.GormDB.Create(&cveRecord).Error)

	// Run migration
	s.Require().NoError(migration.Run(dbs))

	// Verify image_cve_infos populated
	var count int64
	s.Require().NoError(dbs.GormDB.Table("image_cve_infos").Count(&count).Error)
	s.Assert().Equal(int64(1), count, "should have one image_cve_infos record")

	// Verify timestamp is correct
	var info schema.ImageCveInfos
	expectedID := cve.ImageCVEInfoID("CVE-2021-44228", "curl", "ubuntu-updater::ubuntu:20.04")
	s.Require().NoError(dbs.GormDB.Table("image_cve_infos").Where("id = ?", expectedID).First(&info).Error)
	s.Assert().Equal(expectedID, info.ID)
	s.Assert().Equal("CVE-2021-44228", info.Cve)
	s.Assert().NotNil(info.FirstSystemOccurrence)
	s.Assert().Equal(cveTime.Unix(), info.FirstSystemOccurrence.Unix())
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
	s.Require().NoError(dbs.GormDB.Table("image_cve_infos").Count(&count).Error)
	s.Assert().Equal(int64(1), count, "should aggregate to one record")

	// Verify it has the MIN timestamp
	var info schema.ImageCveInfos
	expectedID := cve.ImageCVEInfoID("CVE-2021-3711", "openssl", "rhel-updater::rhel:8")
	s.Require().NoError(dbs.GormDB.Table("image_cve_infos").Where("id = ?", expectedID).First(&info).Error)
	s.Assert().NotNil(info.FirstSystemOccurrence)
	s.Assert().Equal(time3.Unix(), info.FirstSystemOccurrence.Unix(), "should use earliest timestamp")
}

// TestMigration_HandlesEmptyDatasource tests Red Hat vulnerabilities with empty datasource
func (s *migrationTestSuite) TestMigration_HandlesEmptyDatasource() {
	dbs := s.getDatabases()

	componentID := uuid.NewV4().String()
	component := s.createComponent(componentID, "glibc", "2.31")
	cveTime := time.Date(2022, 1, 15, 10, 0, 0, 0, time.UTC)

	// Create CVE with empty datasource (Red Hat vulnerability pattern)
	cveRecord := s.createImageCVE("CVE-2022-23218", componentID, "", &cveTime)
	s.Require().NoError(dbs.GormDB.Create(&component).Error)
	s.Require().NoError(dbs.GormDB.Create(&cveRecord).Error)

	// Run migration
	s.Require().NoError(migration.Run(dbs))

	// Verify ID format is correct: cve#package# (empty datasource)
	expectedID := cve.ImageCVEInfoID("CVE-2022-23218", "glibc", "")
	var info schema.ImageCveInfos
	s.Require().NoError(dbs.GormDB.Table("image_cve_infos").Where("id = ?", expectedID).First(&info).Error)
	s.Assert().Equal(expectedID, info.ID)
	s.Assert().Equal("CVE-2022-23218", info.Cve)
}

// TestMigration_DoesNotPopulateFixAvailableTimestamp verifies user decision to leave fix timestamp NULL
func (s *migrationTestSuite) TestMigration_DoesNotPopulateFixAvailableTimestamp() {
	dbs := s.getDatabases()

	componentID := uuid.NewV4().String()
	component := s.createComponent(componentID, "nginx", "1.18.0")
	cveTime := time.Date(2022, 3, 1, 10, 0, 0, 0, time.UTC)

	cveRecord := s.createImageCVE("CVE-2022-1234", componentID, "alpine-updater::alpine:3.14", &cveTime)
	s.Require().NoError(dbs.GormDB.Create(&component).Error)
	s.Require().NoError(dbs.GormDB.Create(&cveRecord).Error)

	// Run migration
	s.Require().NoError(migration.Run(dbs))

	// Verify fix_available_timestamp is NULL
	var info schema.ImageCveInfos
	expectedID := cve.ImageCVEInfoID("CVE-2022-1234", "nginx", "alpine-updater::alpine:3.14")
	s.Require().NoError(dbs.GormDB.Table("image_cve_infos").Where("id = ?", expectedID).First(&info).Error)
	s.Assert().Nil(info.FixAvailableTimestamp, "migration should not populate fix_available_timestamp")
}

// TestMigration_PopulatesCVEColumn verifies the indexed cve column is populated
func (s *migrationTestSuite) TestMigration_PopulatesCVEColumn() {
	dbs := s.getDatabases()

	componentID := uuid.NewV4().String()
	component := s.createComponent(componentID, "python", "3.8.10")
	cveTime := time.Date(2022, 5, 1, 10, 0, 0, 0, time.UTC)

	cveRecord := s.createImageCVE("CVE-2022-5678", componentID, "ubuntu-updater::ubuntu:22.04", &cveTime)
	s.Require().NoError(dbs.GormDB.Create(&component).Error)
	s.Require().NoError(dbs.GormDB.Create(&cveRecord).Error)

	// Run migration
	s.Require().NoError(migration.Run(dbs))

	// Verify cve column is populated (required for efficient queries)
	var nullCount int64
	s.Require().NoError(dbs.GormDB.Table("image_cve_infos").Where("cve IS NULL").Count(&nullCount).Error)
	s.Assert().Equal(int64(0), nullCount, "all records should have cve column populated")

	// Verify cve column value is correct
	var info schema.ImageCveInfos
	expectedID := cve.ImageCVEInfoID("CVE-2022-5678", "python", "ubuntu-updater::ubuntu:22.04")
	s.Require().NoError(dbs.GormDB.Table("image_cve_infos").Where("id = ?", expectedID).First(&info).Error)
	s.Assert().Equal("CVE-2022-5678", info.Cve)
}

// TestMigration_HandlesMultipleDatasources tests different CVEs across different datasources
func (s *migrationTestSuite) TestMigration_HandlesMultipleDatasources() {
	dbs := s.getDatabases()

	componentID := uuid.NewV4().String()
	component := s.createComponent(componentID, "curl", "7.68.0")
	cveTime := time.Date(2022, 6, 1, 10, 0, 0, 0, time.UTC)

	// Same CVE + package but different datasources should create separate records
	cve1 := s.createImageCVE("CVE-2022-9999", componentID, "ubuntu-updater::ubuntu:20.04", &cveTime)
	cve2 := s.createImageCVE("CVE-2022-9999", componentID, "debian-updater::debian:11", &cveTime)

	s.Require().NoError(dbs.GormDB.Create(&component).Error)
	s.Require().NoError(dbs.GormDB.Create(&cve1).Error)
	s.Require().NoError(dbs.GormDB.Create(&cve2).Error)

	// Run migration
	s.Require().NoError(migration.Run(dbs))

	// Verify two separate image_cve_infos records created
	var count int64
	s.Require().NoError(dbs.GormDB.Table("image_cve_infos").Count(&count).Error)
	s.Assert().Equal(int64(2), count, "different datasources should create separate records")

	// Verify both records exist with correct IDs
	id1 := cve.ImageCVEInfoID("CVE-2022-9999", "curl", "ubuntu-updater::ubuntu:20.04")
	id2 := cve.ImageCVEInfoID("CVE-2022-9999", "curl", "debian-updater::debian:11")

	var info1, info2 schema.ImageCveInfos
	s.Require().NoError(dbs.GormDB.Table("image_cve_infos").Where("id = ?", id1).First(&info1).Error)
	s.Require().NoError(dbs.GormDB.Table("image_cve_infos").Where("id = ?", id2).First(&info2).Error)
	s.Assert().Equal("CVE-2022-9999", info1.Cve)
	s.Assert().Equal("CVE-2022-9999", info2.Cve)
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
