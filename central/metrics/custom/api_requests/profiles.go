package api_requests

import (
	"github.com/stackrox/rox/pkg/clientprofile"
)

const unknownProfile = "unknown"

// builtinProfiles maps profile name to its matching criteria. A request can
// match multiple profiles.
var builtinProfiles = map[string]clientprofile.RuleSet{
	"servicenow": {{
		Headers: clientprofile.GlobMap{
			"User-Agent": "*ServiceNow*",
			"Rh-*":       clientprofile.NoHeaderOrAnyValue,
		},
	}},
	"splunk_ta": {
		clientprofile.PathPattern("/api/splunk/ta/*"),
	},
	"roxctl": {{
		Headers: clientprofile.GlobMap{
			"User-Agent": "roxctl/*",
			"Rh-*":       clientprofile.NoHeaderOrAnyValue,
		},
	}},
	"central": {
		clientprofile.HeaderPattern("User-Agent", "Rox Central/*"),
	},
	"sensor": {
		clientprofile.HeaderPattern("User-Agent", "Rox Sensor/*"),
	},
}

func init() {
	for _, criteria := range builtinProfiles {
		if err := criteria.Compile(); err != nil {
			panic(err)
		}
	}
}
