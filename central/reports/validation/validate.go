package validation

import (
	"context"

	"github.com/pkg/errors"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
)

var allAccessCtx = sac.WithAllAccess(context.Background())

// ValidateAndGenerateReportRequest validates the report configuration for which report is requested and generates a report request
func ValidateAndGenerateReportRequest(reportConfigStore reportConfigDS.DataStore, collectionStore collectionDS.DataStore,
	notifierStore notifierDS.DataStore, configID string, requester *storage.SlimUser,
	notificationMethod storage.ReportStatus_NotificationMethod,
	requestType storage.ReportStatus_RunMethod) (*reportGen.ReportRequest, error) {
	config, found, err := reportConfigStore.GetReportConfiguration(allAccessCtx, configID)
	if err != nil {
		return nil, errors.Wrapf(err, "Error finding report configuration %s", configID)
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Report configuration id not found %s", configID)
	}

	collection, found, err := collectionStore.Get(allAccessCtx, config.GetResourceScope().GetCollectionId())
	if err != nil {
		return nil, errors.Wrapf(err, "Error finding collection ID '%s'", config.GetResourceScope().GetCollectionId())
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Collection ID '%s' not found", config.GetResourceScope().GetCollectionId())
	}

	notifierIDs := make([]string, 0, len(config.GetNotifiers()))
	for _, notifierConf := range config.GetNotifiers() {
		notifierIDs = append(notifierIDs, notifierConf.GetEmailConfig().GetNotifierId())
	}
	protoNotifiers, err := notifierStore.GetManyNotifiers(allAccessCtx, notifierIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Error finding attached notifiers")
	}
	if len(protoNotifiers) != len(notifierIDs) {
		return nil, errors.Wrap(errox.NotFound, "Some of the attached notifiers not found")
	}

	return &reportGen.ReportRequest{
		ReportConfig:   config,
		Collection:     collection,
		ReportSnapshot: generateReportSnapshot(config, requester, collection, protoNotifiers, notificationMethod, requestType),
	}, nil
}

// ValidateCancelReportRequest validates if the given requester can cancel the report job with job ID = reportID.
func ValidateCancelReportRequest(reportSnapshotStore reportSnapshotDS.DataStore, reportID string, requester *storage.SlimUser) error {
	snapshot, found, err := reportSnapshotStore.Get(allAccessCtx, reportID)
	if err != nil {
		return errors.Wrapf(err, "Error finding report snapshot with job ID '%s'.", reportID)
	}
	if !found {
		return errors.Wrapf(errox.NotFound, "Report snapshot with job ID '%s' does not exist", reportID)
	}

	runState := snapshot.GetReportStatus().GetRunState()
	if runState == storage.ReportStatus_SUCCESS || runState == storage.ReportStatus_FAILURE {
		return errors.Wrapf(errox.InvalidArgs, "Cannot cancel. Report job ID '%s' has already completed execution.", reportID)
	} else if runState == storage.ReportStatus_PREPARING {
		return errors.Wrapf(errox.InvalidArgs, "Cannot cancel. Report job ID '%s' is currently being prepared.", reportID)
	}

	if requester.GetId() != snapshot.GetRequester().GetId() {
		return errors.Wrap(errox.NotAuthorized, "Report job cannot be cancelled by a user who did not request the report.")
	}
	return nil
}

func generateReportSnapshot(config *storage.ReportConfiguration, requester *storage.SlimUser,
	collection *storage.ResourceCollection, protoNotifiers []*storage.Notifier,
	notificationMethod storage.ReportStatus_NotificationMethod,
	requestType storage.ReportStatus_RunMethod) *storage.ReportSnapshot {
	snapshot := &storage.ReportSnapshot{
		ReportConfigurationId: config.GetId(),
		Name:                  config.GetName(),
		Description:           config.GetDescription(),
		Type:                  storage.ReportSnapshot_VULNERABILITY,
		Filter: &storage.ReportSnapshot_VulnReportFilters{
			VulnReportFilters: config.GetVulnReportFilters(),
		},
		Collection: &storage.CollectionSnapshot{
			Id:   config.GetResourceScope().GetCollectionId(),
			Name: collection.GetName(),
		},
		Schedule: config.GetSchedule(),
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_WAITING,
			ReportRequestType:        requestType,
			ReportNotificationMethod: notificationMethod,
		},
		Requester: requester,
	}

	notifierSnaps := make([]*storage.NotifierSnapshot, 0, len(config.GetNotifiers()))

	for i, notifierConf := range config.GetNotifiers() {
		notifierSnaps = append(notifierSnaps, &storage.NotifierSnapshot{
			NotifierConfig: &storage.NotifierSnapshot_EmailConfig{
				EmailConfig: notifierConf.GetEmailConfig(),
			},
			NotifierName: protoNotifiers[i].GetName(),
		})
	}
	snapshot.Notifiers = notifierSnaps
	return snapshot
}
