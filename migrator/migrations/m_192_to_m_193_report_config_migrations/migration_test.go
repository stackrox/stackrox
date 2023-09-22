//go:build sql_integration

package m192tom193

import (
	"context"
	"testing"

	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	"github.com/stackrox/rox/generated/storage"
	newSchema "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_report_config_migrations/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type migrationTestSuite struct {
	suite.Suite

	db        *pghelper.TestPostgres
	ctx       context.Context
	gormDB    *gorm.DB
	testDB    *pgtest.TestPostgres
	datastore reportConfigDS.DataStore
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())
	s.datastore = reportConfigDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)

	//create report config table to insert v1 config for testing
	s.db = pghelper.ForT(s.T(), false)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), newSchema.CreateTableReportConfigurationsStmt)
	s.gormDB = s.db.GetGormDB()
	s.gormDB = s.gormDB.WithContext(s.ctx).Table("report_configurations")
	ret := fixtures.GetValidReportConfiguration()
	ret.LastRunStatus = &storage.ReportLastRunStatus{
		ReportStatus: storage.ReportLastRunStatus_SUCCESS,
	}
	converted, err := newSchema.ConvertReportConfigurationFromProto(ret)
	s.Require().NoError(err)
	convertedReportConfigs := []*newSchema.ReportConfigurations{converted}
	err = s.gormDB.Clauses(clause.OnConflict{UpdateAll: true}).Model(newSchema.CreateTableReportConfigurationsStmt.GormModel).Create(&convertedReportConfigs).Error
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
	s.Require().NoError(migration.Run(dbs))

	configs, _ := s.gormDB.Rows()
	actualConfigProto := []*storage.ReportConfiguration{}
	for configs.Next() {
		var reportConfig *newSchema.ReportConfigurations
		err := s.gormDB.ScanRows(configs, &reportConfig)
		s.Require().NoError(err)
		config, _ := newSchema.ConvertReportConfigurationToProto(reportConfig)
		actualConfigProto = append(actualConfigProto, config)
	}
	//there should be 2 copies of report config
	s.Equal(len(actualConfigProto), 2)
}
