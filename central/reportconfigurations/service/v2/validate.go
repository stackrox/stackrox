package v2

import (
	"context"
	"net/mail"

	"github.com/pkg/errors"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
)

// Use this context only to
// 1) check if notifiers and collection attached to report config exist
// 2) Populating notifier and collection names before returning v2.ReportConfiguration response
var allAccessCtx = sac.WithAllAccess(context.Background())

// ValidateReportConfiguration validates the given report configuration object
func (s *serviceImpl) ValidateReportConfiguration(config *apiV2.ReportConfiguration) error {
	if config.GetName() == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration name is empty")
	}

	if err := s.validateSchedule(config); err != nil {
		return err
	}
	if err := s.validateNotifiers(config); err != nil {
		return err
	}
	if err := s.validateResourceScope(config); err != nil {
		return err
	}
	if err := s.validateReportFilters(config); err != nil {
		return err
	}

	return nil
}

func (s *serviceImpl) validateSchedule(config *apiV2.ReportConfiguration) error {
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
			if day < 1 || day > 7 {
				return errors.Wrap(errox.InvalidArgs, "Invalid schedule: Days of the week can be Sunday (1) - Saturday(7)")
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

func (s *serviceImpl) validateNotifiers(config *apiV2.ReportConfiguration) error {
	notifiers := config.GetNotifiers()
	if len(notifiers) == 0 {
		return nil
	}
	for _, notifier := range notifiers {
		if notifier.GetEmailConfig() == nil {
			return errors.Wrap(errox.InvalidArgs, "Notifier must specify an email notifier configuration")
		}
		if err := s.validateEmailConfig(notifier.GetEmailConfig()); err != nil {
			return err
		}
	}
	return nil
}

func (s *serviceImpl) validateEmailConfig(emailConfig *apiV2.EmailNotifierConfiguration) error {
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

	// Use allAccessCtx since report creator/updater might not have permissions for integrationSAC
	exists, err := s.notifierDatastore.Exists(allAccessCtx, emailConfig.GetNotifierId())
	if err != nil {
		return errors.Errorf("Error looking up attached notifier, Notifier ID: %s, Error: %s", emailConfig.GetNotifierId(), err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Notifier with ID %s not found.", emailConfig.GetNotifierId())
	}
	return nil
}

func (s *serviceImpl) validateResourceScope(config *apiV2.ReportConfiguration) error {
	if config.GetResourceScope() == nil || config.GetResourceScope().GetCollectionScope() == nil {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid resource scope")
	}
	collectionID := config.GetResourceScope().GetCollectionScope().GetCollectionId()

	if collectionID == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid collection ID")
	}

	// Use allAccessCtx since report creator/updater might not have permissions for workflowAdministrationSAC
	exists, err := s.collectionDatastore.Exists(allAccessCtx, collectionID)
	if err != nil {
		return errors.Errorf("Error trying to lookup attached collection, Collection: %s, Error: %s", collectionID, err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Collection %s not found.", collectionID)
	}

	return nil
}

func (s *serviceImpl) validateReportFilters(config *apiV2.ReportConfiguration) error {
	if config.GetVulnReportFilters() == nil {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must include Vulnerability report filters")
	}
	if config.GetVulnReportFilters().GetCvesSince() == nil {
		return errors.Wrap(errox.InvalidArgs, "Vulnerability report filters must specify how far back in time to look for CVEs. "+
			"The valid options are 'since last successful report', 'all CVEs', and 'since a custom timestamp'")
	}
	return nil
}
