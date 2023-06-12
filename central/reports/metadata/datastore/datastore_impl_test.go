package datastore

import (
	"context"
	"fmt"
	"testing"

	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
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
	var err error
	s.testDB = pgtest.ForT(s.T())
	s.datastore, err = GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.NoError(err)
	s.reportConfigStore, err = reportConfigDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.NoError(err)

	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
}

func (s *ReportMetadataDatastoreTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *ReportMetadataDatastoreTestSuite) TearDownTest() {
	s.truncateTable(postgresSchema.ReportMetadatasTableName)
	s.truncateTable(postgresSchema.ReportConfigurationsTableName)
}

//func (s *ReportMetadataDatastoreTestSuite) TestSearch() {
//	reportConfig := fixtures.GetValidReportConfigWithMultipleNotifiers()
//	s.reportConfigStore.AddReportConfiguration(s.ctx)
//}

func (s *ReportMetadataDatastoreTestSuite) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}
