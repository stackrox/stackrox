package reportgenerator

import (
	"bytes"
	"context"
	"time"

	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/utils"
)

// RetryableSendReportResults sends report results via the notifier with retry logic.
func RetryableSendReportResults(ctx context.Context, reportNotifier notifiers.ReportNotifier, mailingList []string,
	zippedCSVData *bytes.Buffer, emailSubject, emailBody, baseFilename string) error {
	return retry.WithRetry(func() error {
		return reportNotifier.ReportNotify(ctx, zippedCSVData, mailingList, emailSubject, emailBody, baseFilename)
	},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

// SaveReportData saves zipped CSV report data to blob storage.
func SaveReportData(ctx context.Context, blobStore blobDS.Datastore, configID, reportID string, data *bytes.Buffer) error {
	if data == nil {
		return errors.Errorf("No data found for report config %q and id %q", configID, reportID)
	}
	b := &storage.Blob{
		Name:         common.GetReportBlobPath(configID, reportID),
		LastUpdated:  protocompat.TimestampNow(),
		ModifiedTime: protocompat.TimestampNow(),
		Length:       int64(data.Len()),
	}
	return blobStore.Upsert(ctx, b, data)
}

// UpdateReportStatus updates the run state of a report snapshot.
func UpdateReportStatus(ctx context.Context, snapshotStore reportSnapshotDS.DataStore, snapshot *storage.ReportSnapshot, status storage.ReportStatus_RunState) error {
	snapshot.ReportStatus.RunState = status
	return snapshotStore.UpdateReportSnapshot(ctx, snapshot)
}

// LogAndUpsertError logs the error and updates the report snapshot status to FAILURE.
// requestCtx is used to check for user cancellation; statusCtx is used for the DB update
// (should be an all-access context since requestCtx may be cancelled).
func LogAndUpsertError(requestCtx, statusCtx context.Context, snapshotStore reportSnapshotDS.DataStore, reportErr error, req *ReportRequest) {
	if req.ReportSnapshot == nil || req.ReportSnapshot.GetReportStatus() == nil {
		utils.Should(errors.New("Request does not have non-nil report snapshot with a non-nil report status"))
		return
	}
	if errors.Is(context.Cause(requestCtx), ErrUserCancelled) {
		log.Infof("Report for config '%s' was cancelled by user", req.ReportSnapshot.GetName())
		req.ReportSnapshot.ReportStatus.ErrorMsg = ErrUserCancelled.Error()
	} else if reportErr != nil {
		log.Errorf("Error while running report for config '%s': %s", req.ReportSnapshot.GetName(), reportErr)
		req.ReportSnapshot.ReportStatus.ErrorMsg = reportErr.Error()
	}
	req.ReportSnapshot.ReportStatus.CompletedAt = protocompat.TimestampNow()
	if err := UpdateReportStatus(statusCtx, snapshotStore, req.ReportSnapshot, storage.ReportStatus_FAILURE); err != nil {
		log.Errorf("Error changing report status to FAILURE for report config '%s', report ID '%s': %s",
			req.ReportSnapshot.GetName(), req.ReportSnapshot.GetReportId(), err)
	}
}
