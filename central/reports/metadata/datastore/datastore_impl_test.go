//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestReportMetadataDatastore(t *testing.T) {
	suite.Run(t, new(ReportMetadataDatastoreTestSuite))
}

type ReportMetadataDatastoreTestSuite struct {
	suite.Suite

	testDB            *pgtest.TestPostgres
	datastore         DataStore
	reportConfigStore reportConfigDS.DataStore
	ctx               context.Context
}

func (s *ReportMetadataDatastoreTestSuite) SetupSuite() {
	s.T().Setenv(features.VulnMgmtReportingEnhancements.EnvVar(), "true")
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		s.T().Skip("Skip tests when ROX_VULN_MGMT_REPORTING_ENHANCEMENTS disabled")
		s.T().SkipNow()
	}

	var err error
	s.testDB = pgtest.ForT(s.T())
	s.datastore, err = GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.NoError(err)
	s.reportConfigStore, err = reportConfigDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.NoError(err)

	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
}

func (s *ReportMetadataDatastoreTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *ReportMetadataDatastoreTestSuite) TestReportMetadataWorkflows() {
	reportConfig := fixtures.GetValidReportConfigWithMultipleNotifiers()
	reportConfig.Id = ""
	configID, err := s.reportConfigStore.AddReportConfiguration(s.ctx, reportConfig)
	s.NoError(err)

	noAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	// Test AddReportMetadata: error without write access
	report := fixtures.GetReportMetadata()
	report.ReportId = ""
	report.ReportConfigId = configID
	reportID, err := s.datastore.AddReportMetadata(noAccessCtx, report)
	s.Error(err)
	s.Equal("", reportID)

	// Test AddReportMetadata: no error with write access
	reportID, err = s.datastore.AddReportMetadata(s.ctx, report)
	s.NoError(err)

	// Test Get: no result without read access
	report, found, err := s.datastore.Get(noAccessCtx, reportID)
	s.NoError(err)
	s.False(found)
	s.Nil(report)

	// Test Get: returns report with read access
	report, found, err = s.datastore.Get(s.ctx, reportID)
	s.NoError(err)
	s.True(found)
	s.Equal(reportID, report.ReportId)

	// Test UpdateReportMetadata: error without write access
	report.ReportStatus.ReportRequestType = storage.ReportStatus_SCHEDULED
	err = s.datastore.UpdateReportMetadata(noAccessCtx, report)
	s.Error(err)

	// Test UpdateReportMetadata: error with ReportId unset
	reportWithoutID := report.Clone()
	reportWithoutID.ReportId = ""
	err = s.datastore.UpdateReportMetadata(s.ctx, reportWithoutID)
	s.Error(err)

	// Test UpdateReportMetadata: success
	err = s.datastore.UpdateReportMetadata(s.ctx, report)
	s.NoError(err)
	report, found, err = s.datastore.Get(s.ctx, reportID)
	s.NoError(err)
	s.True(found)
	s.Equal(storage.ReportStatus_SCHEDULED, report.ReportStatus.ReportRequestType)

	// Test Search: Without read access
	results, err := s.datastore.Search(noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Nil(results)

	// Test Search: With read access
	results, err = s.datastore.Search(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal(reportID, results[0].ID)

	// Test Search: Search by run state
	failedReport := fixtures.GetReportMetadata()
	failedReport.ReportId = ""
	failedReport.ReportStatus.RunState = storage.ReportStatus_FAILURE
	failedReport.ReportConfigId = configID
	failedreportID, err := s.datastore.AddReportMetadata(s.ctx, failedReport)
	s.NoError(err)

	results, err = s.datastore.Search(s.ctx, search.MatchFieldQuery(search.ReportState.String(), storage.ReportStatus_FAILURE.String(), false))
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal(failedreportID, results[0].ID)

	// Test Count: returns 0 without read access
	count, err := s.datastore.Count(noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(0, count)

	// Test Count: return true count with read access
	count, err = s.datastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(2, count)

	// Test Exists: returns false without read access
	exists, err := s.datastore.Exists(noAccessCtx, reportID)
	s.NoError(err)
	s.False(exists)

	// Test Exists: returns correct value with read access
	exists, err = s.datastore.Exists(s.ctx, reportID)
	s.NoError(err)
	s.True(exists)

	// Test GetMany: returns no reports without read access
	reportIDs := []string{reportID, failedreportID}
	reports, err := s.datastore.GetMany(noAccessCtx, reportIDs)
	s.NoError(err)
	s.Nil(reports)

	// Test GetMany: returns requested reports with read access
	reports, err = s.datastore.GetMany(s.ctx, reportIDs)
	s.NoError(err)
	s.Equal(len(reportIDs), len(reports))

	// Test DeleteReportMetadata: returns error without write access
	err = s.datastore.DeleteReportMetadata(noAccessCtx, reportID)
	s.Error(err)

	// Test DeleteReportMetadata: successfully deletes with write access
	err = s.datastore.DeleteReportMetadata(s.ctx, reportID)
	s.NoError(err)
	report, found, err = s.datastore.Get(s.ctx, reportID)
	s.NoError(err)
	s.False(found)
	s.Nil(report)
}
