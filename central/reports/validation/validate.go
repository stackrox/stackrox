package validation

import (
	"context"
	"net/mail"

	"github.com/pkg/errors"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	// CustomEmailSubjectMaxLen is the maximum allowed length for custom email subject
	CustomEmailSubjectMaxLen = 256
	// CustomEmailBodyMaxLen is the maximum allowed length for custom email body
	CustomEmailBodyMaxLen = 1500
)

// Use this context only to
// 1) check if notifiers and collection attached to report config exist
// 2) Populating notifiers and collection in report snapshot
var allAccessCtx = sac.WithAllAccess(context.Background())

// Validator validates the requests to report service and generates job request for RunReport service
type Validator struct {
	reportConfigDatastore reportConfigDS.DataStore
	snapshotDatastore     snapshotDS.DataStore
	collectionDatastore   collectionDS.DataStore
	notifierDatastore     notifierDS.DataStore
}

// New Validator instance
func New(reportConfigDatastore reportConfigDS.DataStore, reportSnapshotDatastore snapshotDS.DataStore,
	collectionDatastore collectionDS.DataStore, notifierDatastore notifierDS.DataStore) *Validator {
	return &Validator{
		reportConfigDatastore: reportConfigDatastore,
		snapshotDatastore:     reportSnapshotDatastore,
		collectionDatastore:   collectionDatastore,
		notifierDatastore:     notifierDatastore,
	}
}

// ValidateReportConfiguration validates the given report configuration object
func (v *Validator) ValidateReportConfiguration(config *apiV2.ReportConfiguration) error {
	if config.GetName() == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration name is empty")
	}

	if err := v.validateSchedule(config); err != nil {
		return err
	}
	if err := v.validateNotifiers(config); err != nil {
		return err
	}
	if err := v.validateResourceScope(config); err != nil {
		return err
	}
	if err := v.validateReportFilters(config); err != nil {
		return err
	}

	return nil
}

func (v *Validator) validateSchedule(config *apiV2.ReportConfiguration) error {
	schedule := config.GetSchedule()
	if schedule == nil {
		return nil
	}
	switch schedule.GetIntervalType() {
	case apiV2.ReportSchedule_UNSET:
		return errors.Wrap(errox.InvalidArgs, "Report configuration schedule must be one of WEEKLY or MONTHLY")
	case apiV2.ReportSchedule_WEEKLY:
		if schedule.GetDaysOfWeek() == nil || len(schedule.GetDaysOfWeek().GetDays()) == 0 {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify days of week for weekly schedule")
		}
		for _, day := range schedule.GetDaysOfWeek().GetDays() {
			if day < 0 || day > 6 {
				return errors.Wrap(errox.InvalidArgs, "Invalid schedule: Days of the week can be Sunday (0) - Saturday(6)")
			}
		}
	case apiV2.ReportSchedule_MONTHLY:
		if schedule.GetDaysOfMonth() == nil || len(schedule.GetDaysOfMonth().GetDays()) == 0 {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify days of the month for monthly schedule")
		}
		for _, day := range schedule.GetDaysOfMonth().GetDays() {
			if day != 1 && day != 15 {
				return errors.Wrap(errox.InvalidArgs, "Reports can be sent out only 1st or 15th day of the month")
			}
		}
	}
	return nil
}

func (v *Validator) validateNotifiers(config *apiV2.ReportConfiguration) error {
	notifiers := config.GetNotifiers()
	if len(notifiers) == 0 {
		if config.GetSchedule() != nil {
			return errors.Wrap(errox.InvalidArgs, "Report configurations with a schedule must specify a notifier.")
		}
		return nil
	}
	for _, notifier := range notifiers {
		if notifier.GetEmailConfig() == nil {
			return errors.Wrap(errox.InvalidArgs, "Notifier must specify an email notifier configuration")
		}
		if err := v.validateEmailConfig(notifier.GetEmailConfig()); err != nil {
			return err
		}
	}
	return nil
}

func (v *Validator) validateEmailConfig(emailConfig *apiV2.EmailNotifierConfiguration) error {
	if emailConfig.GetNotifierId() == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid email notifier")
	}
	if len(emailConfig.GetMailingLists()) == 0 {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify at least one email recipient to send the report to")
	}
	if len(emailConfig.GetCustomSubject()) > CustomEmailSubjectMaxLen {
		return errors.Wrapf(errox.InvalidArgs, "Custom email subject must be fewer than %d characters", CustomEmailSubjectMaxLen)
	}
	if len(emailConfig.GetCustomBody()) > CustomEmailBodyMaxLen {
		return errors.Wrapf(errox.InvalidArgs, "Custom email body must be fewer than than %d characters", CustomEmailBodyMaxLen)
	}

	errorList := errorhelpers.NewErrorList("Invalid email addresses in mailing list: ")
	for _, addr := range emailConfig.GetMailingLists() {
		if _, err := mail.ParseAddress(addr); err != nil {
			errorList.AddError(errors.Wrapf(errox.InvalidArgs, "Invalid email recipient address: %s", addr))
		}
	}
	if !errorList.Empty() {
		return errorList.ToError()
	}

	// Use allAccessCtx since report creator/updater might not have permissions for integrationSAC
	exists, err := v.notifierDatastore.Exists(allAccessCtx, emailConfig.GetNotifierId())
	if err != nil {
		return errors.Errorf("Error looking up attached notifier, Notifier ID: %s, Error: %s", emailConfig.GetNotifierId(), err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Notifier with ID %s not found.", emailConfig.GetNotifierId())
	}
	return nil
}

func (v *Validator) validateResourceScope(config *apiV2.ReportConfiguration) error {
	if config.GetResourceScope() == nil || config.GetResourceScope().GetCollectionScope() == nil {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid resource scope")
	}
	collectionID := config.GetResourceScope().GetCollectionScope().GetCollectionId()

	if collectionID == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid collection ID")
	}

	// Use allAccessCtx since report creator/updater might not have permissions for workflowAdministrationSAC
	exists, err := v.collectionDatastore.Exists(allAccessCtx, collectionID)
	if err != nil {
		return errors.Errorf("Error trying to lookup attached collection, Collection: %s, Error: %s", collectionID, err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Collection %s not found.", collectionID)
	}

	return nil
}

func (v *Validator) validateReportFilters(config *apiV2.ReportConfiguration) error {
	filters := config.GetVulnReportFilters()
	if filters == nil {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must include Vulnerability report filters")
	}

	if len(filters.GetImageTypes()) == 0 {
		return errors.Wrap(errox.InvalidArgs, "Vulnerability report filters should specify which image types to scan for CVEs. "+
			"The valid options are 'DEPLOYED' and 'WATCHED'.")
	}

	if filters.GetCvesSince() == nil {
		return errors.Wrap(errox.InvalidArgs, "Vulnerability report filters must specify how far back in time to look for CVEs. "+
			"The valid options are 'sinceLastSentScheduledReport', 'allVuln', and 'startDate'")
	}
	return nil
}

// ValidateAndGenerateReportRequest validates the report configuration for which report is requested and generates a report request
func (v *Validator) ValidateAndGenerateReportRequest(
	configID string,
	notificationMethod storage.ReportStatus_NotificationMethod,
	requestType storage.ReportStatus_RunMethod,
	requesterID authn.Identity,
) (*reportGen.ReportRequest, error) {
	config, found, err := v.reportConfigDatastore.GetReportConfiguration(allAccessCtx, configID)
	if err != nil {
		return nil, errors.Wrapf(err, "Error finding report configuration %s", configID)
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Report configuration id not found %s", configID)
	}
	if !common.IsV2ReportConfig(config) {
		return nil, errors.Wrap(errox.InvalidArgs, "report configuration does not belong to reporting version 2.0")
	}

	if notificationMethod == storage.ReportStatus_EMAIL && len(config.GetNotifiers()) == 0 {
		return nil, errors.Wrap(errox.InvalidArgs,
			"Email request sent for a report configuration that does not have any email notifiers configured")
	}

	collection, found, err := v.collectionDatastore.Get(allAccessCtx, config.GetResourceScope().GetCollectionId())
	if err != nil {
		return nil, errors.Wrapf(err, "Error finding collection ID '%s'", config.GetResourceScope().GetCollectionId())
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Collection ID '%s' not found", config.GetResourceScope().GetCollectionId())
	}

	notifierIDs := make([]string, 0, len(config.GetNotifiers()))
	for _, notifierConf := range config.GetNotifiers() {
		notifierIDs = append(notifierIDs, notifierConf.GetId())
	}
	protoNotifiers, err := v.notifierDatastore.GetManyNotifiers(allAccessCtx, notifierIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Error finding attached notifiers")
	}
	if len(protoNotifiers) != len(notifierIDs) {
		return nil, errors.Wrap(errox.NotFound, "Some of the attached notifiers not found")
	}

	return &reportGen.ReportRequest{
		Collection:     collection,
		ReportSnapshot: generateReportSnapshot(config, collection, protoNotifiers, notificationMethod, requestType, requesterID),
	}, nil
}

// ValidateCancelReportRequest validates if the given requester can cancel the report job with job ID = reportID.
func (v *Validator) ValidateCancelReportRequest(reportID string, requester *storage.SlimUser) error {
	snapshot, found, err := v.snapshotDatastore.Get(allAccessCtx, reportID)
	if err != nil {
		return errors.Wrapf(err, "Error finding report snapshot with job ID '%s'.", reportID)
	}
	if !found {
		return errors.Wrapf(errox.NotFound, "Report snapshot with job ID '%s' does not exist", reportID)
	}

	switch snapshot.GetReportStatus().GetRunState() {
	case storage.ReportStatus_DELIVERED, storage.ReportStatus_GENERATED, storage.ReportStatus_FAILURE:
		return errors.Wrapf(errox.InvalidArgs, "Cannot cancel. Report job ID '%s' has already completed execution.", reportID)
	case storage.ReportStatus_PREPARING:
		return errors.Wrapf(errox.InvalidArgs, "Cannot cancel. Report job ID '%s' is currently being prepared.", reportID)
	}

	if requester.GetId() != snapshot.GetRequester().GetId() {
		return errors.Wrap(errox.NotAuthorized, "Report job cannot be cancelled by a user who did not request the report.")
	}
	return nil
}

func generateReportSnapshot(
	config *storage.ReportConfiguration,
	collection *storage.ResourceCollection,
	protoNotifiers []*storage.Notifier,
	notificationMethod storage.ReportStatus_NotificationMethod,
	requestType storage.ReportStatus_RunMethod,
	requesterID authn.Identity,
) *storage.ReportSnapshot {
	snapshot := &storage.ReportSnapshot{
		ReportConfigurationId: config.GetId(),
		Name:                  config.GetName(),
		Description:           config.GetDescription(),
		Type:                  storage.ReportSnapshot_VULNERABILITY,
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
	}

	reportFilters := config.GetVulnReportFilters().Clone()
	var requester *storage.SlimUser
	switch requestType {
	case storage.ReportStatus_ON_DEMAND:
		reportFilters.AccessScopeRules = common.ExtractAccessScopeRules(requesterID)
		requester = &storage.SlimUser{
			Id:   requesterID.UID(),
			Name: stringutils.FirstNonEmpty(requesterID.FullName(), requesterID.FriendlyName()),
		}
	case storage.ReportStatus_SCHEDULED:
		requester = config.GetCreator()
	}
	snapshot.Requester = requester
	snapshot.Filter = &storage.ReportSnapshot_VulnReportFilters{
		VulnReportFilters: reportFilters,
	}

	notifierSnaps := make([]*storage.NotifierSnapshot, 0, len(config.GetNotifiers()))

	for i, notifierConf := range config.GetNotifiers() {
		notifierSnaps = append(notifierSnaps, &storage.NotifierSnapshot{
			NotifierConfig: &storage.NotifierSnapshot_EmailConfig{
				EmailConfig: func() *storage.EmailNotifierConfiguration {
					cfg := notifierConf.GetEmailConfig()
					cfg.NotifierId = notifierConf.GetId()
					return cfg
				}(),
			},
			NotifierName: protoNotifiers[i].GetName(),
		})
	}
	snapshot.Notifiers = notifierSnaps
	return snapshot
}
