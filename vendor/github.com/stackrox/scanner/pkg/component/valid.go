package component

import (
	"github.com/stackrox/rox/pkg/stringutils"
)

// Valid returns whether the component is well-formed.
// Analyzers MUST ensure that all components they return pass
// this function.
// Downstream subsystems can rely on this being the case.
func Valid(c *Component) bool {
	return stringutils.AllNotEmpty(c.Name, c.Version) && c.SourceType != UnsetSourceType
}

// FilterToOnlyValid filters the given slice of components to only
// valid ones. It mutates the same underlying array instead of allocating
// a new one.
func FilterToOnlyValid(components []*Component) []*Component {
	filtered := components[:0]
	for _, c := range components {
		if Valid(c) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
