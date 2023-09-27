//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestReportConfigurationPostgresDatastore(t *testing.T) {
	suite.Run(t, new(ReportConfigurationPostgresDatastoreTests))
}

type ReportConfigurationPostgresDatastoreTests struct {
	suite.Suite

	testDB    *pgtest.TestPostgres
	datastore DataStore
	ctx       context.Context
}

func (s *ReportConfigurationPostgresDatastoreTests) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.datastore = GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
}

func (s *ReportConfigurationPostgresDatastoreTests) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *ReportConfigurationPostgresDatastoreTests) TearDownTest() {
	s.truncateTable(postgresSchema.ReportConfigurationsTableName)
}

func (s *ReportConfigurationPostgresDatastoreTests) TestReportsConfigDataStore() {
	reportConfig := fixtures.GetValidReportConfiguration()
	// Test add
	_, err := s.datastore.AddReportConfiguration(s.ctx, reportConfig)
	s.Require().NoError(err)

	// Test get
	foundReportConfig, found, err := s.datastore.GetReportConfiguration(s.ctx, reportConfig.GetId())
	s.Require().NoError(err)
	s.True(found)
	s.Equal(reportConfig, foundReportConfig)

	// Test search by name
	query := search.NewQueryBuilder().AddStrings(search.ReportName, reportConfig.Name).ProtoQuery()
	searchResults, err := s.datastore.Search(s.ctx, query)
	s.NoError(err)
	s.Len(searchResults, 1)
	s.Equal(searchResults[0].ID, foundReportConfig.Id)

	// Test not found
	_, found, err = s.datastore.GetReportConfiguration(s.ctx, "NONEXISTENT")
	s.Require().NoError(err)
	s.False(found)

	// Test search by type
	query = search.NewQueryBuilder().AddStrings(search.ReportType, storage.ReportConfiguration_VULNERABILITY.String()).ProtoQuery()
	results, err := s.datastore.GetReportConfigurations(s.ctx, query)
	s.Require().NoError(err)
	s.Assert().Len(results, 1)

	// Test search all
	query = search.NewQueryBuilder().AddStrings(search.ReportType, search.EmptyQuery().String()).ProtoQuery()
	parsedQuery, err := search.ParseQuery(query.String(), search.MatchAllIfEmpty())
	s.Require().NoError(err)
	results, err = s.datastore.GetReportConfigurations(s.ctx, parsedQuery)
	s.Require().NoError(err)
	s.Assert().Len(results, 1)

	// Test remove
	err = s.datastore.RemoveReportConfiguration(s.ctx, reportConfig.GetId())
	s.Require().NoError(err)

	// Verify empty store
	_, found, err = s.datastore.GetReportConfiguration(s.ctx, reportConfig.GetId())
	s.Require().NoError(err)
	s.False(found)
}

func (s *ReportConfigurationPostgresDatastoreTests) TestMultipleReportNotifiers() {
	s.T().Setenv(features.VulnReportingEnhancements.EnvVar(), "true")

	if !features.VulnReportingEnhancements.Enabled() {
		s.T().Skip("Skip Reporting 2.0 tests")
		s.T().SkipNow()
	}

	reportConfig := fixtures.GetValidReportConfigWithMultipleNotifiersV1()

	// Test add
	_, err := s.datastore.AddReportConfiguration(s.ctx, reportConfig)
	s.Require().NoError(err)

	// Test get
	foundReportConfig, found, err := s.datastore.GetReportConfiguration(s.ctx, reportConfig.GetId())
	s.Require().NoError(err)
	s.True(found)
	s.Equal(reportConfig, foundReportConfig)
}

func (s *ReportConfigurationPostgresDatastoreTests) TestNoNotifiers() {
	s.T().Setenv(features.VulnReportingEnhancements.EnvVar(), "true")

	if !features.VulnReportingEnhancements.Enabled() {
		s.T().Skip("Skip Reporting 2.0 tests")
		s.T().SkipNow()
	}

	reportConfig := fixtures.GetValidReportConfigWithMultipleNotifiersV1()
	reportConfig.Notifiers = nil

	// Test add
	_, err := s.datastore.AddReportConfiguration(s.ctx, reportConfig)
	s.Require().NoError(err)

	// Test get
	foundReportConfig, found, err := s.datastore.GetReportConfiguration(s.ctx, reportConfig.GetId())
	s.Require().NoError(err)
	s.True(found)
	s.Equal(reportConfig, foundReportConfig)
}

func (s *ReportConfigurationPostgresDatastoreTests) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}
