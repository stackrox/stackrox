package scheduler

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/helpers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

// runReportJob is the report job of a due or submitted report: it generates and delivers a
// report for the job's scan configuration from the current datastore state.
func (m *managerImpl) runReportJob(ctx context.Context, job *reportJob) {
	reportJobsFired.WithLabelValues(job.scanConfigName).Inc()

	scanConfig, found, err := m.scanConfigDataStore.GetScanConfiguration(ctx, job.scanConfigID)
	if err != nil {
		log.Errorf("Compliance report scheduler: unable to retrieve scan configuration %q: %v", job.scanConfigID, err)
		return
	}
	if !found {
		// The configuration was deleted after the job was submitted.
		log.Debugf("Compliance report scheduler: scan configuration %q not found, no report generated", job.scanConfigID)
		return
	}
	// Notifier parity with the report manager's automatic path: configurations without
	// notifiers get no scheduled reports, and no snapshot is persisted for them.
	if len(scanConfig.GetNotifiers()) == 0 {
		log.Debugf("Compliance report scheduler: scan configuration %q has no notifiers, no report generated", job.scanConfigName)
		return
	}
	m.generateReport(ctx, scanConfig, job.cycleStart)
}

// generateReport generates and delivers a report for the scan configuration from the current
// datastore state, persisting a SCHEDULED snapshot that the UI renders like any other report.
//
// The sequence composes the same shared building blocks as the report manager's on-demand
// no-watcher path (helpers.* and the report generator): snapshot bookkeeping, failed-cluster
// and cluster-data assembly, then generation and delivery. The report manager itself is left
// untouched: while the rollout feature is disabled, its path remains the behavior to fall
// back to.
func (m *managerImpl) generateReport(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, cycleStart time.Time) {
	snapshot := m.buildScheduledSnapshot(ctx, scanConfig)
	if err := m.snapshotDataStore.UpsertSnapshot(ctx, snapshot); err != nil {
		log.Errorf("Compliance report scheduler: unable to persist the report snapshot of scan configuration %q: %v", scanConfig.GetScanConfigName(), err)
		return
	}

	failedClusters := m.auditExpectations(ctx, scanConfig, cycleStart)
	clusterData, err := helpers.GetClusterData(ctx, snapshot.GetReportData(), failedClusters, m.scanDataStore)
	if err != nil {
		log.Errorf("Compliance report scheduler: unable to get cluster information of scan configuration %q: %v", scanConfig.GetScanConfigName(), err)
		if dbErr := helpers.UpdateSnapshotOnError(ctx, snapshot, report.ErrReportGeneration, m.snapshotDataStore); dbErr != nil {
			log.Errorf("Compliance report scheduler: unable to update the snapshot on generation failure: %v", dbErr)
		}
		return
	}
	if _, err := m.addFailedClustersToSnapshot(ctx, failedClusters, snapshot); err != nil {
		log.Errorf("Compliance report scheduler: unable to record failed clusters on the snapshot of scan configuration %q: %v", scanConfig.GetScanConfigName(), err)
		return
	}

	if err := m.reportGenerator.ProcessReportRequest(&report.Request{
		ScanConfigName:     scanConfig.GetScanConfigName(),
		ScanConfigID:       scanConfig.GetId(),
		Profiles:           profileNames(scanConfig),
		ClusterIDs:         clusterIDs(scanConfig),
		Notifiers:          scanConfig.GetNotifiers(),
		Ctx:                ctx,
		SnapshotID:         snapshot.GetReportId(),
		NotificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
		ClusterData:        clusterData,
		NumFailedClusters:  len(failedClusters),
	}); err != nil {
		log.Errorf("Compliance report scheduler: unable to process the report request of scan configuration %q: %v", scanConfig.GetScanConfigName(), err)
	}
}

// buildScheduledSnapshot returns the snapshot for a scheduled report of the scan
// configuration, mirroring the report manager's automatic snapshots: SCHEDULED request type,
// email notification, attributed to the configuration's last modifier.
func (m *managerImpl) buildScheduledSnapshot(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) *storage.ComplianceOperatorReportSnapshotV2 {
	// A conversion failure is not fatal: the report manager's path logs it and proceeds with
	// no report data, and so does this one.
	reportData, err := helpers.ConvertScanConfigurationToReportData(ctx, scanConfig, m.scanConfigDataStore, m.suiteDataStore, m.bindingsDataStore)
	if err != nil {
		log.Warnf("Compliance report scheduler: unable to convert scan configuration %q to report data: %v", scanConfig.GetScanConfigName(), err)
	}
	return &storage.ComplianceOperatorReportSnapshotV2{
		ReportId:            uuid.NewV4().String(),
		ScanConfigurationId: scanConfig.GetId(),
		Name:                scanConfig.GetScanConfigName(),
		Description:         scanConfig.GetDescription(),
		ReportStatus: &storage.ComplianceOperatorReportStatus{
			RunState:                 storage.ComplianceOperatorReportStatus_PREPARING,
			StartedAt:                protocompat.TimestampNow(),
			ReportRequestType:        storage.ComplianceOperatorReportStatus_SCHEDULED,
			ReportNotificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
		},
		ReportData: reportData,
		User:       scanConfig.GetModifiedBy(),
	}
}

// addFailedClustersToSnapshot records the failed clusters on the snapshot and persists it,
// mirroring the report manager's snapshot bookkeeping so the UI renders the same failures.
func (m *managerImpl) addFailedClustersToSnapshot(ctx context.Context, failedClusters map[string]*report.FailedCluster, snapshot *storage.ComplianceOperatorReportSnapshotV2) (*storage.ComplianceOperatorReportSnapshotV2, error) {
	if len(failedClusters) == 0 {
		return snapshot, nil
	}
	failedClustersSlice := make([]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster, 0, len(failedClusters))
	for _, failedCluster := range failedClusters {
		scans := make([]string, 0, len(failedCluster.FailedScans))
		for _, scan := range failedCluster.FailedScans {
			scans = append(scans, scan.GetScanName())
		}
		failedClustersSlice = append(failedClustersSlice, &storage.ComplianceOperatorReportSnapshotV2_FailedCluster{
			ClusterId:       failedCluster.ClusterId,
			ClusterName:     failedCluster.ClusterName,
			OperatorVersion: failedCluster.OperatorVersion,
			Reasons:         failedCluster.Reasons,
			ScanNames:       scans,
		})
	}
	snapshot.FailedClusters = failedClustersSlice
	if err := m.snapshotDataStore.UpsertSnapshot(ctx, snapshot); err != nil {
		return snapshot, errors.Wrapf(err, "unable to upsert the snapshot %s", snapshot.GetReportId())
	}
	return snapshot, nil
}

func profileNames(scanConfig *storage.ComplianceOperatorScanConfigurationV2) []string {
	profiles := make([]string, 0, len(scanConfig.GetProfiles()))
	for _, profile := range scanConfig.GetProfiles() {
		profiles = append(profiles, profile.GetProfileName())
	}
	return profiles
}

func clusterIDs(scanConfig *storage.ComplianceOperatorScanConfigurationV2) []string {
	clusterIds := make([]string, 0, len(scanConfig.GetClusters()))
	for _, cluster := range scanConfig.GetClusters() {
		clusterIds = append(clusterIds, cluster.GetClusterId())
	}
	return clusterIds
}
