package charts

import (
	"github.com/stackrox/rox/pkg/roxctl/defaults"
	"github.com/stackrox/rox/pkg/version"
)

// MetaValues are the values to be passed to the central-services chart template.
type MetaValues struct {
	Versions     version.Versions
	RenderMode   string
	MainRegistry string
}

// DefaultMetaValues are the default meta values for rendering the chart in production.
func DefaultMetaValues() MetaValues {
	return MetaValues{
		Versions:     version.GetAllVersions(),
		MainRegistry: defaults.MainImageRegistry(),
	}
}
