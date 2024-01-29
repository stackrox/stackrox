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
	pkgNotifiers "github.com/stackrox/rox/pkg/notifiers"
)

// ValidateNotifierConfig validates notifier configuration based on the given notifier's type
func ValidateNotifierConfig(notifier *storage.Notifier, validateSecret bool) error {
	switch notifier.GetType() {
	case pkgNotifiers.AWSSecurityHubType:
		if err := awssh.Validate(notifier.GetAwsSecurityHub(), validateSecret); err != nil {
			return errors.Wrap(err, "failed to validate AWS SecurityHub config")
		}
	case pkgNotifiers.CSCCType:
		if err := cscc.Validate(notifier.GetCscc(), validateSecret); err != nil {
			return errors.Wrap(err, "failed to validate CSCC config")
		}
	case pkgNotifiers.JiraType:
		if err := jira.Validate(notifier.GetJira(), validateSecret); err != nil {
			return errors.Wrap(err, "failed to validate Jira config")
		}
	case pkgNotifiers.EmailType:
		if err := email.Validate(notifier.GetEmail(), validateSecret); err != nil {
			return errors.Wrap(err, "failed to validate Email config")
		}
	case pkgNotifiers.GenericType:
		if err := generic.Validate(notifier.GetGeneric(), validateSecret); err != nil {
			return errors.Wrap(err, "failed to validate Generic config")
		}
	case pkgNotifiers.PagerDutyType:
		if err := pagerduty.Validate(notifier.GetPagerduty(), validateSecret); err != nil {
			return errors.Wrap(err, "failed to validate PagerDuty config")
		}
	case pkgNotifiers.SplunkType:
		if err := splunk.Validate(notifier.GetSplunk(), validateSecret); err != nil {
			return errors.Wrap(err, "failed to validate Splunk config")
		}
	}
	return nil
}
