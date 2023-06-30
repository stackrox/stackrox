//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	metadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestReportConfigurationDatastoreV2(t *testing.T) {
	suite.Run(t, new(ReportConfigurationDatastoreV2Tests))
}

type ReportConfigurationDatastoreV2Tests struct {
	suite.Suite

	testDB              *pgtest.TestPostgres
	datastore           DataStore
	reportMetadataStore metadataDS.DataStore
	ctx                 context.Context
}

func (s *ReportConfigurationDatastoreV2Tests) SetupSuite() {
	s.T().Setenv(features.VulnMgmtReportingEnhancements.EnvVar(), "true")
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		s.T().Skip("Skip tests when ROX_VULN_MGMT_REPORTING_ENHANCEMENTS disabled")
		s.T().SkipNow()
	}

	var err error
	s.testDB = pgtest.ForT(s.T())
	s.datastore, err = GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.NoError(err)
	s.reportMetadataStore, err = metadataDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.NoError(err)

	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
}

func (s *ReportConfigurationDatastoreV2Tests) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *ReportConfigurationDatastoreV2Tests) TestSortReportConfigByCompletionTime() {
	reportConfig1 := fixtures.GetValidReportConfigWithMultipleNotifiers()
	reportConfig1.Id = ""
	configID1, err := s.datastore.AddReportConfiguration(s.ctx, reportConfig1)
	s.NoError(err)

	reportConfig2 := fixtures.GetValidReportConfigWithMultipleNotifiers()
	reportConfig2.Id = ""
	configID2, err := s.datastore.AddReportConfiguration(s.ctx, reportConfig2)
	s.NoError(err)

	reportConfig3 := fixtures.GetValidReportConfigWithMultipleNotifiers()
	reportConfig3.Id = ""
	configID3, err := s.datastore.AddReportConfiguration(s.ctx, reportConfig3)
	s.NoError(err)

	// time1 is the most recent, time6 is the least recent
	time1, err := types.TimestampProto(time.Now().Add(-1 * time.Hour))
	s.NoError(err)
	time2, err := types.TimestampProto(time.Now().Add(-2 * time.Hour))
	s.NoError(err)
	time3, err := types.TimestampProto(time.Now().Add(-3 * time.Hour))
	s.NoError(err)
	time4, err := types.TimestampProto(time.Now().Add(-4 * time.Hour))
	s.NoError(err)
	time5, err := types.TimestampProto(time.Now().Add(-5 * time.Hour))
	s.NoError(err)
	time6, err := types.TimestampProto(time.Now().Add(-6 * time.Hour))
	s.NoError(err)

	reportMetadatas := []*storage.ReportMetadata{
		generateReportMetadata(configID3, time1),
		generateReportMetadata(configID2, time2),
		generateReportMetadata(configID2, time3),
		generateReportMetadata(configID1, time4),
		generateReportMetadata(configID3, time5),
		generateReportMetadata(configID1, time6),
	}
	expectedSortedConfigIDs := []string{configID3, configID2, configID1}

	for _, metadata := range reportMetadatas {
		_, err = s.reportMetadataStore.AddReportMetadata(s.ctx, metadata)
		s.NoError(err)
	}

	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportState, storage.ReportStatus_SUCCESS.String(), storage.ReportStatus_FAILURE.String()).
		WithPagination(search.NewPagination().
			AddSortOption(search.NewSortOption(search.ReportCompletionTime).Reversed(true))).ProtoQuery()

	configs, err := s.datastore.GetReportConfigurations(s.ctx, query)
	s.NoError(err)
	s.Equal(3, len(configs))

	configIDs := make([]string, 0, len(configs))
	for _, conf := range configs {
		configIDs = append(configIDs, conf.Id)
	}
	s.Equal(expectedSortedConfigIDs, configIDs)
}

func generateReportMetadata(configID string, completionTime *types.Timestamp) *storage.ReportMetadata {
	metadata := fixtures.GetReportMetadata()
	metadata.ReportId = ""
	metadata.ReportStatus.CompletedAt = completionTime
	metadata.ReportConfigId = configID

	return metadata
}
