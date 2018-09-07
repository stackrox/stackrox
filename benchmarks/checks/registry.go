package checks

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
)

// Registry is a map of check name to check object
var Registry = map[string]utils.Check{}

// AddToRegistry is a method that takes in a series of checks and adds them to the registry
func AddToRegistry(checks ...utils.Check) {
	for _, check := range checks {
		Registry[check.Definition().Name] = check
	}
}
