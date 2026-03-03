//go:build sql_integration

package m220tom221

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	postgresStore "github.com/stackrox/rox/migrator/migrations/m_220_to_m_221_remove_v1_report_configs/postgres"
	pkgSchema "github.com/stackrox/rox/migrator/migrations/m_220_to_m_221_remove_v1_report_configs/postgres/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
	suite.Suite

	db    *pghelper.TestPostgres
	ctx   context.Context
	store postgresStore.Store
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), pkgSchema.CreateTableReportConfigurationsStmt)
}

func (s *migrationTestSuite) SetupTest() {
	s.store = postgresStore.New(s.db.DB)
}

func (s *migrationTestSuite) TearDownTest() {
	configs, err := s.store.GetIDs(s.ctx)
	s.Require().NoError(err)
	s.Require().NoError(s.store.DeleteMany(s.ctx, configs))
}

func (s *migrationTestSuite) TestMigrationWithNoConfigs() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	configs, err := s.store.GetIDs(s.ctx)
	s.Require().NoError(err)
	s.Require().Empty(configs)
}

func (s *migrationTestSuite) TestMigrationWithOnlyV1Configs() {
	v1Configs := []*storage.ReportConfiguration{
		s.createReportConfig("v1-config-1", 1),
		s.createReportConfig("v1-config-2", 1),
		s.createReportConfig("v1-config-3", 1),
	}

	s.Require().NoError(s.store.UpsertMany(s.ctx, v1Configs))

	configsBefore, err := s.store.GetIDs(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(configsBefore, 3)

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	configsAfter, err := s.store.GetIDs(s.ctx)
	s.Require().NoError(err)
	s.Require().Empty(configsAfter, "All V1 configs should be deleted")
}

func (s *migrationTestSuite) TestMigrationWithOnlyV2Configs() {
	v2Configs := []*storage.ReportConfiguration{
		s.createReportConfig("v2-config-1", 2),
		s.createReportConfig("v2-config-2", 2),
		s.createReportConfig("v2-config-3", 2),
	}

	s.Require().NoError(s.store.UpsertMany(s.ctx, v2Configs))

	configsBefore, err := s.store.GetIDs(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(configsBefore, 3)

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	configsAfter, err := s.store.GetIDs(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(configsAfter, len(configsBefore), "All V2 configs should remain")

	for _, v2Config := range v2Configs {
		config, exists, err := s.store.Get(s.ctx, v2Config.GetId())
		s.Require().NoError(err)
		s.Require().True(exists)
		s.Require().Equal(int32(2), config.GetVersion())
		s.Require().Equal(v2Config.GetName(), config.GetName())
	}
}

func (s *migrationTestSuite) TestMigrationWithMixedConfigs() {
	v1Configs := []*storage.ReportConfiguration{
		s.createReportConfig("v1-config-1", 1),
		s.createReportConfig("v1-config-2", 1),
	}

	v2Configs := []*storage.ReportConfiguration{
		s.createReportConfig("v2-config-1", 2),
		s.createReportConfig("v2-config-2", 2),
		s.createReportConfig("v2-config-3", 2),
	}

	allConfigs := append(v1Configs, v2Configs...)
	s.Require().NoError(s.store.UpsertMany(s.ctx, allConfigs))

	configsBefore, err := s.store.GetIDs(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(configsBefore, len(allConfigs))

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	configsAfter, err := s.store.GetIDs(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(configsAfter, len(v2Configs), "Only V2 configs should remain")

	for _, v1Config := range v1Configs {
		_, exists, err := s.store.Get(s.ctx, v1Config.GetId())
		s.Require().NoError(err)
		s.Require().False(exists, "V1 config %s should be deleted", v1Config.GetName())
	}

	for _, v2Config := range v2Configs {
		config, exists, err := s.store.Get(s.ctx, v2Config.GetId())
		s.Require().NoError(err)
		s.Require().True(exists, "V2 config %s should exist", v2Config.GetName())
		s.Require().Equal(int32(2), config.GetVersion())
		s.Require().Equal(v2Config.GetName(), config.GetName())
	}
}

func (s *migrationTestSuite) TestMigrationWithUnmigratedV1Configs() {
	unmigratedV1Configs := []*storage.ReportConfiguration{
		s.createReportConfig("unmigrated-v1-config-1", 0),
		s.createReportConfig("unmigrated-v1-config-2", 0),
	}

	migratedV1Configs := []*storage.ReportConfiguration{
		s.createReportConfig("migrated-v1-config-1", 1),
		s.createReportConfig("migrated-v1-config-2", 1),
	}

	v2Configs := []*storage.ReportConfiguration{
		s.createReportConfig("v2-config-1", 2),
	}

	allConfigs := append(unmigratedV1Configs, migratedV1Configs...)
	allConfigs = append(allConfigs, v2Configs...)
	s.Require().NoError(s.store.UpsertMany(s.ctx, allConfigs))

	configsBefore, err := s.store.GetIDs(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(configsBefore, 5)

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	configsAfter, err := s.store.GetIDs(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(configsAfter, 1, "Only V2 configs should remain")

	for _, migratedV1Config := range migratedV1Configs {
		_, exists, err := s.store.Get(s.ctx, migratedV1Config.GetId())
		s.Require().NoError(err)
		s.Require().False(exists, "Migrated V1 config (version=1) %s should be deleted", migratedV1Config.GetName())
	}

	for _, unmigratedV1Config := range unmigratedV1Configs {
		_, exists, err := s.store.Get(s.ctx, unmigratedV1Config.GetId())
		s.Require().NoError(err)
		s.Require().False(exists, "Unmigrated V1 config (version=0) %s should be deleted", unmigratedV1Config.GetName())
	}

	for _, v2Config := range v2Configs {
		config, exists, err := s.store.Get(s.ctx, v2Config.GetId())
		s.Require().NoError(err)
		s.Require().True(exists, "V2 config %s should exist", v2Config.GetName())
		s.Require().Equal(int32(2), config.GetVersion())
	}
}

func (s *migrationTestSuite) createReportConfig(name string, version int32) *storage.ReportConfiguration {
	return &storage.ReportConfiguration{
		Id:          uuid.NewV4().String(),
		Name:        name,
		Description: "Test report configuration",
		Type:        storage.ReportConfiguration_VULNERABILITY,
		Version:     version,
		Filter: &storage.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: &storage.VulnerabilityReportFilters{
				Fixability: storage.VulnerabilityReportFilters_FIXABLE,
				Severities: []storage.VulnerabilitySeverity{
					storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				},
			},
		},
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_CollectionId{
				CollectionId: uuid.NewV4().String(),
			},
		},
	}
}
