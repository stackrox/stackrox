package generator

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	benchmarksDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	remediationDS "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	snapshotDataStore "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	reportUtils "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/helpers"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/sender"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/central/reports/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	reportGenCtx = sac.WithAllAccess(context.Background())
)

const (
	defaultNumberOfTriesOnEmailSend = 3
)

type stoppable[T any] interface {
	Stop()
}

type newResponseHandler[T any] func(func(T) error, func(), <-chan T) (sender.AsyncResponseHandler[T], error)

type complianceReportGeneratorImpl struct {
	checkResultsDS           checkResults.DataStore
	notificationProcessor    notifier.Processor
	scanDS                   scanDS.DataStore
	profileDS                profileDS.DataStore
	remediationDS            remediationDS.DataStore
	benchmarkDS              benchmarksDS.DataStore
	complianceRuleDS         complianceRuleDS.DataStore
	snapshotDS               snapshotDataStore.DataStore
	blobStore                blobDS.Datastore
	numberOfTriesOnEmailSend int

	resultsAggregator ResultsAggregator
	formatter         Formatter
	reportSender      ReportSender

	handlersMutex          sync.Mutex
	senderResponseHandlers map[string]stoppable[error]
	newHandlerFn           newResponseHandler[error]
}

func (rg *complianceReportGeneratorImpl) ProcessReportRequest(req *report.Request) error {

	log.Infof("Processing report request %s", req)

	var snapshot *storage.ComplianceOperatorReportSnapshotV2
	if req.SnapshotID != "" {
		var found bool
		var err error
		snapshot, found, err = rg.snapshotDS.GetSnapshot(req.Ctx, req.SnapshotID)
		if err != nil {
			return errors.Wrap(err, "unable to retrieve the snapshot from the store")
		}
		if !found {
			return errors.New("unable to find snapshot in the store")
		}
	}

	reportData := rg.resultsAggregator.GetReportData(req)

	zipData, err := rg.formatter.FormatCSVReport(reportData.ResultCSVs, req.FailedClusters)
	if err != nil {
		if dbErr := reportUtils.UpdateSnapshotOnError(req.Ctx, snapshot, report.ErrReportGeneration, rg.snapshotDS); dbErr != nil {
			return errors.Wrap(dbErr, "unable to update the snapshot on report generation failure")
		}
		return errors.Wrapf(err, "unable to zip the compliance reports for scan config %s", req.ScanConfigName)
	}

	if snapshot != nil {
		snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_GENERATED
		if len(req.FailedClusters) > 0 {
			snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_PARTIAL_ERROR
			if len(req.FailedClusters) == len(req.ClusterIDs) {
				snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_FAILURE
			}
		}
		if err := rg.snapshotDS.UpsertSnapshot(req.Ctx, snapshot); err != nil {
			return errors.Wrap(err, "unable to update snapshot on report generation success")
		}

		if req.NotificationMethod == storage.ComplianceOperatorReportStatus_DOWNLOAD {
			if err := rg.saveReportData(req.Ctx, snapshot.GetScanConfigurationId(), snapshot.GetReportId(), zipData); err != nil {
				if dbErr := reportUtils.UpdateSnapshotOnError(req.Ctx, snapshot, err, rg.snapshotDS); dbErr != nil {
					return errors.Wrap(err, "unable to update snapshot on download failure upsert")
				}
				return errors.Wrap(err, "unable to save the report download")
			}
			snapshot.GetReportStatus().CompletedAt = protocompat.TimestampNow()
			if err := rg.snapshotDS.UpsertSnapshot(req.Ctx, snapshot); err != nil {
				return errors.Wrap(err, "unable to update snapshot on report download ready")
			}
			return nil
		}
	}
	reportName := req.ScanConfigName

	log.Infof("Sending email for scan config %s", reportName)
	errC := rg.reportSender.SendEmail(reportGenCtx, reportName, zipData, reportData, req.Notifiers)
	handler, err := rg.newHandlerFn(
		rg.getOnAsyncSenderSuccess(req.Ctx, snapshot),
		rg.getOnAsyncSenderError(req.Ctx, snapshot),
		errC)
	if err != nil {
		// we should never get here as NewAsyncResponseHandler will only return an error if we pass nil callbacks
		log.Errorf("unable to create the async response handler for %s", snapshot.GetReportId())
		if dbErr := reportUtils.UpdateSnapshotOnError(req.Ctx, snapshot, err, rg.snapshotDS); dbErr != nil {
			return errors.Wrap(err, "unable to update snapshot on failure upsert")
		}
	}
	concurrency.WithLock(&rg.handlersMutex, func() {
		rg.senderResponseHandlers[snapshot.GetReportId()] = handler
	})
	handler.Start()
	return nil
}

func (rg *complianceReportGeneratorImpl) Stop() {
	concurrency.WithLock(&rg.handlersMutex, func() {
		for _, stopper := range rg.senderResponseHandlers {
			stopper.Stop()
		}
	})
}

func (rg *complianceReportGeneratorImpl) getOnAsyncSenderSuccess(ctx context.Context, snapshot *storage.ComplianceOperatorReportSnapshotV2) func(error) error {
	return func(err error) error {
		concurrency.WithLock(&rg.handlersMutex, func() {
			delete(rg.senderResponseHandlers, snapshot.GetReportId())
		})
		if err != nil {
			return err
		}
		if snapshot.GetReportStatus().GetRunState() == storage.ComplianceOperatorReportStatus_GENERATED {
			snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_DELIVERED
		}
		snapshot.GetReportStatus().CompletedAt = protocompat.TimestampNow()
		if dbErr := rg.snapshotDS.UpsertSnapshot(ctx, snapshot); dbErr != nil {
			log.Errorf("Unable to update snapshot on send email success: %v", dbErr)
		}
		return err
	}
}

func (rg *complianceReportGeneratorImpl) getOnAsyncSenderError(ctx context.Context, snapshot *storage.ComplianceOperatorReportSnapshotV2) func() {
	return func() {
		concurrency.WithLock(&rg.handlersMutex, func() {
			delete(rg.senderResponseHandlers, snapshot.GetReportId())
		})
		if dbErr := reportUtils.UpdateSnapshotOnError(ctx, snapshot, report.ErrSendingEmail, rg.snapshotDS); dbErr != nil {
			log.Errorf("Unable to update snapshot on send email failure: %v", dbErr)
		}
	}
}

func (rg *complianceReportGeneratorImpl) saveReportData(ctx context.Context, configID, snapshotID string, data *bytes.Buffer) error {
	if data == nil {
		return errors.Errorf("no data found for snapshot %s and scan configuration %s", snapshotID, configID)
	}

	b := &storage.Blob{
		Name:         common.GetComplianceReportBlobPath(configID, snapshotID),
		LastUpdated:  protocompat.TimestampNow(),
		ModifiedTime: protocompat.TimestampNow(),
		Length:       int64(data.Len()),
	}
	return rg.blobStore.Upsert(ctx, b, data)
}
