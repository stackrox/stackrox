package all

import (
	_ "github.com/stackrox/rox/pkg/notifiers/generic"
	_ "github.com/stackrox/rox/pkg/notifiers/syslog"
	_ "github.com/stackrox/rox/sensor/admission-control/notifiers/jira"
)
