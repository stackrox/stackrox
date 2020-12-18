package charts

import (
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/roxctl/defaults"
	"github.com/stackrox/rox/pkg/version"
)

// MetaValues are the values to be passed to the StackRox chart templates.
type MetaValues map[string]interface{}

// DefaultMetaValues are the default meta values for rendering the StackRox charts in production.
func DefaultMetaValues() MetaValues {
	metaValues := map[string]interface{}{
		"Versions":          version.GetAllVersions(),
		"MainRegistry":      defaults.MainImageRegistry(),
		"CollectorRegistry": defaults.CollectorImageRegistry(),
		"RenderMode":        "",
	}

	featureFlagVals := make(map[string]interface{})
	for _, feature := range features.Flags {
		featureFlagVals[feature.EnvVar()] = feature.Enabled()
	}
	metaValues["FeatureFlags"] = featureFlagVals

	return metaValues
}
