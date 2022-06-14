package standards

import (
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/pkg/set"
)

func gatherDataDependencies(checks []framework.Check) set.StringSet {
	result := set.NewStringSet()
	for _, check := range checks {
		for _, dep := range check.DataDependencies() {
			result.Add(dep)
		}
	}
	return result
}
