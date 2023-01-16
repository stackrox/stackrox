package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/reportconfigurations/index"
	reportConfigurationSearch "github.com/stackrox/rox/central/reportconfigurations/search"
	store "github.com/stackrox/rox/central/reportconfigurations/store/rocksdb"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestReportConfigurationDatastore(t *testing.T) {
	suite.Run(t, new(ReportConfigurationDatastoreTestSuite))
}

type ReportConfigurationDatastoreTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

	db *rocksdb.RocksDB

	indexer           index.Indexer
	searcher          reportConfigurationSearch.Searcher
	reportConfigStore store.Store
	datastore         DataStore

	hasReadWriteVulnReportAccess             context.Context
	hasReadWriteWorkflowAdministrationAccess context.Context
}

func (suite *ReportConfigurationDatastoreTestSuite) SetupSuite() {
	var err error
	suite.bleveIndex, err = globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	db, err := rocksdb.NewTemp(suite.T().Name() + ".db")
	suite.Require().NoError(err)

	suite.db = db

	suite.reportConfigStore, err = store.New(db)
	suite.Require().NoError(err)
	suite.Require().NoError(err)

	suite.indexer = index.New(suite.bleveIndex)
	suite.searcher = reportConfigurationSearch.New(suite.reportConfigStore, suite.indexer)
	suite.datastore, err = New(suite.reportConfigStore, suite.indexer, suite.searcher)
	suite.Require().NoError(err)

	suite.hasReadWriteVulnReportAccess = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			// TODO: ROX-13888 Replace VulnerabilityReports with WorkflowAdministration.
			sac.ResourceScopeKeys(resources.VulnerabilityReports)))

	suite.hasReadWriteWorkflowAdministrationAccess = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			// TODO: ROX-13888 Remove this duplicated context.
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
}

func (suite *ReportConfigurationDatastoreTestSuite) TearDownSuite() {
	suite.NoError(suite.bleveIndex.Close())
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *ReportConfigurationDatastoreTestSuite) TestReportsConfigDataStore() {
	reportConfig := fixtures.GetValidReportConfiguration()
	_, err := suite.datastore.AddReportConfiguration(suite.hasReadWriteVulnReportAccess, reportConfig)
	suite.Require().NoError(err)

	foundReportConfig, found, err := suite.datastore.GetReportConfiguration(suite.hasReadWriteVulnReportAccess, reportConfig.GetId())
	suite.Require().NoError(err)
	suite.True(found)
	suite.Equal(reportConfig, foundReportConfig)

	// TODO: ROX-13888 Remove this duplicated test.
	// Expect no error when trying to retrieve the report with the replacing resource WorkflowAdministration.
	foundReportConfig, found, err = suite.datastore.GetReportConfiguration(
		suite.hasReadWriteWorkflowAdministrationAccess, reportConfig.GetId())
	suite.Require().NoError(err)
	suite.True(found)
	suite.Equal(reportConfig, foundReportConfig)

	_, found, err = suite.datastore.GetReportConfiguration(suite.hasReadWriteVulnReportAccess, "NONEXISTENT")
	suite.Require().NoError(err)
	suite.False(found)

	query := search.NewQueryBuilder().AddStrings(search.ReportType, storage.ReportConfiguration_VULNERABILITY.String()).ProtoQuery()
	results, err := suite.datastore.GetReportConfigurations(suite.hasReadWriteVulnReportAccess, query)
	suite.Require().NoError(err)
	suite.Assert().Len(results, 1)

	// TODO: ROX-13888 Remove this duplicated section.
	// Expect no error when trying to retrieve the report with the replacing resource WorkflowAdministration.
	query = search.NewQueryBuilder().AddStrings(search.ReportType, storage.ReportConfiguration_VULNERABILITY.String()).ProtoQuery()
	results, err = suite.datastore.GetReportConfigurations(suite.hasReadWriteWorkflowAdministrationAccess, query)
	suite.Require().NoError(err)
	suite.Assert().Len(results, 1)

	query = search.NewQueryBuilder().AddStrings(search.ReportType, search.EmptyQuery().String()).ProtoQuery()
	parsedQuery, err := search.ParseQuery(query.String(), search.MatchAllIfEmpty())
	suite.Require().NoError(err)
	results, err = suite.datastore.GetReportConfigurations(suite.hasReadWriteVulnReportAccess, parsedQuery)
	suite.Require().NoError(err)
	suite.Assert().Len(results, 1)

	err = suite.datastore.RemoveReportConfiguration(suite.hasReadWriteVulnReportAccess, reportConfig.GetId())
	suite.Require().NoError(err)

	_, found, err = suite.datastore.GetReportConfiguration(suite.hasReadWriteVulnReportAccess, reportConfig.GetId())
	suite.Require().NoError(err)
	suite.False(found)

	// TODO: ROX-13888 Remove this duplicated section.
	// Expect no error when upserting and deleting a report configuration with the replacing resource WorkflowAdministration.
	reportConfig = fixtures.GetValidReportConfiguration()
	_, err = suite.datastore.AddReportConfiguration(suite.hasReadWriteWorkflowAdministrationAccess, reportConfig)
	suite.Require().NoError(err)

	err = suite.datastore.RemoveReportConfiguration(suite.hasReadWriteWorkflowAdministrationAccess, reportConfig.GetId())
	suite.Require().NoError(err)

	_, found, err = suite.datastore.GetReportConfiguration(suite.hasReadWriteWorkflowAdministrationAccess, reportConfig.GetId())
	suite.Require().NoError(err)
	suite.False(found)
}
