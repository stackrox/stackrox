package all

import (
	_ "github.com/stackrox/rox/pkg/notifiers/cscc"   // Import the CSCC package
	_ "github.com/stackrox/rox/pkg/notifiers/email"  // Import the email package
	_ "github.com/stackrox/rox/pkg/notifiers/jira"   // Import the Jira package
	_ "github.com/stackrox/rox/pkg/notifiers/slack"  // Import the Slack package
	_ "github.com/stackrox/rox/pkg/notifiers/splunk" // Import the Splunk package
)
