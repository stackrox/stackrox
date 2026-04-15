package notifiers

import (
	"github.com/stackrox/rox/central/notifiers/acscsemail"
	"github.com/stackrox/rox/central/notifiers/awssh"
	"github.com/stackrox/rox/central/notifiers/cscc"
	"github.com/stackrox/rox/central/notifiers/email"
	genericnotifier "github.com/stackrox/rox/central/notifiers/generic"
	"github.com/stackrox/rox/central/notifiers/jira"
	"github.com/stackrox/rox/central/notifiers/microsoftsentinel"
	"github.com/stackrox/rox/central/notifiers/pagerduty"
	"github.com/stackrox/rox/central/notifiers/slack"
	"github.com/stackrox/rox/central/notifiers/splunk"
	"github.com/stackrox/rox/central/notifiers/sumologic"
	"github.com/stackrox/rox/central/notifiers/syslog"
	"github.com/stackrox/rox/central/notifiers/teams"
)

func Init() {
	acscsemail.RegisterACSCSEmail()
	awssh.RegisterAWSSecurityHub()
	cscc.RegisterCSCC()
	email.RegisterEmail()
	genericnotifier.RegisterGeneric()
	jira.RegisterJira()
	microsoftsentinel.RegisterMicrosoftSentinel()
	pagerduty.RegisterPagerDuty()
	slack.RegisterSlack()
	splunk.RegisterSplunk()
	sumologic.RegisterSumoLogic()
	syslog.RegisterSyslog()
	teams.RegisterTeams()
}
