package charts

import (
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/roxctl/defaults"
	"github.com/stackrox/rox/pkg/version"
)

// MetaValues are the values to be passed to the StackRox chart templates.
type MetaValues map[string]interface{}

// ChartRepo contains information about where the Helm charts are published.
type ChartRepo struct {
	URL string
}

// DefaultMetaValues are the default meta values for rendering the StackRox charts in production.
func DefaultMetaValues() MetaValues {
	metaValues := map[string]interface{}{
		"Versions":          version.GetAllVersions(),
		"MainRegistry":      defaults.MainImageRegistry(),
		"CollectorRegistry": defaults.CollectorImageRegistry(),
		"RenderMode":        "",
		"ChartRepo": ChartRepo{
			URL: "https://charts.stackrox.io",
		},
	}

	featureFlagVals := make(map[string]interface{})
	for _, feature := range features.Flags {
		featureFlagVals[feature.EnvVar()] = feature.Enabled()
	}
	metaValues["FeatureFlags"] = featureFlagVals

	return metaValues
}

// RHACSMetaValues are the meta values for rendering the StackRox charts in RHACS flavor.
func RHACSMetaValues() MetaValues {
	metaValues := map[string]interface{}{
		"Versions":          version.GetAllVersions(),
		"MainRegistry":      "registry.redhat.io/rh-acs",
		"CollectorRegistry": "registry.redhat.io/rh-acs",
		"RenderMode":        "",
		"ChartRepo": ChartRepo{
			URL: "http://mirror.openshift.com/pub/rhacs/charts",
		},
	}

	featureFlagVals := make(map[string]interface{})
	for _, feature := range features.Flags {
		featureFlagVals[feature.EnvVar()] = feature.Enabled()
	}
	metaValues["FeatureFlags"] = featureFlagVals

	return metaValues
}
