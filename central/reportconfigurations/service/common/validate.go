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
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
)

// ValidateReportConfiguration validates the given report configuration object
func ValidateReportConfiguration(ctx context.Context, config *storage.ReportConfiguration,
	accessScopeDatastore accessScopeDS.DataStore, collectionDatastore collectionDS.DataStore,
	notifierDatastore notifierDS.DataStore) error {
	if config.GetName() == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration name empty")
	}

	if err := validateSchedule(config); err != nil {
		return err
	}
	if err := validateNotifiers(ctx, config, notifierDatastore); err != nil {
		return err
	}
	if err := validateResourceScope(ctx, config, collectionDatastore, accessScopeDatastore); err != nil {
		return err
	}
	if err := validateReportFilters(config); err != nil {
		return err
	}

	return nil
}

func validateSchedule(config *storage.ReportConfiguration) error {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		if config.GetSchedule() == nil {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must have a schedule")
		}
	}
	schedule := config.GetSchedule()
	if schedule == nil {
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

func validateNotifiers(ctx context.Context, config *storage.ReportConfiguration,
	notifierDatastore notifierDS.DataStore) error {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		if config.GetEmailConfig() == nil {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify an email notifier configuration")
		}
		return validateEmailConfig(ctx, config.GetEmailConfig(), notifierDatastore)
	}

	notifiers := config.GetNotifiers()
	if len(notifiers) == 0 {
		return nil
	}
	for _, notifier := range notifiers {
		if notifier.GetEmailConfig() == nil {
			return errors.Wrap(errox.InvalidArgs, "Notifier must specify an email notifier configuration")
		}
		if err := validateEmailConfig(ctx, notifier.GetEmailConfig(), notifierDatastore); err != nil {
			return err
		}
	}
	return nil
}

func validateEmailConfig(ctx context.Context, emailConfig *storage.EmailNotifierConfiguration,
	notifierDatastore notifierDS.DataStore) error {
	if emailConfig.GetNotifierId() == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid email notifier")
	}
	if len(emailConfig.GetMailingLists()) == 0 {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify one more recipients to send the report to")
	}

	for _, addr := range emailConfig.GetMailingLists() {
		if _, err := mail.ParseAddress(addr); err != nil {
			return errors.Wrapf(errox.InvalidArgs, "Invalid mailing list address: %s", addr)
		}
	}

	_, found, err := notifierDatastore.GetNotifier(ctx, emailConfig.GetNotifierId())
	if !found || err != nil {
		return errors.Wrapf(errox.NotFound, "Notifier %s not found. Error: %s", emailConfig.GetNotifierId(), err)
	}
	return nil
}

func validateResourceScope(ctx context.Context, config *storage.ReportConfiguration,
	collectionDatastore collectionDS.DataStore, accessScopeDatastore accessScopeDS.DataStore) error {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		if config.GetScopeId() == "" {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid scope ID")
		}
		_, found, err := accessScopeDatastore.GetAccessScope(ctx, config.GetScopeId())
		if !found || err != nil {
			return errors.Wrapf(errox.NotFound, "Access scope %s not found. Error: %s", config.GetScopeId(), err)
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

	_, found, err := collectionDatastore.Get(ctx, collectionID)
	if !found || err != nil {
		return errors.Wrapf(errox.NotFound, "Collection %s not found. Error: %s", collectionID, err)
	}
	return nil
}

func validateReportFilters(config *storage.ReportConfiguration) error {
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
