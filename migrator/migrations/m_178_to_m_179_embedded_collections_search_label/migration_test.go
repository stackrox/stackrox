//go:build sql_integration

package m178tom179

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	newStore "github.com/stackrox/rox/migrator/migrations/m_178_to_m_179_embedded_collections_search_label/reportconfigstore"
	oldStore "github.com/stackrox/rox/migrator/migrations/m_178_to_m_179_embedded_collections_search_label/test"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

var (
	reportsToUpsert = []*storage.ReportConfiguration{
		{
			Id:      "report1",
			Name:    "report1",
			ScopeId: "scope1",
		},
		{
			Id:      "report2",
			Name:    "report2",
			ScopeId: "scope2",
		},
		{
			Id:      "report3",
			Name:    "report3",
			ScopeId: "scope3",
		},
		{
			Id:      "report4",
			Name:    "report4",
			ScopeId: "scope4",
		},
		{
			Id:      "report5",
			Name:    "report5",
			ScopeId: "scope5",
		},
	}
)

type reportConfigMigrationTestSuite struct {
	suite.Suite

	db             *pghelper.TestPostgres
	oldReportStore oldStore.Store
	newReportStore newStore.Store
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(reportConfigMigrationTestSuite))
}

func (s *reportConfigMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), false)
	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), oldSchema.CreateTableReportConfigurationsStmt)
	s.oldReportStore = oldStore.New(s.db.DB)
	s.newReportStore = newStore.New(s.db.DB)
}

func (s *reportConfigMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *reportConfigMigrationTestSuite) TestMigration() {
	ctx := sac.WithAllAccess(context.Background())

	err := s.oldReportStore.UpsertMany(ctx, reportsToUpsert)
	s.NoError(err)
	count, err := s.oldReportStore.Count(ctx)
	s.NoError(err)
	s.Equal(len(reportsToUpsert), count)

	err = migrateReportConfigs(s.db.DB, s.db.GetGormDB())
	s.NoError(err)

	count, err = s.newReportStore.Count(ctx)
	s.NoError(err)
	s.Equal(len(reportsToUpsert), count)
	row := s.db.QueryRow(ctx, "select count(*) from report_configurations where scopeid is NULL")
	err = row.Scan(&count)
	s.NoError(err)
	s.Equal(0, count)
}
