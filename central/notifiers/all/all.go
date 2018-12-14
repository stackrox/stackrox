package all

import (
	_ "github.com/stackrox/rox/central/notifiers/cscc"   // Import the CSCC package
	_ "github.com/stackrox/rox/central/notifiers/email"  // Import the email package
	_ "github.com/stackrox/rox/central/notifiers/jira"   // Import the Jira package
	_ "github.com/stackrox/rox/central/notifiers/slack"  // Import the Slack package
	_ "github.com/stackrox/rox/central/notifiers/splunk" // Import the Splunk package
)
