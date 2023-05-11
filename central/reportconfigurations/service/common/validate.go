package common

import (
	"context"
	"net/mail"

	"github.com/pkg/errors"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	accessScopeDS "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
)

// Validator is used to validate storage.ReportConfiguration instances
type Validator struct {
	accessScopeDatastore accessScopeDS.DataStore
	collectionDatastore  collectionDS.DataStore
	notifierDatastore    notifierDS.DataStore
}

// NewValidator returns a new validator
func NewValidator(accessScopeDatastore accessScopeDS.DataStore,
	collectionDatastore collectionDS.DataStore,
	notifierDatastore notifierDS.DataStore) *Validator {
	return &Validator{
		accessScopeDatastore: accessScopeDatastore,
		collectionDatastore:  collectionDatastore,
		notifierDatastore:    notifierDatastore,
	}
}

// ValidateReportConfiguration validates the given report configuration object
func (validator *Validator) ValidateReportConfiguration(ctx context.Context, config *storage.ReportConfiguration) error {
	if config.GetName() == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration name empty")
	}

	if err := validator.validateSchedule(config); err != nil {
		return err
	}
	if err := validator.validateNotifiers(ctx, config); err != nil {
		return err
	}
	if err := validator.validateResourceScope(ctx, config); err != nil {
		return err
	}
	if err := validator.validateReportFilters(config); err != nil {
		return err
	}

	return nil
}

func (validator *Validator) validateSchedule(config *storage.ReportConfiguration) error {
	schedule := config.GetSchedule()
	if schedule == nil {
		if !features.VulnMgmtReportingEnhancements.Enabled() {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must have a schedule")
		}
		return nil
	}
	switch schedule.GetIntervalType() {
	case storage.Schedule_UNSET:
	case storage.Schedule_DAILY:
		return errors.Wrap(errox.InvalidArgs, "Report configuration must have a valid schedule type")
	case storage.Schedule_WEEKLY:
		if schedule.GetDaysOfWeek() == nil || len(schedule.GetDaysOfWeek().GetDays()) == 0 {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify days of week for the schedule")
		}
		for _, day := range schedule.GetDaysOfWeek().GetDays() {
			if day < 0 || day > 6 {
				return errors.Wrap(errox.InvalidArgs, "Invalid schedule: Days of the week can be Sunday (0) - Saturday(6)")
			}
		}
	case storage.Schedule_MONTHLY:
		if schedule.GetDaysOfMonth() == nil || len(schedule.GetDaysOfMonth().GetDays()) == 0 {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify days of the month for the schedule")
		}
		for _, day := range schedule.GetDaysOfMonth().GetDays() {
			if day != 1 && day != 15 {
				return errors.Wrap(errox.InvalidArgs, "Reports can be sent out only 1st or 15th of the month")
			}
		}
	}
	return nil
}

func (validator *Validator) validateNotifiers(ctx context.Context, config *storage.ReportConfiguration) error {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		if config.GetEmailConfig() == nil {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify an email notifier configuration")
		}
		return validator.validateEmailConfig(ctx, config.GetEmailConfig())
	}

	notifiers := config.GetNotifiers()
	if len(notifiers) == 0 {
		return nil
	}
	for _, notifier := range notifiers {
		if notifier.GetEmailConfig() == nil {
			return errors.Wrap(errox.InvalidArgs, "Notifier must specify an email notifier configuration")
		}
		if err := validator.validateEmailConfig(ctx, notifier.GetEmailConfig()); err != nil {
			return err
		}
	}
	return nil
}

func (validator *Validator) validateEmailConfig(ctx context.Context, emailConfig *storage.EmailNotifierConfiguration) error {
	if emailConfig.GetNotifierId() == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid email notifier")
	}
	if len(emailConfig.GetMailingLists()) == 0 {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify at least one email recipient to send the report to")
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

	exists, err := validator.notifierDatastore.Exists(ctx, emailConfig.GetNotifierId())
	if err != nil {
		return errors.Errorf("Error trying to lookup attached notofier, Notifier: %s, Error: %s", emailConfig.GetNotifierId(), err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Notifier %s not found.", emailConfig.GetNotifierId())
	}
	return nil
}

func (validator *Validator) validateResourceScope(ctx context.Context, config *storage.ReportConfiguration) error {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		if config.GetScopeId() == "" {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid scope ID")
		}
		exists, err := validator.accessScopeDatastore.AccessScopeExists(ctx, config.GetScopeId())
		if err != nil {
			return errors.Errorf("Error trying to lookup attached scope, Scope: %s, Error: %s", config.GetScopeId(), err)
		}
		if !exists {
			return errors.Wrapf(errox.NotFound, "Access scope %s not found.", config.GetScopeId())
		}
		return nil
	}

	var collectionID string
	if features.VulnMgmtReportingEnhancements.Enabled() {
		if config.GetResourceScope() == nil || config.GetResourceScope().GetCollectionId() == "" {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid resource scope")
		}
		collectionID = config.GetResourceScope().GetCollectionId()
	} else {
		if config.GetScopeId() == "" {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid collection ID in the 'scopeId' field")
		}
		collectionID = config.GetScopeId()
	}

	exists, err := validator.collectionDatastore.Exists(ctx, collectionID)
	if err != nil {
		return errors.Errorf("Error trying to lookup attached collection, Collection: %s, Error: %s", collectionID, err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Collection %s not found.", collectionID)
	}
	return nil
}

func (validator *Validator) validateReportFilters(config *storage.ReportConfiguration) error {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		return nil
	}
	if config.GetVulnReportFilters() == nil {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must include Vulnerability report filters")
	}
	if config.GetVulnReportFilters().GetCvesSince() == nil {
		return errors.Wrap(errox.InvalidArgs, "Vulnerability report filters must specify how far back in time to look for CVEs "+
			"The valid options are since last successful report, all CVEs or since a custom timestamp")
	}
	return nil
}
