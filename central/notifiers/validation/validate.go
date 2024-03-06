package validation

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers/awssh"
	"github.com/stackrox/rox/central/notifiers/cscc"
	"github.com/stackrox/rox/central/notifiers/email"
	"github.com/stackrox/rox/central/notifiers/generic"
	"github.com/stackrox/rox/central/notifiers/jira"
	"github.com/stackrox/rox/central/notifiers/pagerduty"
	"github.com/stackrox/rox/central/notifiers/splunk"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/endpoints"
	"github.com/stackrox/rox/pkg/errorhelpers"
	pkgNotifiers "github.com/stackrox/rox/pkg/notifiers"
)

// ValidateNotifierConfig validates notifier configuration based on the given notifier's type
func ValidateNotifierConfig(notifier *storage.Notifier, validateSecret bool) error {
	if notifier == nil {
		return errors.New("empty notifier")
	}
	errorList := errorhelpers.NewErrorList("Validation")
	if notifier.GetName() == "" {
		errorList.AddString("notifier name must be defined")
	}
	if notifier.GetType() == "" {
		errorList.AddString("notifier type must be defined")
	}
	if notifier.GetUiEndpoint() == "" {
		errorList.AddString("notifier UI endpoint must be defined")
	}
	if err := endpoints.ValidateEndpoints(notifier.Config); err != nil {
		errorList.AddWrap(err, "invalid endpoint")
	}
	switch notifier.GetType() {
	case pkgNotifiers.AWSSecurityHubType:
		if err := awssh.Validate(notifier.GetAwsSecurityHub(), validateSecret); err != nil {
			errorList.AddWrap(err, "failed to validate AWS SecurityHub config")
			return errorList.ToError()
		}
	case pkgNotifiers.CSCCType:
		if err := cscc.Validate(notifier.GetCscc(), validateSecret); err != nil {
			errorList.AddWrap(err, "failed to validate CSCC config")
			return errorList.ToError()
		}
	case pkgNotifiers.JiraType:
		if err := jira.Validate(notifier.GetJira(), validateSecret); err != nil {
			errorList.AddWrap(err, "failed to validate Jira config")
			return errorList.ToError()
		}
	case pkgNotifiers.EmailType:
		if err := email.Validate(notifier.GetEmail(), validateSecret); err != nil {
			errorList.AddWrap(err, "failed to validate Email config")
			return errorList.ToError()
		}
	case pkgNotifiers.GenericType:
		if err := generic.Validate(notifier.GetGeneric(), validateSecret); err != nil {
			errorList.AddWrap(err, "failed to validate Generic config")
			return errorList.ToError()
		}
	case pkgNotifiers.PagerDutyType:
		if err := pagerduty.Validate(notifier.GetPagerduty(), validateSecret); err != nil {
			errorList.AddWrap(err, "failed to validate PagerDuty config")
			return errorList.ToError()
		}
	case pkgNotifiers.SplunkType:
		if err := splunk.Validate(notifier.GetSplunk(), validateSecret); err != nil {
			errorList.AddWrap(err, "failed to validate Splunk config")
			return errorList.ToError()
		}
	}
	return errorList.ToError()
}
