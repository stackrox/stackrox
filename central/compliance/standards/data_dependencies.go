package standards

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/set"
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
