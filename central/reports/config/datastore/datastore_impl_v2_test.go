//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDatastore "github.com/stackrox/rox/central/resourcecollection/datastore"
	collectionSearch "github.com/stackrox/rox/central/resourcecollection/datastore/search"
	collectionPgStore "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
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
	collectionDS            collectionDatastore.DataStore
}

func (s *ReportConfigurationDatastoreV2Tests) SetupSuite() {
	s.T().Setenv(env.VulnReportingEnhancements.EnvVar(), "true")
	if !env.VulnReportingEnhancements.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_VULN_MGMT_REPORTING_ENHANCEMENTS disabled")
		s.T().SkipNow()
	}

	s.testDB = pgtest.ForT(s.T())
	s.datastore = GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.reportSnapshotDataStore = reportSnapshotDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.notifierDataStore = notifierDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.datastore, err = GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.NoError(err)
	s.reportSnapshotStore = reportSnapshotDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	storageCollection := collectionPgStore.New(s.testDB.DB)
	indexer := collectionPgStore.NewIndexer(s.testDB.DB)
	s.collectionDS, _, _ = collectionDatastore.New(storageCollection, collectionSearch.New(storageCollection, indexer))

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

	s.addCollectionToReportConfiguration(reportConfig1, "")
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

	collectionID := s.addCollectionToReportConfiguration(reportConfig2, "")
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
	s.addCollectionToReportConfiguration(reportConfig3, collectionID)
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
		AddExactMatches(search.CollectionID, collectionID).
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

func (s *ReportConfigurationDatastoreV2Tests) addCollectionToReportConfiguration(reportConfig *storage.ReportConfiguration, collectionID string) string {
	if collectionID != "" {
		reportConfig.ResourceScope = &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_CollectionId{
				CollectionId: collectionID,
			},
		}
		return collectionID
	}
	collection := storage.ResourceCollection{
		Name: " Test Collection" + uuid.NewV4().String(),
	}
	err := s.collectionDS.AddCollection(s.ctx, &collection)
	s.Require().NoError(err)
	query := search.NewQueryBuilder().AddExactMatches(search.CollectionName, collection.GetName()).ProtoQuery()
	collections, err := s.collectionDS.SearchCollections(s.ctx, query)
	s.Require().NoError(err)
	newCollectionID := collections[0].GetId()
	reportConfig.ResourceScope = &storage.ResourceScope{
		ScopeReference: &storage.ResourceScope_CollectionId{
			CollectionId: newCollectionID,
		},
	}
	return newCollectionID
}

func generateReportSnapshot(configID string, completionTime *types.Timestamp) *storage.ReportSnapshot {
	metadata := fixtures.GetReportSnapshot()
	metadata.ReportStatus.CompletedAt = completionTime
	metadata.ReportConfigurationId = configID

	return metadata
}
