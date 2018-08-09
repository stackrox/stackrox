package all

import (
	// Import the Slack package
	_ "github.com/stackrox/rox/pkg/notifications/notifiers/slack"
	// Import the email package
	_ "github.com/stackrox/rox/pkg/notifications/notifiers/email"
	// Import the Jira package
	_ "github.com/stackrox/rox/pkg/notifications/notifiers/jira"
	// Import the CSCC package
	_ "github.com/stackrox/rox/pkg/notifications/notifiers/cscc"
)
