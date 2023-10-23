//go:build sql_integration

package m192tom193

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	newSchema "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_report_config_migrations/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	reportID = uuid.NewV4().String()
)

type migrationTestSuite struct {
	suite.Suite

	db             *pghelper.TestPostgres
	ctx            context.Context
	gormDB         *gorm.DB
	snapshotgormdB *gorm.DB
	testDB         *pgtest.TestPostgres
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	//create report config table to insert v1 config for testing
	s.db = pghelper.ForT(s.T(), false)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), newSchema.CreateTableReportConfigurationsStmt)
	s.gormDB = s.db.GetGormDB()
	s.gormDB = s.gormDB.WithContext(s.ctx).Table("report_configurations")
	s.snapshotgormdB = s.db.GetGormDB()
	s.snapshotgormdB = s.snapshotgormdB.WithContext(s.ctx).Table("report_configurations_notifiers")

	notifierdB := s.db.GetGormDB()
	notifierdB = notifierdB.WithContext(s.ctx).Table("notifiers")
	ret := fixtures.GetValidReportConfiguration()
	ret.Id = reportID
	notifierID := uuid.NewV4().String()
	ret.NotifierConfig = &storage.ReportConfiguration_EmailConfig{
		EmailConfig: &storage.EmailNotifierConfiguration{
			NotifierId:   notifierID,
			MailingLists: []string{"foo@yahoo.com"},
		},
	}
	ret.LastRunStatus = &storage.ReportLastRunStatus{
		ReportStatus: storage.ReportLastRunStatus_SUCCESS,
	}

	ret.LastSuccessfulRunTime = timestamp.Now().GogoProtobuf()
	converted, err := newSchema.ConvertReportConfigurationFromProto(ret)
	s.Require().NoError(err)
	convertedReportConfigs := []*newSchema.ReportConfigurations{converted}
	err = s.gormDB.Clauses(clause.OnConflict{UpdateAll: true}).Model(newSchema.CreateTableReportConfigurationsStmt.GormModel).Create(&convertedReportConfigs).Error
	s.Require().NoError(err)

	notifierProto := &storage.Notifier{
		Id: notifierID,
	}

	notifier, err := newSchema.ConvertNotifierFromProto(notifierProto)
	notifiers := []*newSchema.Notifiers{notifier}
	s.Require().NoError(err)
	err = notifierdB.Clauses(clause.OnConflict{UpdateAll: true}).Model(newSchema.CreateTableNotifiersStmt.GormModel).Create(&notifiers).Error
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

	migration.Run(dbs)

	configs, _ := s.gormDB.Rows()
	snapshots, _ := s.snapshotgormdB.Rows()
	actualConfigProto := []*storage.ReportConfiguration{}
	v1Config := &storage.ReportConfiguration{}
	v2Config := &storage.ReportConfiguration{}
	for configs.Next() {
		var reportConfig *newSchema.ReportConfigurations
		err := s.gormDB.ScanRows(configs, &reportConfig)
		s.Require().NoError(err)
		config, _ := newSchema.ConvertReportConfigurationToProto(reportConfig)
		actualConfigProto = append(actualConfigProto, config)
		if config.GetId() == reportID {
			v1Config = config
		} else {
			v2Config = config
		}
	}
	actualSnapahshotProto := []*storage.ReportSnapshot{}

	for snapshots.Next() {
		var snapshot *newSchema.ReportSnapshots
		err := s.snapshotgormdB.ScanRows(snapshots, &snapshot)
		s.Require().NoError(err)
		repSnapshot, _ := newSchema.ConvertReportSnapshotToProto(snapshot)
		actualSnapahshotProto = append(actualSnapahshotProto, repSnapshot)
	}
	//there should be 2 copies of report config
	s.Equal(len(actualConfigProto), 2)
	s.Equal(int32(1), v1Config.GetVersion())
	s.Equal(int32(2), v2Config.GetVersion())
	s.Equal(v2Config.GetResourceScope().GetCollectionId(), v1Config.GetScopeId())

	s.Equal(len(actualSnapahshotProto), 1)
}
