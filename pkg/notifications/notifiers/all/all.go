package all

import (
	// Import the slack package
	_ "bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers/slack"
	// Import the email package
	_ "bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers/email"
	// Import the Jira package
	_ "bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers/jira"
)
