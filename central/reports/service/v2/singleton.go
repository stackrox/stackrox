package v2

import (
	"context"

	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	metadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	metadataStore := metadataDS.Singleton()
	snapshotDatastore := snapshotDS.Singleton()
	scheduler := initializeScheduler(metadataStore)
	svc = New(metadataStore, snapshotDatastore, scheduler)
}

func initializeScheduler(metadataDataStore metadataDS.DataStore) schedulerV2.Scheduler {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))

	scheduler := schedulerV2.Singleton()
	reportConfigDatastore := reportConfigDS.Singleton()

	// Queuing pending and scheduled reports in separate routines to prevent blocking main routine during startup
	go queuePendingReports(ctx, scheduler, metadataDataStore, reportConfigDatastore)
	go queueScheduledReports(ctx, scheduler, reportConfigDatastore)

	scheduler.Start()
	return scheduler
}

func queuePendingReports(ctx context.Context, scheduler schedulerV2.Scheduler,
	metadataStore metadataDS.DataStore, reportConfigDatastore reportConfigDS.DataStore) {
	pendingReportsQuery := search.NewQueryBuilder().
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		WithPagination(search.NewPagination().AddSortOption(search.NewSortOption(search.ReportQueuedTime))).
		ProtoQuery()
	pendingReports, err := metadataStore.SearchReportMetadatas(ctx, pendingReportsQuery)
	if err != nil {
		log.Errorf("Error finding pending reports: %s", err)
		return
	}

	for _, report := range pendingReports {
		reportConfig, found, err := reportConfigDatastore.GetReportConfiguration(ctx, report.GetReportConfigId())
		if err != nil {
			log.Errorf("Error rescheduling pending report for report config ID '%s': %s", report.GetReportConfigId(), err)
			continue
		}
		if !found {
			log.Warnf("Report configuration with ID %s had pending reports but the configuration no longer exists",
				report.GetReportConfigId())
			continue
		}
		_, err = scheduler.SubmitReport(&schedulerV2.ReportRequest{
			ReportConfig:   reportConfig,
			ReportMetadata: report,
		}, true)
		if err != nil {
			log.Errorf("Error rescheduling pending report for report config '%s': %s", report.GetReportConfigId(), err)
		}
	}
}

func queueScheduledReports(ctx context.Context, scheduler schedulerV2.Scheduler,
	reportConfigDatastore reportConfigDS.DataStore) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportType, storage.ReportConfiguration_VULNERABILITY.String()).
		ProtoQuery()
	reportConfigs, err := reportConfigDatastore.GetReportConfigurations(ctx, query)
	if err != nil {
		log.Error("Error finding scheduled reports: %s", err)
	}
	for _, rc := range reportConfigs {
		if rc.GetSchedule() != nil {
			if err := scheduler.UpsertReportSchedule(rc); err != nil {
				log.Errorf("Error queuing scheduled report for report configuration with ID %s: %v", rc.GetId(), err)
			}
		}
	}
}

// Singleton provides the instance of the service to register.
func Singleton() Service {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return svc
}
