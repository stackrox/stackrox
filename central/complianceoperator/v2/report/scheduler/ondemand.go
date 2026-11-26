package scheduler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/helpers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
)

// SubmitReportRequest submits an on-demand report execution request for the scan
// configuration.
//
// The report snapshot is persisted in the WAITING state immediately -- the UI shows it right
// away -- and the report job runs through the same execution loop as scheduled reports,
// anchored at the cycle that most recently ticked. This is the "on-demand reduces to the same
// primitive" convergence of the decoupled reporting redesign: settle, audit, snapshot.
//
// A second request for a configuration with a pending on-demand report is rejected, mirroring
// the report manager. A request whose job never runs (the scheduler could not start on this
// instance) leaves a pending snapshot that the snapshot store's startup purge removes, like
// every other interrupted report.
func (m *managerImpl) SubmitReportRequest(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, method storage.ComplianceOperatorReportStatus_NotificationMethod) error {
	if !m.enabled() {
		return nil
	}
	requesterID := authn.IdentityFromContextOrNil(ctx)
	if requesterID == nil {
		return errors.New("could not determine user identity from provided context")
	}
	// Email delivery requires notifiers; downloads do not.
	if method == storage.ComplianceOperatorReportStatus_EMAIL && len(scanConfig.GetNotifiers()) == 0 {
		return report.ErrNoNotifiersConfigured
	}
	// One pending on-demand report per configuration.
	if !concurrency.WithLock1(&m.onDemandLock, func() bool {
		return m.onDemandPending.Add(scanConfig.GetId())
	}) {
		return errors.Errorf("report request for scan configuration %q already in process", scanConfig.GetScanConfigName())
	}

	snapshot := m.buildOnDemandSnapshot(ctx, scanConfig, method, requesterID)
	if err := m.snapshotDataStore.UpsertSnapshot(ctx, snapshot); err != nil {
		concurrency.WithLock(&m.onDemandLock, func() {
			m.onDemandPending.Remove(scanConfig.GetId())
		})
		return errors.Wrap(err, "unable to upsert snapshot on report submission")
	}
	// The queue buffers the job until the execution loop runs: a request racing the
	// scheduler's startup simply waits the few milliseconds for it.
	m.submitJob(&reportJob{
		scanConfigID:   scanConfig.GetId(),
		scanConfigName: scanConfig.GetScanConfigName(),
		snapshotID:     snapshot.GetReportId(),
	})
	return nil
}

// buildOnDemandSnapshot returns the WAITING snapshot for the on-demand report, attributed to
// the requester. A report-data conversion failure is not fatal: the report manager's path logs
// it and proceeds with no report data, and so does this one.
func (m *managerImpl) buildOnDemandSnapshot(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, method storage.ComplianceOperatorReportStatus_NotificationMethod, requesterID authn.Identity) *storage.ComplianceOperatorReportSnapshotV2 {
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
			RunState:                 storage.ComplianceOperatorReportStatus_WAITING,
			StartedAt:                protocompat.TimestampNow(),
			ReportRequestType:        storage.ComplianceOperatorReportStatus_ON_DEMAND,
			ReportNotificationMethod: method,
		},
		ReportData: reportData,
		User: &storage.SlimUser{
			Id:   requesterID.UID(),
			Name: stringutils.FirstNonEmpty(requesterID.FullName(), requesterID.FriendlyName()),
		},
	}
}
