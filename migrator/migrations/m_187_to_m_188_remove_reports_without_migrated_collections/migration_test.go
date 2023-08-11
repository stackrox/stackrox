//go:build sql_integration

package m187tom188

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_187_to_m_188_remove_reports_without_migrated_collections/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

type migrationTestSuite struct {
	suite.Suite

	db     *pghelper.TestPostgres
	gormDB *gorm.DB
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.db = pghelper.ForT(s.T(), false)
	s.gormDB = s.db.GetGormDB().WithContext(ctx)

	pgutils.CreateTableFromModel(ctx, s.db.GetGormDB(), schema.CreateTableReportConfigurationsStmt)
	pgutils.CreateTableFromModel(ctx, s.db.GetGormDB(), schema.CreateTableCollectionsStmt)
}

func (s *migrationTestSuite) createConfigInDB(id string, scopeID string, collectionID string) {
	c, err := schema.ConvertReportConfigurationFromProto(&storage.ReportConfiguration{
		Id:      id,
		Name:    id,
		ScopeId: scopeID,
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_CollectionId{
				CollectionId: collectionID,
			},
		},
	})
	s.Require().NoError(err)
	s.Require().NoError(s.gormDB.Create(c).Error)
}

func (s *migrationTestSuite) createCollectionInDB(id string) {
	c, err := schema.ConvertResourceCollectionFromProto(&storage.ResourceCollection{Id: id, Name: id})
	s.Require().NoError(err)
	s.Require().NoError(s.gormDB.Create(c).Error)
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) TestMigration() {
	var validV1Reports []string
	var validV2Reports []string
	var invalidV1Reports []string
	var invalidReports []string
	var collections []string

	// Create some collections
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("collection-%d", i)
		collections = append(collections, id)
		s.createCollectionInDB(id)
	}

	// Valid V1 reports (i.e. scope points to collections)
	for i := 0; i < len(collections); i++ {
		id := fmt.Sprintf("v1-report-config-%d", i)
		validV1Reports = append(validV1Reports, id)
		s.createConfigInDB(id, collections[i], "")
	}

	// Invalid V1 reports (scope points to non-existent collections)
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("invalid-report-config-%d", i)
		scopeID := fmt.Sprintf("invalid-collection-%d", i)
		invalidV1Reports = append(invalidV1Reports, id)
		s.createConfigInDB(id, scopeID, "")
	}

	// Valid V2 reports (collection_id is set)
	for i := 0; i < len(collections); i++ {
		id := fmt.Sprintf("v2-report-config-%d", i)
		validV2Reports = append(validV2Reports, id)
		s.createConfigInDB(id, "", collections[i])
	}

	// Valid V2 reports with both scope and collection_id
	for i := 0; i < len(collections); i++ {
		id := fmt.Sprintf("v2-report-config-%d", i+10)
		validV2Reports = append(validV2Reports, id)
		s.createConfigInDB(id, collections[i], collections[i])
	}

	// Valid V2 reports with both invalid scope and collection_id
	for i := 0; i < len(collections); i++ {
		id := fmt.Sprintf("v2-report-config-%d", i+20)
		scopeID := fmt.Sprintf("invalid-collection-%d", i)
		validV2Reports = append(validV2Reports, id)
		s.createConfigInDB(id, scopeID, collections[i])
	}

	// Invalid configs that have neither collections nor scope ids
	// Empty string
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("invalid-report-config-%d", i+10)
		invalidReports = append(invalidReports, id)
		s.createConfigInDB(id, "", "")
	}

	// Whitespace
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("invalid-report-config-%d", i+13)
		invalidReports = append(invalidReports, id)
		s.createConfigInDB(id, " ", " ")
	}

	// Null ids
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("invalid-report-config-%d", i+16)
		invalidReports = append(invalidReports, id)
		c, err := schema.ConvertReportConfigurationFromProto(&storage.ReportConfiguration{
			Id:   id,
			Name: id,
		})
		s.Require().NoError(err)
		s.Require().NoError(s.gormDB.Create(c).Error)
	}

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
	}
	s.Require().NoError(migration.Run(dbs))

	// Validate that valid v1 and v2 reports still exist in db
	for _, id := range append(validV1Reports, validV2Reports...) {
		var reportConfig schema.ReportConfigurations
		result := s.gormDB.Table(schema.ReportConfigurationsTableName).Limit(1).Where(&schema.ReportConfigurations{ID: id}).Find(&reportConfig)
		s.Require().NoError(result.Error)
		s.Equal(reportConfig.ID, id)
	}

	// Validate that all invalid reports are no longer there
	for _, id := range append(invalidV1Reports, invalidReports...) {
		var count int64
		result := s.gormDB.Table(schema.ReportConfigurationsTableName).Where(&schema.ReportConfigurations{ID: id}).Count(&count)
		s.Require().NoError(result.Error)
		s.Zero(count)
	}
}
