//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestReportConfigurationDatastoreV2(t *testing.T) {
	suite.Run(t, new(ReportConfigurationDatastoreV2Tests))
}

type ReportConfigurationDatastoreV2Tests struct {
	suite.Suite

	testDB                  *pgtest.TestPostgres
	datastore               DataStore
	reportSnapshotDataStore reportSnapshotDS.DataStore
	notifierDataStore       notifierDS.DataStore
	ctx                     context.Context
}

func (s *ReportConfigurationDatastoreV2Tests) SetupSuite() {
	s.T().Setenv(features.VulnReportingEnhancements.EnvVar(), "true")
	if !features.VulnReportingEnhancements.Enabled() {
		s.T().Skip("Skip tests when ROX_VULN_MGMT_REPORTING_ENHANCEMENTS disabled")
		s.T().SkipNow()
	}

	s.testDB = pgtest.ForT(s.T())
	s.datastore = GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.reportSnapshotDataStore = reportSnapshotDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.notifierDataStore = notifierDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)

	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
}

func (s *ReportConfigurationDatastoreV2Tests) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *ReportConfigurationDatastoreV2Tests) TestSortReportConfigByCompletionTime() {
	reportConfig1 := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	reportConfig1.Id = ""
	reportConfig1.ResourceScope = &storage.ResourceScope{
		ScopeReference: &storage.ResourceScope_CollectionId{
			CollectionId: "collection-1",
		},
	}

	// Add all required notifiers to the database.
	for i, n := range reportConfig1.GetNotifiers() {
		reportConfig1.Notifiers[i].Ref = s.storeNotifier(n.GetId())
	}

	configID1, err := s.datastore.AddReportConfiguration(s.ctx, reportConfig1)
	s.NoError(err)

	reportConfig2 := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	reportConfig2.Id = ""
	reportConfig2.ResourceScope = &storage.ResourceScope{
		ScopeReference: &storage.ResourceScope_CollectionId{
			CollectionId: "collection-2",
		},
	}
	for i := range reportConfig2.GetNotifiers() {
		reportConfig2.Notifiers[i] = reportConfig1.Notifiers[i]
	}

	configID2, err := s.datastore.AddReportConfiguration(s.ctx, reportConfig2)
	s.NoError(err)

	reportConfig3 := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	reportConfig3.Id = ""
	reportConfig3.ResourceScope = &storage.ResourceScope{
		ScopeReference: &storage.ResourceScope_CollectionId{
			CollectionId: "collection-2",
		},
	}
	for i := range reportConfig2.GetNotifiers() {
		reportConfig3.Notifiers[i] = reportConfig1.Notifiers[i]
	}

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

	reportSnapshots := []*storage.ReportSnapshot{
		generateReportSnapshot(configID3, time1),
		generateReportSnapshot(configID2, time2),
		generateReportSnapshot(configID2, time3),
		generateReportSnapshot(configID1, time4),
		generateReportSnapshot(configID3, time5),
		generateReportSnapshot(configID1, time6),
	}

	for _, snap := range reportSnapshots {
		_, err = s.reportSnapshotDataStore.AddReportSnapshot(s.ctx, snap)
		s.NoError(err)
	}

	// Test query with report metadata fields
	query1 := search.NewQueryBuilder().
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		WithPagination(search.NewPagination().
			AddSortOption(search.NewSortOption(search.ReportCompletionTime).Reversed(true))).ProtoQuery()

	expectedSortedConfigIDs := []string{configID3, configID2, configID1}
	configs, err := s.datastore.GetReportConfigurations(s.ctx, query1)
	s.NoError(err)
	s.Equal(len(expectedSortedConfigIDs), len(configs))

	configIDs := make([]string, 0, len(configs))
	for _, conf := range configs {
		configIDs = append(configIDs, conf.Id)
	}
	s.Equal(expectedSortedConfigIDs, configIDs)

	// Test a query with combination of report config and report metadata fields
	query2 := search.NewQueryBuilder().
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		AddExactMatches(search.CollectionID, "collection-2").
		WithPagination(search.NewPagination().
			AddSortOption(search.NewSortOption(search.ReportCompletionTime).Reversed(true))).ProtoQuery()

	expectedSortedConfigIDs = []string{configID3, configID2}
	configs, err = s.datastore.GetReportConfigurations(s.ctx, query2)
	s.NoError(err)
	s.Equal(len(expectedSortedConfigIDs), len(configs))

	configIDs = make([]string, 0, len(configs))
	for _, conf := range configs {
		configIDs = append(configIDs, conf.Id)
	}
	s.Equal(expectedSortedConfigIDs, configIDs)
}

func (s *ReportConfigurationDatastoreV2Tests) storeNotifier(name string) *storage.NotifierConfiguration_Id {
	allCtx := sac.WithAllAccess(context.Background())

	id, err := s.notifierDataStore.AddNotifier(allCtx, &storage.Notifier{Name: name})
	s.Require().NoError(err)
	return &storage.NotifierConfiguration_Id{Id: id}
}

func generateReportSnapshot(configID string, completionTime *types.Timestamp) *storage.ReportSnapshot {
	metadata := fixtures.GetReportSnapshot()
	metadata.ReportStatus.CompletedAt = completionTime
	metadata.ReportConfigurationId = configID

	return metadata
}
