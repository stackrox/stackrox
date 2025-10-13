package service

import (
	"context"
	"net/mail"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/reports/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
)

// validateReportConfiguration validates the given report configuration object
func (s *serviceImpl) validateReportConfiguration(ctx context.Context, config *storage.ReportConfiguration) error {
	if config.GetName() == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration name is empty")
	}

	if err := s.validateSchedule(config); err != nil {
		return err
	}
	if err := s.validateNotifier(ctx, config); err != nil {
		return err
	}
	if err := s.validateResourceScope(ctx, config); err != nil {
		return err
	}

	return nil
}

func (s *serviceImpl) validateSchedule(config *storage.ReportConfiguration) error {
	schedule := config.GetSchedule()
	if schedule == nil {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must have a schedule")
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
				return errors.Wrap(errox.InvalidArgs, "Reports can be sent out only 1st or 15th day of the month")
			}
		}
	}
	return nil
}

func (s *serviceImpl) validateNotifier(ctx context.Context, config *storage.ReportConfiguration) error {
	if config.GetEmailConfig() == nil {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify an email notifier configuration")
	}
	return s.validateEmailConfig(ctx, config.GetEmailConfig())
}

func (s *serviceImpl) validateEmailConfig(ctx context.Context, emailConfig *storage.EmailNotifierConfiguration) error {
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

	exists, err := s.notifierDatastore.Exists(ctx, emailConfig.GetNotifierId())
	if err != nil {
		return errors.Errorf("Error looking up attached notifier, Notifier: %s, Error: %s", emailConfig.GetNotifierId(), err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Notifier %s not found.", emailConfig.GetNotifierId())
	}
	return nil
}

func (s *serviceImpl) validateResourceScope(ctx context.Context, config *storage.ReportConfiguration) error {
	if !common.IsV1ReportConfig(config) {
		return errors.Wrap(errox.InvalidArgs, "Report configuration belonging to reporting version 1.0 should not set the 'resourceScope' field."+
			"Instead, set the 'scopeId' field to the desired collection ID.")
	}
	if config.GetScopeId() == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid collection ID in the 'scopeId' field")
	}

	exists, err := s.collectionDatastore.Exists(ctx, config.GetScopeId())
	if err != nil {
		return errors.Errorf("Error trying to lookup attached collection, Collection: %s, Error: %s", config.GetScopeId(), err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Collection %s not found.", config.GetScopeId())
	}
	return nil
}
