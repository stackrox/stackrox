package all

import (
	// Import the CSCC package
	_ "github.com/stackrox/rox/pkg/notifiers/cscc"
	// Import the email package
	_ "github.com/stackrox/rox/pkg/notifiers/email"
	// Import the Jira package
	_ "github.com/stackrox/rox/pkg/notifiers/jira"
	// Import the Slack package
	_ "github.com/stackrox/rox/pkg/notifiers/slack"
)
