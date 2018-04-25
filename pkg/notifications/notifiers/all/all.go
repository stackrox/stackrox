package all

import (
	// Import the Slack package
	_ "bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers/slack"
	// Import the email package
	_ "bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers/email"
	// Import the Jira package
	_ "bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers/jira"
	// Import the CSCC package
	_ "bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers/cscc"
)
