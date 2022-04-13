package all

import (
	// Register notifiers.
	_ "github.com/stackrox/stackrox/central/notifiers/awssh"
	_ "github.com/stackrox/stackrox/central/notifiers/cscc"
	_ "github.com/stackrox/stackrox/central/notifiers/email"
	_ "github.com/stackrox/stackrox/central/notifiers/generic"
	_ "github.com/stackrox/stackrox/central/notifiers/jira"
	_ "github.com/stackrox/stackrox/central/notifiers/pagerduty"
	_ "github.com/stackrox/stackrox/central/notifiers/slack"
	_ "github.com/stackrox/stackrox/central/notifiers/splunk"
	_ "github.com/stackrox/stackrox/central/notifiers/sumologic"
	_ "github.com/stackrox/stackrox/central/notifiers/syslog"
	_ "github.com/stackrox/stackrox/central/notifiers/teams"
)
